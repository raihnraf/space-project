package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAggregateCalculationsCorrectness verifies that aggregates calculate AVG, MIN, MAX correctly
func TestAggregateCalculationsCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert test data with known values within a single hour
	baseTime := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	testPoints := []TestTelemetryPoint{
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-TEST-01",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(15 * time.Minute),
			SatelliteID:          "SAT-TEST-01",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       60000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(30 * time.Minute),
			SatelliteID:          "SAT-TEST-01",
			BatteryChargePercent: 70.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -70.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh the hourly aggregate
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Query the aggregate
	var avgBattery, minBattery, maxBattery float64
	var avgStorage, minStorage, maxStorage float64
	var avgSignal, minSignal, maxSignal float64
	var dataPoints int

	query := `
		SELECT avg_battery, min_battery, max_battery,
		       avg_storage, min_storage, max_storage,
		       avg_signal, min_signal, max_signal,
		       data_points
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`
	err = pool.QueryRow(ctx, query, "SAT-TEST-01").Scan(
		&avgBattery, &minBattery, &maxBattery,
		&avgStorage, &minStorage, &maxStorage,
		&avgSignal, &minSignal, &maxSignal,
		&dataPoints,
	)
	require.NoError(t, err)

	// Verify calculations (with small tolerance for floating point)
	assert.InDelta(t, 80.0, avgBattery, 0.01, "avg_battery should be 80.0")
	assert.InDelta(t, 70.0, minBattery, 0.01, "min_battery should be 70.0")
	assert.InDelta(t, 90.0, maxBattery, 0.01, "max_battery should be 90.0")

	assert.InDelta(t, 50000.0, avgStorage, 0.01, "avg_storage should be 50000.0")
	assert.InDelta(t, 40000.0, minStorage, 0.01, "min_storage should be 40000.0")
	assert.InDelta(t, 60000.0, maxStorage, 0.01, "max_storage should be 60000.0")

	assert.InDelta(t, -61.67, avgSignal, 0.1, "avg_signal should be approximately -61.67")
	assert.InDelta(t, -70.0, minSignal, 0.01, "min_signal should be -70.0")
	assert.InDelta(t, -55.0, maxSignal, 0.01, "max_signal should be -55.0")

	assert.Equal(t, 3, dataPoints, "data_points should be 3")
}

// TestAnomalyCountInAggregates verifies that anomaly_count field correctly counts anomalies
func TestAnomalyCountInAggregates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert test data with some anomalies
	baseTime := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	testPoints := []TestTelemetryPoint{
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-ANOMALY-01",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(10 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-01",
			BatteryChargePercent: 5.0, // Anomaly
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            true,
		},
		{
			Timestamp:            baseTime.Add(20 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-01",
			BatteryChargePercent: 3.0, // Anomaly
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            true,
		},
		{
			Timestamp:            baseTime.Add(30 * time.Minute),
			SatelliteID:          "SAT-ANOMALY-01",
			BatteryChargePercent: 85.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh the hourly aggregate
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Query the anomaly count
	var anomalyCount int
	query := `
		SELECT anomaly_count
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`
	err = pool.QueryRow(ctx, query, "SAT-ANOMALY-01").Scan(&anomalyCount)
	require.NoError(t, err)

	assert.Equal(t, 2, anomalyCount, "anomaly_count should be 2")
}

// TestMultiSatelliteAggregation verifies data is correctly grouped by satellite_id
func TestMultiSatelliteAggregation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert data for multiple satellites in the same time bucket
	baseTime := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Hour)
	testPoints := []TestTelemetryPoint{
		// SAT-A data
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-A",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(15 * time.Minute),
			SatelliteID:          "SAT-A",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		// SAT-B data (different values)
		{
			Timestamp:            baseTime,
			SatelliteID:          "SAT-B",
			BatteryChargePercent: 50.0,
			StorageUsageMB:       30000.0,
			SignalStrengthDBM:    -80.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseTime.Add(15 * time.Minute),
			SatelliteID:          "SAT-B",
			BatteryChargePercent: 40.0,
			StorageUsageMB:       30000.0,
			SignalStrengthDBM:    -80.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh aggregates
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Query for SAT-A
	var satAAvgBattery float64
	query := `
		SELECT avg_battery
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
	`
	err = pool.QueryRow(ctx, query, "SAT-A").Scan(&satAAvgBattery)
	require.NoError(t, err)

	// Query for SAT-B
	var satBAvgBattery float64
	err = pool.QueryRow(ctx, query, "SAT-B").Scan(&satBAvgBattery)
	require.NoError(t, err)

	// Verify each satellite has its own aggregate values
	assert.InDelta(t, 85.0, satAAvgBattery, 0.01, "SAT-A avg_battery should be 85.0")
	assert.InDelta(t, 45.0, satBAvgBattery, 0.01, "SAT-B avg_battery should be 45.0")

	// Verify we have exactly 2 rows (one per satellite)
	var count int
	countQuery := `SELECT COUNT(*) FROM satellite_stats_hourly`
	err = pool.QueryRow(ctx, countQuery).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "Should have 2 aggregate rows (one per satellite)")
}

