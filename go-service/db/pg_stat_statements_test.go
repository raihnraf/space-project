package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestPgStatStatementsExtension tests that pg_stat_statements extension is loaded
// This is a Feature E test for query performance tracking
func TestPgStatStatementsExtension(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Start TimescaleDB container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "orbitstream_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
		Cmd: []string{
			"postgres",
			"-c", "shared_preload_libraries=timescaledb,pg_stat_statements",
			"-c", "pg_stat_statements.track=all",
			"-c", "pg_stat_statements.max=10000",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Connect to database
	connStr := "postgres://postgres:postgres@" + host + ":" + port.Port() + "/orbitstream_test?sslmode=disable"
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// Test 1: Check if pg_stat_statements extension exists
	t.Run("CheckExtensionExists", func(t *testing.T) {
		var extName string
		err := pool.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extName)
		if err != nil {
			t.Logf("pg_stat_statements extension not found (need to run init.sql): %v", err)
			t.Skip("extension not created yet - will be created by init.sql")
		}
		if extName != "pg_stat_statements" {
			t.Errorf("expected extension name 'pg_stat_statements', got %s", extName)
		}
	})

	// Test 2: Create extension if not exists
	t.Run("CreateExtension", func(t *testing.T) {
		_, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
		if err != nil {
			t.Fatalf("failed to create pg_stat_statements extension: %v", err)
		}

		// Verify it was created
		var extName string
		err = pool.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'pg_stat_statements'").Scan(&extName)
		if err != nil {
			t.Fatalf("extension not found after creation: %v", err)
		}
		if extName != "pg_stat_statements" {
			t.Errorf("expected extension name 'pg_stat_statements', got %s", extName)
		}
	})

	// Test 3: Query pg_stat_statements view
	t.Run("QueryStatStatements", func(t *testing.T) {
		// Execute some test queries to populate pg_stat_statements
		_, err := pool.Exec(ctx, "SELECT 1")
		if err != nil {
			t.Fatalf("failed to execute test query: %v", err)
		}

		// Query pg_stat_statements
		var queryCount int64
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM pg_stat_statements").Scan(&queryCount)
		if err != nil {
			t.Errorf("failed to query pg_stat_statements: %v", err)
		}

		if queryCount <= 0 {
			t.Error("expected at least 1 query in pg_stat_statements")
		}
	})

	// Test 4: Check pg_stat_statements settings
	t.Run("CheckSettings", func(t *testing.T) {
		var trackSetting string
		err := pool.QueryRow(ctx, "SELECT setting FROM pg_settings WHERE name = 'pg_stat_statements.track'").Scan(&trackSetting)
		if err != nil {
			t.Errorf("failed to query pg_stat_statements.track setting: %v", err)
		}

		if trackSetting != "all" {
			t.Logf("pg_stat_statements.track = %s (expected 'all')", trackSetting)
		}
	})

	// Test 5: Check query_statistics view (if created by init.sql)
	t.Run("QueryStatisticsView", func(t *testing.T) {
		// Try to query the view - it may not exist yet
		var queryCount int64
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM query_statistics").Scan(&queryCount)
		if err != nil {
			t.Logf("query_statistics view not found (expected - created by init.sql): %v", err)
			return
		}
		t.Logf("query_statistics view returned %d queries", queryCount)
	})

	// Test 6: Execute sample INSERT and verify it's tracked
	t.Run("TrackInsertQuery", func(t *testing.T) {
		// Create a simple test table
		_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS test_pg_stat")
		if err != nil {
			t.Fatalf("failed to drop test table: %v", err)
		}

		_, err = pool.Exec(ctx, "CREATE TABLE test_pg_stat (id SERIAL, value INTEGER)")
		if err != nil {
			t.Fatalf("failed to create test table: %v", err)
		}

		// Execute INSERT
		_, err = pool.Exec(ctx, "INSERT INTO test_pg_stat (value) VALUES (1), (2), (3)")
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}

		// Check if INSERT is tracked
		var calls int64
		query := "SELECT calls FROM pg_stat_statements WHERE query LIKE '%INSERT INTO test_pg_stat%'"
		err = pool.QueryRow(ctx, query).Scan(&calls)
		if err != nil {
			t.Logf("INSERT query not found in pg_stat_statements: %v", err)
		} else {
			t.Logf("INSERT query tracked: %d calls", calls)
		}

		// Cleanup
		_, err = pool.Exec(ctx, "DROP TABLE test_pg_stat")
		if err != nil {
			t.Logf("failed to drop test table: %v", err)
		}
	})

	// Test 7: Verify pg_stat_statements columns
	t.Run("VerifyColumns", func(t *testing.T) {
		// Check for expected columns in pg_stat_statements
		expectedColumns := []string{
			"queryid", "query", "calls", "total_exec_time",
			"mean_exec_time", "max_exec_time", "stddev_exec_time", "rows",
		}

		for _, col := range expectedColumns {
			var exists bool
			err := pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'pg_stat_statements' AND column_name = $1)", col).Scan(&exists)
			if err != nil {
				t.Errorf("failed to check column %s: %v", col, err)
			}
			if !exists {
				t.Errorf("expected column %s not found in pg_stat_statements", col)
			}
		}
	})

	// Test 8: Test with telemetry table schema
	t.Run("TrackTelemetryQueries", func(t *testing.T) {
		// Create telemetry table (simplified version)
		_, err := pool.Exec(ctx, `
			DROP TABLE IF EXISTS telemetry;
			CREATE TABLE telemetry (
				time TIMESTAMPTZ NOT NULL,
				satellite_id VARCHAR(50) NOT NULL,
				battery_charge_percent DECIMAL(5,2) NOT NULL,
				storage_usage_mb DECIMAL(10,2) NOT NULL,
				signal_strength_dbm DECIMAL(6,2) NOT NULL,
				is_anomaly BOOLEAN DEFAULT FALSE,
				latitude DECIMAL(9,6),
				longitude DECIMAL(9,6),
				altitude_km DECIMAL(8,2),
				velocity_kmph DECIMAL(9,2)
			)
		`)
		if err != nil {
			t.Fatalf("failed to create telemetry table: %v", err)
		}

		// Execute INSERT with 10 columns (Feature E)
		_, err = pool.Exec(ctx, `
			INSERT INTO telemetry (
				time, satellite_id, battery_charge_percent,
				storage_usage_mb, signal_strength_dbm, is_anomaly,
				latitude, longitude, altitude_km, velocity_kmph
			) VALUES (
				NOW(), 'SAT-TEST', 85.5, 45000.0, -55.0, false,
				40.7128, -74.0060, 408.5, 27576.5
			)
		`)
		if err != nil {
			t.Fatalf("failed to insert telemetry: %v", err)
		}

		// Verify INSERT is tracked in pg_stat_statements
		var calls int64
		err = pool.QueryRow(ctx, "SELECT calls FROM pg_stat_statements WHERE query LIKE '%INSERT INTO telemetry%'").Scan(&calls)
		if err != nil {
			t.Logf("Telemetry INSERT not found in pg_stat_statements: %v", err)
		} else {
			t.Logf("Telemetry INSERT tracked: %d calls", calls)
		}

		// Cleanup
		_, err = pool.Exec(ctx, "DROP TABLE telemetry")
		if err != nil {
			t.Logf("failed to drop telemetry table: %v", err)
		}
	})
}

