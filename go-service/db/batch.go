package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"orbitstream/models"
)

type BatchProcessor struct {
	pool            *pgxpool.Pool
	batchSize       int
	batchTimeout    time.Duration
	buffer          []models.TelemetryPoint
	bufferMutex     sync.Mutex
	ticker          *time.Ticker
	done            chan bool
	anomalyConfig   AnomalyConfig
	wal             *WAL
	circuitBreaker  *CircuitBreaker
	maxRetries      int
	retryDelay      time.Duration
	maxBufferSize   int
}

type AnomalyConfig struct {
	BatteryMinPercent float64
	StorageMaxMB      float64
	SignalMinDBM      float64
}

func NewBatchProcessor(pool *pgxpool.Pool, batchSize int, batchTimeout time.Duration, anomalyConfig AnomalyConfig) *BatchProcessor {
	return &BatchProcessor{
		pool:           pool,
		batchSize:      batchSize,
		batchTimeout:   batchTimeout,
		buffer:         make([]models.TelemetryPoint, 0, batchSize),
		done:           make(chan bool),
		anomalyConfig:  anomalyConfig,
		maxRetries:     5,             // Default: 5 retry attempts
		retryDelay:     1 * time.Second, // Default: 1 second initial delay
		maxBufferSize:  10000,          // Default: 10K max buffer size
		circuitBreaker: NewCircuitBreaker(3, 30*time.Second), // Open after 3 failures, 30s timeout
	}
}

// SetWAL sets the Write Ahead Log for persistent buffering
func (bp *BatchProcessor) SetWAL(wal *WAL) {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	bp.wal = wal
}

// SetCircuitBreaker sets the circuit breaker for fault tolerance
func (bp *BatchProcessor) SetCircuitBreaker(cb *CircuitBreaker) {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	bp.circuitBreaker = cb
}

// SetRetryConfig configures retry behavior
func (bp *BatchProcessor) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	bp.maxRetries = maxRetries
	bp.retryDelay = retryDelay
}

// SetMaxBufferSize sets the maximum buffer size before rejecting new data
func (bp *BatchProcessor) SetMaxBufferSize(size int) {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	bp.maxBufferSize = size
}

func (bp *BatchProcessor) Add(point models.TelemetryPoint) error {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()

	// Check buffer size limit to prevent unbounded growth
	if len(bp.buffer) >= bp.maxBufferSize {
		log.Printf("WARNING: Buffer full (%d records), rejecting new data", len(bp.buffer))
		return fmt.Errorf("buffer at maximum capacity (%d)", bp.maxBufferSize)
	}

	// Check for anomalies
	point.IsAnomaly = bp.detectAnomaly(point)

	bp.buffer = append(bp.buffer, point)

	// If buffer reaches batch size, trigger immediate flush
	if len(bp.buffer) >= bp.batchSize {
		go bp.flush()
	}

	return nil
}

func (bp *BatchProcessor) Start() {
	bp.ticker = time.NewTicker(bp.batchTimeout)

	for {
		select {
		case <-bp.ticker.C:
			bp.flush()
		case <-bp.done:
			bp.ticker.Stop()
			// Final flush on shutdown
			bp.flush()
			return
		}
	}
}

func (bp *BatchProcessor) Stop() {
	close(bp.done)
}

