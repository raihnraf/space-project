package db

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"orbitstream/models"
)

type BatchProcessor struct {
	pool          *pgxpool.Pool
	batchSize     int
	batchTimeout  time.Duration
	buffer        []models.TelemetryPoint
	bufferMutex   sync.Mutex
	ticker        *time.Ticker
	done          chan bool
	anomalyConfig AnomalyConfig
}

type AnomalyConfig struct {
	BatteryMinPercent float64
	StorageMaxMB      float64
	SignalMinDBM      float64
}

func NewBatchProcessor(pool *pgxpool.Pool, batchSize int, batchTimeout time.Duration, anomalyConfig AnomalyConfig) *BatchProcessor {
	return &BatchProcessor{
		pool:         pool,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		buffer:       make([]models.TelemetryPoint, 0, batchSize),
		done:         make(chan bool),
		anomalyConfig: anomalyConfig,
	}
}

func (bp *BatchProcessor) Add(point models.TelemetryPoint) {
	bp.bufferMutex.Lock()
	defer bp.bufferMutex.Unlock()

	// Check for anomalies
	point.IsAnomaly = bp.detectAnomaly(point)

	bp.buffer = append(bp.buffer, point)

	// If buffer reaches batch size, trigger immediate flush
	if len(bp.buffer) >= bp.batchSize {
		go bp.flush()
	}
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

	// Perform batch insert
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	rowsAffected, err := bp.insertBatch(ctx, batch)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Failed to insert batch of %d: %v", len(batch), err)
		return
	}

	pointsPerSecond := float64(rowsAffected) / duration.Seconds()
	log.Printf("Inserted %d rows in %v (%.0f points/sec)",
		rowsAffected, duration, pointsPerSecond)
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
