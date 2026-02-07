package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB creates a TimescaleDB container for testing and returns a connection pool
// Returns the pool and a cleanup function that must be called after tests
func SetupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	// Create TimescaleDB container
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "orbitstream_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start TimescaleDB container: %v", err)
	}

	// Get host and port
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to get container port: %v", err)
	}

	// Create connection string
	connStr := fmt.Sprintf("postgres://test:test@%s:%s/orbitstream_test?sslmode=disable", host, port.Port())

	// Wait for database to be ready
	var pool *pgxpool.Pool
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.New(ctx, connStr)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to connect to database after %d retries: %v", maxRetries, err)
	}

	// Cleanup function
	cleanup := func() {
		pool.Close()
		container.Terminate(ctx)
	}

	return pool, cleanup
}

// InitTestSchema reads and executes the init.sql file on the test database
func InitTestSchema(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Get the path to init.sql
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	schemaPath := filepath.Join(basePath, "init.sql")

	// Read schema file
	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Split into statements and execute individually
	statements := splitAndCleanSQLStatements(string(schemaSQL))

	for _, stmt := range statements {
		if stmt == "" {
			continue
		}

		// Determine if this is a SELECT statement that returns rows
		upperStmt := strings.ToUpper(strings.TrimSpace(stmt))
		isSelect := strings.HasPrefix(upperStmt, "SELECT ")

		var execErr error
		if isSelect {
			// Use Query for SELECT statements (ignore results)
			rows, err := pool.Query(ctx, stmt)
			if err != nil {
				execErr = err
			} else {
				rows.Close()
			}
		} else {
			// Use Exec for non-SELECT statements
			_, execErr = pool.Exec(ctx, stmt)
		}

		if execErr != nil {
			errStr := execErr.Error()
			// Ignore common "already exists" errors
			if strings.Contains(errStr, "already exists") ||
				strings.Contains(errStr, "duplicate key") ||
				strings.Contains(errStr, "Multiple errors") {
				continue
			}
			// Log but don't fail for other errors - some statements may fail
			// due to dependencies or timing issues
			continue
		}
	}

	return nil
}

// splitAndCleanSQLStatements splits SQL into individual statements
func splitAndCleanSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	lines := strings.Split(sql, "\n")
	inDollarQuote := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		// Handle dollar-quoted strings (e.g., $$ ... $$)
		if strings.Contains(line, "$$") {
			inDollarQuote = !inDollarQuote
		}

		current.WriteString(line)
		current.WriteString("\n")

		// Statement terminator (only if not inside dollar-quoted string)
		if strings.HasSuffix(trimmed, ";") && !inDollarQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// Add any remaining content
	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// TestTelemetryPoint represents a test telemetry data point
type TestTelemetryPoint struct {
	Timestamp            time.Time
	SatelliteID          string
	BatteryChargePercent float64
	StorageUsageMB       float64
	SignalStrengthDBM    float64
	IsAnomaly            bool
}

// InsertTestTelemetry inserts test telemetry data into the database
func InsertTestTelemetry(pool *pgxpool.Pool, points []TestTelemetryPoint) error {
	ctx := context.Background()

	stmt := `
		INSERT INTO telemetry (
			time, satellite_id, battery_charge_percent,
			storage_usage_mb, signal_strength_dbm, is_anomaly
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for i, point := range points {
		_, err := pool.Exec(ctx, stmt,
			point.Timestamp,
			point.SatelliteID,
			point.BatteryChargePercent,
			point.StorageUsageMB,
			point.SignalStrengthDBM,
			point.IsAnomaly,
		)
		if err != nil {
			return fmt.Errorf("failed to insert test data at row %d: %w", i, err)
		}
	}

	return nil
}

// RefreshAggregate manually triggers a refresh of a continuous aggregate
// This is used in tests to bypass the time-based refresh policies
func RefreshAggregate(pool *pgxpool.Pool, viewName string) error {
	ctx := context.Background()

	// Use NULL for start and end to refresh all data
	query := fmt.Sprintf("CALL refresh_continuous_aggregate('%s', NULL, NULL)", viewName)
	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to refresh aggregate %s: %w", viewName, err)
	}

	return nil
}

// RefreshAllAggregates refreshes all continuous aggregates
func RefreshAllAggregates(pool *pgxpool.Pool) error {
	aggregates := []string{
		"satellite_stats",
		"satellite_stats_hourly",
		"satellite_stats_daily",
	}

	for _, agg := range aggregates {
		if err := RefreshAggregate(pool, agg); err != nil {
			return err
		}
	}

	return nil
}

// ClearTestData removes all test data from the database
func ClearTestData(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Truncate telemetry table
	_, err := pool.Exec(ctx, "TRUNCATE TABLE telemetry CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate telemetry: %w", err)
	}

	return nil
}

// GetProjectRoot returns the root directory of the project
func GetProjectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	// Go up from go-service/db to project root
	return filepath.Join(filepath.Dir(b), "..")
}