func (bp *BatchProcessor) flush() {
	bp.bufferMutex.Lock()
	if len(bp.buffer) == 0 {
		bp.bufferMutex.Unlock()
		return
	}

	// Swap buffer to minimize lock time
	batch := make([]models.TelemetryPoint, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = make([]models.TelemetryPoint, 0, bp.batchSize)
	bp.bufferMutex.Unlock()

	// Try to flush with retry logic and WAL fallback
	if err := bp.flushWithRetry(batch); err != nil {
		log.Printf("ERROR: Failed to flush batch after all retries: %v", err)
	}
}

// flushWithRetry attempts to flush the batch with retry logic and exponential backoff
// If all retries fail, it falls back to writing to WAL
func (bp *BatchProcessor) flushWithRetry(batch []models.TelemetryPoint) error {
	for attempt := 0; attempt < bp.maxRetries; attempt++ {
		// Check circuit breaker first
		if bp.circuitBreaker != nil && !bp.circuitBreaker.Allow() {
			log.Printf("Circuit breaker OPEN, writing %d records to WAL", len(batch))
			return bp.flushToWAL(batch)
		}

		// Attempt to insert to database
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		rowsAffected, err := bp.insertBatch(ctx, batch)
		cancel()
		duration := time.Since(startTime)

		if err == nil {
			// Success!
			pointsPerSecond := float64(rowsAffected) / duration.Seconds()
			log.Printf("Flushed %d rows in %v (%.0f points/sec)",
				rowsAffected, duration, pointsPerSecond)

			// Record success with circuit breaker
			if bp.circuitBreaker != nil {
				bp.circuitBreaker.RecordSuccess()
			}
			return nil
		}

		log.Printf("Flush attempt %d failed: %v", attempt+1, err)

		// Record failure with circuit breaker
		if bp.circuitBreaker != nil {
			bp.circuitBreaker.RecordFailure()
		}

		// Exponential backoff with jitter (except on last attempt)
		if attempt < bp.maxRetries-1 {
			delay := bp.retryDelay * time.Duration(1<<uint(attempt))
			// Add some jitter (±20%)
			jitter := time.Duration(float64(delay) * 0.2 * (2.0*randFloat64() - 1.0))
			time.Sleep(delay + jitter)
		}
	}

	// All retries failed, write to WAL
	log.Printf("All %d retry attempts failed, writing %d records to WAL", bp.maxRetries, len(batch))
	return bp.flushToWAL(batch)
}

// flushToWAL writes buffered records to the Write Ahead Log
// This is called when the database is unavailable
func (bp *BatchProcessor) flushToWAL(batch []models.TelemetryPoint) error {
	if bp.wal == nil {
		return fmt.Errorf("WAL not configured, data will be lost")
	}

	for _, point := range batch {
		walRecord := WALRecord{
			Timestamp:            point.Timestamp,
			SatelliteID:          point.SatelliteID,
			BatteryChargePercent: point.BatteryChargePercent,
			StorageUsageMB:       point.StorageUsageMB,
			SignalStrengthDBM:    point.SignalStrengthDBM,
			IsAnomaly:            point.IsAnomaly,
		}
		if err := bp.wal.Write(walRecord); err != nil {
			return fmt.Errorf("failed to write to WAL: %w", err)
		}
	}

	log.Printf("Wrote %d records to WAL", len(batch))
	return nil
}

// randFloat64 returns a random float64 between 0 and 1
// Simple implementation without importing math/rand
func randFloat64() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

func (bp *BatchProcessor) insertBatch(ctx context.Context, batch []models.TelemetryPoint) (int64, error) {
	// Use pgx's batch insert for maximum performance
	tx, err := bp.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stmt := `
		INSERT INTO telemetry (
			time, satellite_id, battery_charge_percent,
			storage_usage_mb, signal_strength_dbm, is_anomaly
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, point := range batch {
		_, err := tx.Exec(ctx, stmt,
			point.Timestamp,
			point.SatelliteID,
			point.BatteryChargePercent,
			point.StorageUsageMB,
			point.SignalStrengthDBM,
			point.IsAnomaly,
		)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return int64(len(batch)), nil
}

func (bp *BatchProcessor) detectAnomaly(point models.TelemetryPoint) bool {
	// Simple threshold-based anomaly detection
	if point.BatteryChargePercent < bp.anomalyConfig.BatteryMinPercent {
		log.Printf("ANOMALY: Satellite %s battery critically low: %.2f%%",
			point.SatelliteID, point.BatteryChargePercent)
		return true
	}

	if point.StorageUsageMB > bp.anomalyConfig.StorageMaxMB {
		log.Printf("ANOMALY: Satellite %s storage critically high: %.2f MB",
			point.SatelliteID, point.StorageUsageMB)
		return true
	}

	if point.SignalStrengthDBM < bp.anomalyConfig.SignalMinDBM {
		log.Printf("ANOMALY: Satellite %s signal critically weak: %.2f dBm",
			point.SatelliteID, point.SignalStrengthDBM)
		return true
	}

	return false
}

// GetWAL returns the Write Ahead Log instance
func (bp *BatchProcessor) GetWAL() *WAL {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	return bp.wal
}

// GetCircuitBreaker returns the circuit breaker instance
func (bp *BatchProcessor) GetCircuitBreaker() *CircuitBreaker {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	return bp.circuitBreaker
}

// GetBufferSize returns the current buffer size
func (bp *BatchProcessor) GetBufferSize() int {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()
	return len(bp.buffer)
}

// GetPool returns the database connection pool
func (bp *BatchProcessor) GetPool() *pgxpool.Pool {
	return bp.pool
}
