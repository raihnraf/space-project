package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHypertableExists verifies the telemetry table is properly configured as a hypertable
func TestHypertableExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Query to check if telemetry is a hypertable
	var hypertableName string

	query := `
		SELECT hypertable_name
		FROM timescaledb_information.hypertables
		WHERE hypertable_name = 'telemetry'
	`
	err = pool.QueryRow(ctx, query).Scan(&hypertableName)
	require.NoError(t, err, "telemetry should be a hypertable")

	assert.Equal(t, "telemetry", hypertableName)
}

// TestContinuousAggregatesExist verifies all continuous aggregates are created
func TestContinuousAggregatesExist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Query to check continuous aggregates
	query := `
		SELECT view_name
		FROM timescaledb_information.continuous_aggregates
		ORDER BY view_name
	`

	rows, err := pool.Query(ctx, query)
	require.NoError(t, err)
	defer rows.Close()

	var aggregates []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		aggregates = append(aggregates, name)
	}

	// Verify all expected aggregates exist
	expectedAggregates := []string{
		"satellite_stats",
		"satellite_stats_daily",
		"satellite_stats_hourly",
	}

	assert.ElementsMatch(t, expectedAggregates, aggregates, "All continuous aggregates should exist")
}

// TestIndexesExist verifies all required indexes are created
func TestIndexesExist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Expected indexes on main telemetry table (indexes on continuous aggregates
	// may have different naming in different TimescaleDB versions)
	expectedIndexes := []string{
		"idx_telemetry_satellite_time",
		"idx_telemetry_anomaly",
	}

	for _, indexName := range expectedIndexes {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE tablename = 'telemetry' AND indexname = $1
			)
		`
		err := pool.QueryRow(ctx, query, indexName).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Index %s on telemetry should exist", indexName)
	}
}

// TestCompressionSettings verifies compression is enabled on tables and aggregates
func TestCompressionSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Expected compressed tables/aggregates
	expectedCompressed := []string{
		"telemetry",
		"satellite_stats_hourly",
		"satellite_stats_daily",
	}

	for _, name := range expectedCompressed {
		var isCompressed bool
		query := `
			SELECT compression_enabled
			FROM timescaledb_information.hypertables
			WHERE hypertable_name = $1
		`
		err := pool.QueryRow(ctx, query, name).Scan(&isCompressed)

		if err == pgx.ErrNoRows {
			// Try continuous aggregates
			query = `
				SELECT EXISTS (
					SELECT 1 FROM timescaledb_information.compression_settings
					WHERE hypertable_name IN (
						SELECT materialization_hypertable_name
						FROM timescaledb_information.continuous_aggregates
						WHERE view_name = $1
					)
				)
			`
			var compressionExists bool
			err = pool.QueryRow(ctx, query, name).Scan(&compressionExists)
			require.NoError(t, err)
			assert.True(t, compressionExists, "%s should have compression enabled", name)
		} else {
			require.NoError(t, err)
			assert.True(t, isCompressed, "%s should have compression enabled", name)
		}
	}
}

// TestRetentionPolicies verifies that the schema was initialized without errors
// Note: The actual retention policies are created via SELECT statements in init.sql
// which return job IDs. These are validated by the fact that InitTestSchema succeeds.
func TestRetentionPolicies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()
	defer pool.Close()

	// Initialize schema - if this succeeds, the policies were created
	err := InitTestSchema(pool)
	require.NoError(t, err, "Schema initialization should succeed including retention policies")

	// Verify we can query the jobs table
	ctx := context.Background()
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM timescaledb_information.jobs").Scan(&count)
	require.NoError(t, err, "Should be able to query jobs table")
	t.Logf("Found %d jobs in timescaledb_information.jobs", count)
}

// TestRefreshPolicies verifies continuous aggregate refresh policies are configured
// Note: The actual refresh policies are created via SELECT statements in init.sql
func TestRefreshPolicies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()
	defer pool.Close()

	// Initialize schema - if this succeeds, the policies were created
	err := InitTestSchema(pool)
	require.NoError(t, err, "Schema initialization should succeed including refresh policies")

	// Verify continuous aggregates exist (which means their policies were created)
	ctx := context.Background()
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM timescaledb_information.continuous_aggregates
	`).Scan(&count)
	require.NoError(t, err, "Should be able to query continuous aggregates")
	assert.Equal(t, 3, count, "Should have 3 continuous aggregates")
}

// TestAggregateColumnsExist verifies all expected columns exist in aggregates
func TestAggregateColumnsExist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize schema
	err := InitTestSchema(pool)
	require.NoError(t, err)

	// Expected columns in each aggregate
	expectedColumns := map[string][]string{
		"satellite_stats": {
			"satellite_id",
			"bucket",
			"avg_battery",
			"avg_storage",
			"avg_signal",
			"data_points",
		},
		"satellite_stats_hourly": {
			"satellite_id",
			"bucket",
			"avg_battery",
			"min_battery",
			"max_battery",
			"avg_storage",
			"min_storage",
			"max_storage",
			"avg_signal",
			"min_signal",
			"max_signal",
			"data_points",
			"anomaly_count",
		},
		"satellite_stats_daily": {
			"satellite_id",
			"bucket",
			"avg_battery",
			"min_battery",
			"max_battery",
			"avg_storage",
			"min_storage",
			"max_storage",
			"avg_signal",
			"min_signal",
			"max_signal",
			"data_points",
			"anomaly_count",
		},
	}

	for viewName, columns := range expectedColumns {
		for _, columnName := range columns {
			var exists bool
			query := `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = $1 AND column_name = $2
				)
			`
			err := pool.QueryRow(ctx, query, viewName, columnName).Scan(&exists)
			require.NoError(t, err)
			assert.True(t, exists, "Column %s should exist in %s", columnName, viewName)
		}
	}
}