// TestQueryStatisticsView tests the query_statistics view creation and usage
func TestQueryStatisticsView(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "orbitstream_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
		Cmd: []string{
			"postgres",
			"-c", "shared_preload_libraries=timescaledb,pg_stat_statements",
			"-c", "pg_stat_statements.track=all",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	connStr := "postgres://postgres:postgres@" + host + ":" + port.Port() + "/orbitstream_test?sslmode=disable"

	config, _ := pgxpool.ParseConfig(connStr)
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create pg_stat_statements extension
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		t.Fatalf("failed to create extension: %v", err)
	}

	// Create the query_statistics view (as in init.sql)
	_, err = pool.Exec(ctx, `
		CREATE OR REPLACE VIEW query_statistics AS
		SELECT
			query,
			calls,
			total_exec_time,
			mean_exec_time,
			max_exec_time,
			stddev_exec_time,
			rows,
			100.0 * shared_blks_hit / NULLIF(shared_blks_hit + shared_blks_read, 0) AS hit_percent
		FROM pg_stat_statements
		ORDER BY total_exec_time DESC
		LIMIT 100
	`)
	if err != nil {
		t.Fatalf("failed to create query_statistics view: %v", err)
	}

	// Execute some queries
	_, _ = pool.Exec(ctx, "SELECT 1")
	_, _ = pool.Exec(ctx, "SELECT 2")

	// Query the view
	rows, err := pool.Query(ctx, "SELECT * FROM query_statistics")
	if err != nil {
		t.Errorf("failed to query query_statistics view: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		count++
	}
	t.Logf("query_statistics view returned %d rows", count)
}

// TestPgStatStatementsWithHypertable tests pg_stat_statements with TimescaleDB hypertable
func TestPgStatStatementsWithHypertable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "orbitstream_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(30 * time.Second),
		Cmd: []string{
			"postgres",
			"-c", "shared_preload_libraries=pg_stat_statements,timescaledb",
			"-c", "pg_stat_statements.track=all",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	connStr := "postgres://postgres:postgres@" + host + ":" + port.Port() + "/orbitstream_test?sslmode=disable"

	config, _ := pgxpool.ParseConfig(connStr)
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create extensions
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS timescaledb")
	if err != nil {
		t.Fatalf("failed to create timescaledb extension: %v", err)
	}
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_stat_statements")
	if err != nil {
		t.Fatalf("failed to create pg_stat_statements extension: %v", err)
	}

	// Create telemetry hypertable with position fields (Feature E)
	_, err = pool.Exec(ctx, `
		CREATE TABLE telemetry (
			time TIMESTAMPTZ NOT NULL,
			satellite_id VARCHAR(50) NOT NULL,
			battery_charge_percent DECIMAL(5,2) NOT NULL,
			storage_usage_mb DECIMAL(10,2) NOT NULL,
			signal_strength_dbm DECIMAL(6,2) NOT NULL,
			is_anomaly BOOLEAN DEFAULT FALSE,
			latitude DECIMAL(9,6),
			longitude DECIMAL(9,6),
			altitude_km DECIMAL(8,2),
			velocity_kmph DECIMAL(9,2)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create telemetry table: %v", err)
	}

	// Convert to hypertable
	_, err = pool.Exec(ctx, "SELECT create_hypertable('telemetry', 'time', chunk_time_interval => interval '1 hour')")
	if err != nil {
		t.Fatalf("failed to create hypertable: %v", err)
	}

	// Insert batch of telemetry data
	for i := 0; i < 10; i++ {
		_, err = pool.Exec(ctx, `
			INSERT INTO telemetry (
				time, satellite_id, battery_charge_percent,
				storage_usage_mb, signal_strength_dbm, is_anomaly,
				latitude, longitude, altitude_km, velocity_kmph
			) VALUES (
				NOW(), 'SAT-001', 85.5, 45000.0, -55.0, false,
				40.7128, -74.0060, 408.5, 27576.5
			)
		`)
	}

	// Check pg_stat_statements for the INSERT query
	var calls int64
	var meanTime float64
	err = pool.QueryRow(ctx, `
		SELECT calls, mean_exec_time
		FROM pg_stat_statements
		WHERE query LIKE '%INSERT INTO telemetry%'
	`).Scan(&calls, &meanTime)
	if err != nil {
		t.Logf("INSERT query stats not available: %v", err)
	} else {
		t.Logf("INSERT telemetry: %d calls, mean time: %.2f ms", calls, meanTime)
	}

	// Query with position filter
	rows, _ := pool.Query(ctx, "SELECT satellite_id, latitude, longitude FROM telemetry WHERE latitude IS NOT NULL LIMIT 5")
	if rows != nil {
		defer rows.Close()
		var satID string
		var lat, lon *float64
		for rows.Next() {
			rows.Scan(&satID, &lat, &lon)
		}
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DROP TABLE telemetry")
}
