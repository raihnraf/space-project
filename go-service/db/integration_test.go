package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"orbitstream/models"
)

// TestFullIngestToAggregatePipeline tests the complete data flow:
// HTTP ingest -> batch insert -> aggregate refresh -> verify
func TestFullIngestToAggregatePipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Simulate batch processor inserting data
	baseTime := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)

	// Create test telemetry points simulating real ingest
	testPoints := []TestTelemetryPoint{
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-PIPELINE-01",
			BatteryChargePercent: 85.5,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(5 * time.Minute),
			SatelliteID:          "SAT-PIPELINE-01",
			BatteryChargePercent: 84.0,
			StorageUsageMB:       45200.0,
			SignalStrengthDBM:    -56.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(10 * time.Minute),
			SatelliteID:          "SAT-PIPELINE-01",
			BatteryChargePercent: 83.5,
			StorageUsageMB:       45300.0,
			SignalStrengthDBM:    -57.0,
			IsAnomaly:            false,
		},
	}

	// Step 1: Insert data (simulating batch processor)
	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err, "Should insert telemetry data without error")

	// Step 2: Verify raw data exists
	var rawCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM telemetry WHERE satellite_id = $1
	`, "SAT-PIPELINE-01").Scan(&rawCount)
	require.NoError(t, err)
	assert.Equal(t, 3, rawCount, "Should have 3 raw telemetry rows")

	// Step 3: Refresh continuous aggregates
	err = RefreshAllAggregates(pool)
	require.NoError(t, err, "Should refresh all aggregates without error")

	// Step 4: Verify satellite_stats (5-minute buckets)
	var fiveMinCount int
	var fiveMinAvgBattery float64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*), AVG(avg_battery)
		FROM satellite_stats
		WHERE satellite_id = $1
	`, "SAT-PIPELINE-01").Scan(&fiveMinCount, &fiveMinAvgBattery)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, fiveMinCount, 1, "Should have at least one 5-minute bucket")
	assert.InDelta(t, 84.33, fiveMinAvgBattery, 0.5, "5-min avg_battery should be approximately 84.33")

	// Step 5: Verify satellite_stats_hourly
	var hourlyCount int
	var hourlyAvgBattery float64
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*), AVG(avg_battery)
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`, "SAT-PIPELINE-01").Scan(&hourlyCount, &hourlyAvgBattery)
	require.NoError(t, err)
	assert.Equal(t, 1, hourlyCount, "Should have exactly one hourly bucket")
	assert.InDelta(t, 84.33, hourlyAvgBattery, 0.5, "Hourly avg_battery should be approximately 84.33")

	// Step 6: Verify data aggregation is correct
	assert.InDelta(t, fiveMinAvgBattery, hourlyAvgBattery, 0.5,
		"5-minute and hourly averages should be similar for same data")
}

// TestBatchProcessorWithAnomalies verifies anomaly detection propagates to aggregates
func TestBatchProcessorWithAnomalies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Create batch processor with test configuration
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}
	batchProcessor := NewBatchProcessor(pool, 100, 1*time.Second, anomalyConfig)

	// Simulate data ingest with some anomalies
	baseTime := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)

	testPoints := []models.TelemetryPoint{
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-ANOMALY-TEST",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
		},
		{
			Timestamp:            baseTime.Add(5 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-TEST",
			BatteryChargePercent: 5.0, // Below threshold - should be anomaly
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
		},
		{
			Timestamp:            baseTime.Add(10 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-TEST",
			BatteryChargePercent: 85.0,
			StorageUsageMB:       98000.0, // Above threshold - should be anomaly
			SignalStrengthDBM:    -60.0,
		},
		{
			Timestamp:            baseTime.Add(15 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-TEST",
			BatteryChargePercent: 75.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -110.0, // Below threshold - should be anomaly
		},
	}

	// Add points to batch processor (this marks anomalies)
	for _, point := range testPoints {
		batchProcessor.Add(point)
	}

	// Force immediate flush by calling flush directly
	batchProcessor.flush()

	// Verify raw data with anomalies
	var anomalyCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM telemetry
		WHERE satellite_id = $1 AND is_anomaly = true
	`, "SAT-ANOMALY-TEST").Scan(&anomalyCount)
	require.NoError(t, err)
	assert.Equal(t, 3, anomalyCount, "Should have 3 anomalies in raw data")

	// Refresh aggregates
	err = RefreshAllAggregates(pool)
	require.NoError(t, err)

	// Verify anomaly count in hourly aggregate
	var hourlyAnomalyCount int
	err = pool.QueryRow(ctx, `
		SELECT anomaly_count
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`, "SAT-ANOMALY-TEST").Scan(&hourlyAnomalyCount)
	require.NoError(t, err)
	assert.Equal(t, 3, hourlyAnomalyCount, "Hourly aggregate should have anomaly_count=3")

	// Verify data_points count
	var dataPoints int
	err = pool.QueryRow(ctx, `
		SELECT data_points
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`, "SAT-ANOMALY-TEST").Scan(&dataPoints)
	require.NoError(t, err)
	assert.Equal(t, 4, dataPoints, "Should have 4 total data points")
}



// TestQueryPerformanceComparison demonstrates the performance benefit of aggregates
// This is more of a sanity check than a precise benchmark
func TestQueryPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert test data - multiple satellites, multiple hours
	baseTime := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)
	var testPoints []TestTelemetryPoint

	// Generate data for 5 satellites across 3 hours
	for satIdx := 0; satIdx < 5; satIdx++ {
		satelliteID := fmt.Sprintf("SAT-PERF-%d", satIdx)
		for hourOffset := 0; hourOffset < 3; hourOffset++ {
			hour := baseTime.Add(time.Duration(hourOffset) * time.Hour)
			for minute := 0; minute < 60; minute += 5 {
				testPoints = append(testPoints, TestTelemetryPoint{
					Timestamp:            hour.Add(time.Duration(minute) * time.Minute),
					SatelliteID:          satelliteID,
					BatteryChargePercent: 50.0 + float64(minute),
					StorageUsageMB:       50000.0,
					SignalStrengthDBM:    -60.0,
					IsAnomaly:            false,
				})
			}
		}
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh aggregates
	err = RefreshAllAggregates(pool)
	require.NoError(t, err)

	// Query raw data count
	var rawCount int
	rawStart := time.Now()
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM telemetry
	`).Scan(&rawCount)
	rawDuration := time.Since(rawStart)
	require.NoError(t, err)
	assert.Greater(t, rawCount, 0, "Should have raw data")

	// Query hourly aggregate count
	var hourlyCount int
	hourlyStart := time.Now()
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM satellite_stats_hourly
	`).Scan(&hourlyCount)
	hourlyDuration := time.Since(hourlyStart)
	require.NoError(t, err)

	// Log the comparison (informational, not a strict assertion)
	t.Logf("Query performance: raw=%v (count=%d), hourly=%v (count=%d)",
		rawDuration, rawCount, hourlyDuration, hourlyCount)

	// The hourly aggregate should have significantly fewer rows than raw data
	// (exact counts may vary due to time bucket boundaries)
	assert.Less(t, hourlyCount, rawCount,
		"Hourly aggregate should have fewer rows than raw data")
}




