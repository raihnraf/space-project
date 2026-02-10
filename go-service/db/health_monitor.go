package db

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthMonitor periodically checks database connectivity and triggers WAL replay
// when the database becomes available after an outage.
//
// This ensures that any data buffered to disk during a database outage is
// automatically replayed once the connection is restored.
type HealthMonitor struct {
	pool            *pgxpool.Pool
	checkInterval   time.Duration
	wal             *WAL
	batchProcessor  *BatchProcessor
	stopCh          chan struct{}
	wg              sync.WaitGroup
	isHealthy       bool
	healthMutex     sync.RWMutex
	lastCheckTime   time.Time
	lastCheckResult error
}

// NewHealthMonitor creates a new health monitor
// pool: database connection pool to monitor
// wal: write ahead log to replay when database recovers
// batchProcessor: batch processor to use for replaying records
func NewHealthMonitor(pool *pgxpool.Pool, wal *WAL, batchProcessor *BatchProcessor) *HealthMonitor {
	return &HealthMonitor{
		pool:           pool,
		checkInterval:  5 * time.Second,
		wal:            wal,
		batchProcessor: batchProcessor,
		stopCh:         make(chan struct{}),
		isHealthy:      false, // Will be determined on first check
	}
}

// SetCheckInterval sets the health check interval
func (hm *HealthMonitor) SetCheckInterval(interval time.Duration) {
	hm.checkInterval = interval
}

// Start begins the health monitoring loop
// It runs in a separate goroutine and periodically checks database connectivity
func (hm *HealthMonitor) Start() {
	hm.wg.Add(1)
	go hm.monitorLoop()

	// Perform initial health check
	hm.checkHealth()
}

// Stop gracefully stops the health monitor
// It waits for the monitoring loop to finish
func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
	hm.wg.Wait()
}

// monitorLoop is the main monitoring loop
// It runs health checks at regular intervals until stopped
func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.checkHealth()
		case <-hm.stopCh:
			log.Println("HealthMonitor: Stopping health monitor")
			return
		}
	}
}

// checkHealth performs a single health check and replays WAL if needed
func (hm *HealthMonitor) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := hm.pool.Ping(ctx)

	hm.healthMutex.Lock()
	hm.lastCheckTime = time.Now()
	hm.lastCheckResult = err
	wasHealthy := hm.isHealthy
	hm.isHealthy = (err == nil)
	hm.healthMutex.Unlock()

	// Log state changes
	if err == nil && !wasHealthy {
		log.Println("HealthMonitor: Database is now HEALTHY ✓")
		// Database just recovered, replay WAL
		hm.replayWAL()
	} else if err != nil && wasHealthy {
		log.Printf("HealthMonitor: Database is now UNHEALTHY ✗ (error: %v)", err)
	}

	// If database is healthy and has WAL records, try replay
	if err == nil {
		hm.replayWAL()
	}
}

// replayWAL replays all records from the WAL to the database
// It replays records in batches for efficiency
// If replay fails, it will be retried on the next health check
func (hm *HealthMonitor) replayWAL() {
	records, err := hm.wal.ReadAll()
	if err != nil {
		log.Printf("HealthMonitor: Failed to read WAL: %v", err)
		return
	}

	if len(records) == 0 {
		return
	}

	log.Printf("HealthMonitor: Replaying %d records from WAL", len(records))

	// Replay in batches of 1000 to avoid overwhelming the database
	batchSize := 1000
	successCount := 0

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		if err := hm.insertWALRecords(batch); err != nil {
			log.Printf("HealthMonitor: Failed to replay WAL batch %d-%d: %v", i, end, err)
			// Don't clear WAL - will retry on next check
			return
		}

		successCount += len(batch)
		log.Printf("HealthMonitor: Replayed batch %d-%d (%d/%d records)",
			i, end, successCount, len(records))
	}

	// All records successfully replayed, clear WAL
	if err := hm.wal.Clear(); err != nil {
		log.Printf("HealthMonitor: Failed to clear WAL after replay: %v", err)
		return
	}

	log.Printf("HealthMonitor: Successfully replayed and cleared %d WAL records", successCount)
}

// insertWALRecords inserts a batch of WAL records into the database
func (hm *HealthMonitor) insertWALRecords(records []WALRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := hm.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stmt := `
		INSERT INTO telemetry (
			time, satellite_id, battery_charge_percent,
			storage_usage_mb, signal_strength_dbm, is_anomaly
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, record := range records {
		_, err := tx.Exec(ctx, stmt,
			record.Timestamp,
			record.SatelliteID,
			record.BatteryChargePercent,
			record.StorageUsageMB,
			record.SignalStrengthDBM,
			record.IsAnomaly,
		)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// IsHealthy returns the current health status of the database
func (hm *HealthMonitor) IsHealthy() bool {
	hm.healthMutex.RLock()
	defer hm.healthMutex.RUnlock()
	return hm.isHealthy
}

// GetLastCheckTime returns the time of the last health check
func (hm *HealthMonitor) GetLastCheckTime() time.Time {
	hm.healthMutex.RLock()
	defer hm.healthMutex.RUnlock()
	return hm.lastCheckTime
}

// GetLastCheckResult returns the error from the last health check (if any)
func (hm *HealthMonitor) GetLastCheckResult() error {
	hm.healthMutex.RLock()
	defer hm.healthMutex.RUnlock()
	return hm.lastCheckResult
}