// TestTimeBoundaryHandling verifies data at hour boundaries is correctly bucketed
func TestTimeBoundaryHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert data across hour boundaries
	// Use a time 4 hours ago to ensure it falls within the aggregate's start_offset
	baseHour := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)
	testPoints := []TestTelemetryPoint{
		// Data at the beginning of hour 1
		{
			Timestamp:            baseHour,
			SatelliteID:          "SAT-BOUNDARY",
			BatteryChargePercent: 100.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		// Data at the end of hour 1 (59th minute)
		{
			Timestamp:            baseHour.Add(59 * time.Minute),
			SatelliteID:          "SAT-BOUNDARY",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		// Data at the beginning of hour 2
		{
			Timestamp:            baseHour.Add(1 * time.Hour),
			SatelliteID:          "SAT-BOUNDARY",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		// Data at the end of hour 2
		{
			Timestamp:            baseHour.Add(1*time.Hour + 59*time.Minute),
			SatelliteID:          "SAT-BOUNDARY",
			BatteryChargePercent: 70.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh aggregates
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Query for all buckets
	query := `
		SELECT bucket, avg_battery, data_points
		FROM satellite_stats_hourly
		WHERE satellite_id = $1
		ORDER BY bucket
	`

	rows, err := pool.Query(ctx, query, "SAT-BOUNDARY")
	require.NoError(t, err)
	defer rows.Close()

	var buckets []time.Time
	var avgBatteries []float64
	var dataPointCounts []int

	for rows.Next() {
		var bucket time.Time
		var avgBattery float64
		var dataPoints int
		err := rows.Scan(&bucket, &avgBattery, &dataPoints)
		require.NoError(t, err)
		buckets = append(buckets, bucket)
		avgBatteries = append(avgBatteries, avgBattery)
		dataPointCounts = append(dataPointCounts, dataPoints)
	}

	// Should have 2 buckets (one per hour)
	require.Equal(t, 2, len(buckets), "Should have 2 buckets")

	// First bucket should have avg of 100 and 90
	assert.InDelta(t, 95.0, avgBatteries[0], 0.01, "First bucket avg_battery should be 95.0")
	assert.Equal(t, 2, dataPointCounts[0], "First bucket should have 2 data points")

	// Second bucket should have avg of 80 and 70
	assert.InDelta(t, 75.0, avgBatteries[1], 0.01, "Second bucket avg_battery should be 75.0")
	assert.Equal(t, 2, dataPointCounts[1], "Second bucket should have 2 data points")
}

// TestIncrementalRefresh verifies that new data creates new buckets while old buckets remain unchanged
func TestIncrementalRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert initial data in hour 1 (4 hours ago to be within range)
	hour1 := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)
	initialPoints := []TestTelemetryPoint{
		{
			Timestamp:            hour1,
			SatelliteID:          "SAT-INCREMENTAL",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, initialPoints)
	require.NoError(t, err)

	// First refresh
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Count buckets after first refresh
	var count1 int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM satellite_stats_hourly WHERE satellite_id = $1
	`, "SAT-INCREMENTAL").Scan(&count1)
	require.NoError(t, err)
	assert.Equal(t, 1, count1, "Should have 1 bucket after first refresh")

	// Insert data for hour 2
	hour2 := hour1.Add(1 * time.Hour)
	newPoints := []TestTelemetryPoint{
		{
			Timestamp:            hour2,
			SatelliteID:          "SAT-INCREMENTAL",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       60000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, newPoints)
	require.NoError(t, err)

	// Second refresh
	err = RefreshAggregate(pool, "satellite_stats_hourly")
	require.NoError(t, err)

	// Count buckets after second refresh
	var count2 int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM satellite_stats_hourly WHERE satellite_id = $1
	`, "SAT-INCREMENTAL").Scan(&count2)
	require.NoError(t, err)
	assert.Equal(t, 2, count2, "Should have 2 buckets after second refresh")

	// Verify first bucket still has original value
	var firstBucketAvg float64
	err = pool.QueryRow(ctx, `
		SELECT avg_battery FROM satellite_stats_hourly
		WHERE satellite_id = $1 AND bucket = $2
	`, "SAT-INCREMENTAL", hour1).Scan(&firstBucketAvg)
	require.NoError(t, err)
	assert.InDelta(t, 80.0, firstBucketAvg, 0.01, "First bucket should still have avg_battery=80.0")

	// Verify second bucket has new value
	var secondBucketAvg float64
	err = pool.QueryRow(ctx, `
		SELECT avg_battery FROM satellite_stats_hourly
		WHERE satellite_id = $1 AND bucket = $2
	`, "SAT-INCREMENTAL", hour2).Scan(&secondBucketAvg)
	require.NoError(t, err)
	assert.InDelta(t, 90.0, secondBucketAvg, 0.01, "Second bucket should have avg_battery=90.0")
}

// TestDailyAggregateCorrectness verifies the daily aggregate works correctly
func TestDailyAggregateCorrectness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Insert data across multiple hours within a single day
	// Use a time 8 days ago to be within the daily aggregate's start_offset (7 days)
	baseDay := time.Now().UTC().Add(-8 * 24 * time.Hour).Truncate(24 * time.Hour)
	testPoints := []TestTelemetryPoint{
		{
			Timestamp:            baseDay.Add(1 * time.Hour),
			SatelliteID:          "SAT-DAILY",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       50000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseDay.Add(6 * time.Hour),
			SatelliteID:          "SAT-DAILY",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       60000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            baseDay.Add(12 * time.Hour),
			SatelliteID:          "SAT-DAILY",
			BatteryChargePercent: 70.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -65.0,
			IsAnomaly:            false,
		},
	}

	err = InsertTestTelemetry(pool, testPoints)
	require.NoError(t, err)

	// Refresh the daily aggregate
	err = RefreshAggregate(pool, "satellite_stats_daily")
	require.NoError(t, err)

	// Query the daily aggregate
	var avgBattery, minBattery, maxBattery float64
	var dataPoints int

	query := `
		SELECT avg_battery, min_battery, max_battery, data_points
		FROM satellite_stats_daily
		WHERE satellite_id = $1
	`
	err = pool.QueryRow(ctx, query, "SAT-DAILY").Scan(
		&avgBattery, &minBattery, &maxBattery, &dataPoints,
	)
	require.NoError(t, err)

	// Verify calculations
	assert.InDelta(t, 80.0, avgBattery, 0.01, "avg_battery should be 80.0")
	assert.InDelta(t, 70.0, minBattery, 0.01, "min_battery should be 70.0")
	assert.InDelta(t, 90.0, maxBattery, 0.01, "max_battery should be 90.0")
	assert.Equal(t, 3, dataPoints, "data_points should be 3")
}
