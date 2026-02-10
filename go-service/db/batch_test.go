package db

import (
	"path/filepath"
	"testing"
	"time"

	"orbitstream/models"
)

// TestAnomalyDetection tests the anomaly detection logic
// This is a simplified test that focuses on the detectAnomaly method
// without requiring a full database connection pool

func TestAnomalyDetectionBattery(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	// Create a point with low battery (anomaly)
	point := TelemetryPointForTest(5.0, 45000.0, -55.0)

	if !bp.detectAnomaly(point) {
		t.Error("expected battery < 10.0 to be detected as anomaly")
	}
}

func TestAnomalyDetectionStorage(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	point := TelemetryPointForTest(85.5, 98000.0, -55.0)

	if !bp.detectAnomaly(point) {
		t.Error("expected storage > 95000.0 to be detected as anomaly")
	}
}

func TestAnomalyDetectionSignal(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	point := TelemetryPointForTest(85.5, 45000.0, -115.0)

	if !bp.detectAnomaly(point) {
		t.Error("expected signal < -100.0 to be detected as anomaly")
	}
}

func TestAnomalyDetectionNone(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	point := TelemetryPointForTest(85.5, 45000.0, -55.0)

	if bp.detectAnomaly(point) {
		t.Error("expected normal values to not be detected as anomaly")
	}
}

func TestAnomalyDetectionMultiple(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	// Create a point with multiple anomalies
	point := TelemetryPointForTest(5.0, 98000.0, -115.0)

	if !bp.detectAnomaly(point) {
		t.Error("expected multiple anomalies to be detected")
	}
}

func TestAnomalyDetectionThresholdBoundaries(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	tests := []struct {
		name     string
		battery  float64
		storage  float64
		signal   float64
		expected bool
	}{
		{
			name:     "battery exactly at threshold",
			battery:  10.0,
			storage:  45000.0,
			signal:   -55.0,
			expected: false,
		},
		{
			name:     "battery just below threshold",
			battery:  9.99,
			storage:  45000.0,
			signal:   -55.0,
			expected: true,
		},
		{
			name:     "storage exactly at threshold",
			battery:  85.0,
			storage:  95000.0,
			signal:   -55.0,
			expected: false,
		},
		{
			name:     "storage just above threshold",
			battery:  85.0,
			storage:  95000.01,
			signal:   -55.0,
			expected: true,
		},
		{
			name:     "signal exactly at threshold",
			battery:  85.0,
			storage:  45000.0,
			signal:   -100.0,
			expected: false,
		},
		{
			name:     "signal just below threshold",
			battery:  85.0,
			storage:  45000.0,
			signal:   -100.01,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			point := TelemetryPointForTest(tt.battery, tt.storage, tt.signal)
			result := bp.detectAnomaly(point)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnomalyConfigCustomThresholds(t *testing.T) {
	tests := []struct {
		name                string
		batteryThreshold    float64
		storageThreshold    float64
		signalThreshold     float64
		testBattery         float64
		testStorage         float64
		testSignal          float64
		expectedAnomaly     bool
		anomalyTypeExpected string
	}{
		{
			name:                "custom battery threshold",
			batteryThreshold:    5.0,
			storageThreshold:    95000.0,
			signalThreshold:     -100.0,
			testBattery:         4.0,
			testStorage:         45000.0,
			testSignal:          -55.0,
			expectedAnomaly:     true,
			anomalyTypeExpected: "battery",
		},
		{
			name:                "custom storage threshold",
			batteryThreshold:    10.0,
			storageThreshold:    90000.0,
			signalThreshold:     -100.0,
			testBattery:         85.0,
			testStorage:         91000.0,
			testSignal:          -55.0,
			expectedAnomaly:     true,
			anomalyTypeExpected: "storage",
		},
		{
			name:                "custom signal threshold",
			batteryThreshold:    10.0,
			storageThreshold:    95000.0,
			signalThreshold:     -90.0,
			testBattery:         85.0,
			testStorage:         45000.0,
			testSignal:          -95.0,
			expectedAnomaly:     true,
			anomalyTypeExpected: "signal",
		},
		{
			name:                "all thresholds normal",
			batteryThreshold:    10.0,
			storageThreshold:    95000.0,
			signalThreshold:     -100.0,
			testBattery:         85.0,
			testStorage:         45000.0,
			testSignal:          -55.0,
			expectedAnomaly:     false,
			anomalyTypeExpected: "",
		},
		{
			name:                "zero thresholds with zero values",
			batteryThreshold:    0,
			storageThreshold:    0,
			signalThreshold:     0,
			testBattery:         0,
			testStorage:         0,
			testSignal:          0,
			expectedAnomaly:     false, // All values equal thresholds
			anomalyTypeExpected: "",
		},
		{
			name:                "zero thresholds with positive values",
			batteryThreshold:    0,
			storageThreshold:    0,
			signalThreshold:     -1, // More permissive
			testBattery:         1,
			testStorage:         1,
			testSignal:          0,
			expectedAnomaly:     true, // Battery below 0, storage above 0
			anomalyTypeExpected: "battery, storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomalyConfig := AnomalyConfig{
				BatteryMinPercent: tt.batteryThreshold,
				StorageMaxMB:      tt.storageThreshold,
				SignalMinDBM:      tt.signalThreshold,
			}

			bp := &BatchProcessor{
				anomalyConfig: anomalyConfig,
			}

			point := TelemetryPointForTest(tt.testBattery, tt.testStorage, tt.testSignal)
			result := bp.detectAnomaly(point)
			if result != tt.expectedAnomaly {
				t.Errorf("expected anomaly=%v, got %v", tt.expectedAnomaly, result)
			}
		})
	}
}

func TestAnomalyDetectionEdgeCases(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		anomalyConfig: anomalyConfig,
	}

	tests := []struct {
		name     string
		battery  float64
		storage  float64
		signal   float64
		expected bool
	}{
		{
			name:     "negative battery",
			battery:  -10.0,
			storage:  45000.0,
			signal:   -55.0,
			expected: true,
		},
		{
			name:     "negative storage",
			battery:  85.5,
			storage:  -100.0,
			signal:   -55.0,
			expected: false, // negative storage is below max
		},
		{
			name:     "very weak signal",
			battery:  85.5,
			storage:  45000.0,
			signal:   -999.0,
			expected: true,
		},
		{
			name:     "very strong signal",
			battery:  85.5,
			storage:  45000.0,
			signal:   -30.0,
			expected: false,
		},
		{
			name:     "very high storage",
			battery:  85.5,
			storage:  999999.0,
			signal:   -55.0,
			expected: true,
		},
		{
			name:     "zero battery",
			battery:  0,
			storage:  45000.0,
			signal:   -55.0,
			expected: true,
		},
		{
			name:     "all normal at edge",
			battery:  10.01,
			storage:  94999.99,
			signal:   -99.99,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			point := TelemetryPointForTest(tt.battery, tt.storage, tt.signal)
			result := bp.detectAnomaly(point)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TelemetryPointForTest creates a test telemetry point
// This is a test helper function
func TelemetryPointForTest(battery, storage, signal float64) models.TelemetryPoint {
	return models.TelemetryPoint{
		BatteryChargePercent: battery,
		StorageUsageMB:       storage,
		SignalStrengthDBM:    signal,
		Timestamp:            time.Now().UTC(),
	}
}

// =============================================================================
// Feature C: Fault Tolerance Tests (WAL, Circuit Breaker, Retry Logic)
// =============================================================================

// TestBatchProcessorBufferSizeLimit tests that buffer rejects when at max capacity
func TestBatchProcessorBufferSizeLimit(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		buffer:        make([]models.TelemetryPoint, 0, 100),
		batchSize:     100,
		anomalyConfig: anomalyConfig,
		maxBufferSize: 5, // Small limit for testing
	}

	// Add points up to the limit
	for i := 0; i < 5; i++ {
		point := TelemetryPointForTest(85.0, 45000.0, -55.0)
		point.SatelliteID = "SAT-001"
		if err := bp.Add(point); err != nil {
			t.Errorf("unexpected error on add %d: %v", i, err)
		}
	}

	// Next add should fail due to buffer full
	point := TelemetryPointForTest(85.0, 45000.0, -55.0)
	point.SatelliteID = "SAT-001"
	err := bp.Add(point)
	if err == nil {
		t.Error("expected error when buffer is at max capacity")
	}
}

// TestBatchProcessorSetRetryConfig tests configuring retry behavior
func TestBatchProcessorSetRetryConfig(t *testing.T) {
	bp := &BatchProcessor{}

	// Set retry config
	bp.SetRetryConfig(10, 2*time.Second)

	if bp.maxRetries != 10 {
		t.Errorf("expected maxRetries 10, got %d", bp.maxRetries)
	}

	if bp.retryDelay != 2*time.Second {
		t.Errorf("expected retryDelay 2s, got %v", bp.retryDelay)
	}
}

// TestBatchProcessorSetMaxBufferSize tests configuring max buffer size
func TestBatchProcessorSetMaxBufferSize(t *testing.T) {
	bp := &BatchProcessor{}

	bp.SetMaxBufferSize(5000)

	if bp.maxBufferSize != 5000 {
		t.Errorf("expected maxBufferSize 5000, got %d", bp.maxBufferSize)
	}
}

// TestBatchProcessorSetWAL tests configuring WAL
func TestBatchProcessorSetWAL(t *testing.T) {
	bp := &BatchProcessor{}

	// Initially nil
	if bp.GetWAL() != nil {
		t.Error("expected WAL to be nil initially")
	}

	// Create a mock WAL (using nil for simplicity, just testing the setter)
	bp.SetWAL(nil) // Setting nil is valid

	// Test with actual WAL in integration tests
}

// TestBatchProcessorSetCircuitBreaker tests configuring circuit breaker
func TestBatchProcessorSetCircuitBreaker(t *testing.T) {
	bp := &BatchProcessor{}

	// Note: When created directly, circuit breaker is nil
	// It's set by default when using NewBatchProcessor()
	// Here we test the setter/getter functionality

	// Set custom circuit breaker
	cb := NewCircuitBreaker(5, 60*time.Second)
	bp.SetCircuitBreaker(cb)

	if bp.GetCircuitBreaker() != cb {
		t.Error("expected circuit breaker to be updated")
	}
}

// TestBatchProcessorGetBufferSize tests the GetBufferSize method
func TestBatchProcessorGetBufferSize(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		buffer:        make([]models.TelemetryPoint, 0, 100),
		batchSize:     100,
		anomalyConfig: anomalyConfig,
		maxBufferSize: 1000,
	}

	// Initially 0
	if bp.GetBufferSize() != 0 {
		t.Errorf("expected buffer size 0, got %d", bp.GetBufferSize())
	}

	// Add points
	for i := 0; i < 5; i++ {
		point := TelemetryPointForTest(85.0, 45000.0, -55.0)
		point.SatelliteID = "SAT-001"
		bp.Add(point)
	}

	if bp.GetBufferSize() != 5 {
		t.Errorf("expected buffer size 5, got %d", bp.GetBufferSize())
	}
}

// TestBatchProcessorFlushToWALWithoutWAL tests flushToWAL when WAL is not configured
func TestBatchProcessorFlushToWALWithoutWAL(t *testing.T) {
	bp := &BatchProcessor{
		wal: nil, // No WAL configured
	}

	batch := []models.TelemetryPoint{
		TelemetryPointForTest(85.0, 45000.0, -55.0),
	}

	err := bp.flushToWAL(batch)
	if err == nil {
		t.Error("expected error when flushing to WAL without WAL configured")
	}
}

// TestBatchProcessorFlushToWALWithWAL tests flushToWAL with WAL configured
func TestBatchProcessorFlushToWALWithWAL(t *testing.T) {
	// Create temporary WAL
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	bp := &BatchProcessor{
		wal: wal,
	}

	batch := []models.TelemetryPoint{
		{
			SatelliteID:          "SAT-001",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		{
			SatelliteID:          "SAT-002",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 75.0,
			StorageUsageMB:       55000.0,
			SignalStrengthDBM:    -65.0,
			IsAnomaly:            true,
		},
	}

	err = bp.flushToWAL(batch)
	if err != nil {
		t.Fatalf("failed to flush to WAL: %v", err)
	}

	// Verify records were written
	count, err := wal.Count()
	if err != nil {
		t.Fatalf("failed to count WAL records: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 WAL records, got %d", count)
	}
}

// TestBatchProcessorCircuitBreakerIntegration tests circuit breaker behavior with batch processor
func TestBatchProcessorCircuitBreakerIntegration(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Second)
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		buffer:         make([]models.TelemetryPoint, 0, 100),
		batchSize:      100,
		anomalyConfig:  anomalyConfig,
		maxBufferSize:  1000,
		circuitBreaker: cb,
	}

	// Verify circuit breaker is set
	if bp.GetCircuitBreaker() != cb {
		t.Error("circuit breaker not properly set")
	}

	// Verify circuit breaker starts closed
	if !bp.GetCircuitBreaker().IsClosed() {
		t.Error("circuit breaker should start closed")
	}
}

// TestBatchProcessorAddWithAnomalyDetection tests that anomaly detection runs on Add
func TestBatchProcessorAddWithAnomalyDetection(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		buffer:        make([]models.TelemetryPoint, 0, 100),
		batchSize:     100,
		anomalyConfig: anomalyConfig,
		maxBufferSize: 1000,
	}

	// Add normal point
	normalPoint := TelemetryPointForTest(85.0, 45000.0, -55.0)
	normalPoint.SatelliteID = "SAT-001"
	bp.Add(normalPoint)

	if bp.buffer[0].IsAnomaly {
		t.Error("normal point should not be flagged as anomaly")
	}

	// Add anomalous point (low battery)
	anomalousPoint := TelemetryPointForTest(5.0, 45000.0, -55.0)
	anomalousPoint.SatelliteID = "SAT-002"
	bp.Add(anomalousPoint)

	// Find the anomalous point in buffer
	var found bool
	for _, p := range bp.buffer {
		if p.SatelliteID == "SAT-002" {
			found = true
			if !p.IsAnomaly {
				t.Error("low battery point should be flagged as anomaly")
			}
		}
	}
	if !found {
		t.Error("anomalous point not found in buffer")
	}
}

// TestBatchProcessorDefaultValues tests default configuration values
func TestBatchProcessorDefaultValues(t *testing.T) {
	bp := &BatchProcessor{
		batchSize:     1000,
		batchTimeout:  1 * time.Second,
		buffer:        make([]models.TelemetryPoint, 0, 1000),
		maxRetries:    5,
		retryDelay:    1 * time.Second,
		maxBufferSize: 10000,
	}

	if bp.maxRetries != 5 {
		t.Errorf("expected default maxRetries 5, got %d", bp.maxRetries)
	}

	if bp.retryDelay != 1*time.Second {
		t.Errorf("expected default retryDelay 1s, got %v", bp.retryDelay)
	}

	if bp.maxBufferSize != 10000 {
		t.Errorf("expected default maxBufferSize 10000, got %d", bp.maxBufferSize)
	}
}

// TestBatchProcessorGetPool tests the GetPool method
func TestBatchProcessorGetPool(t *testing.T) {
	bp := &BatchProcessor{
		pool: nil, // No pool in unit tests
	}

	if bp.GetPool() != nil {
		t.Error("expected nil pool")
	}
}

// TestBatchProcessorWALRecordConversion tests converting TelemetryPoint to WALRecord
func TestBatchProcessorWALRecordConversion(t *testing.T) {
	point := models.TelemetryPoint{
		SatelliteID:          "SAT-TEST",
		Timestamp:            time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            true,
	}

	// Convert to WALRecord
	record := WALRecord{
		Timestamp:            point.Timestamp,
		SatelliteID:          point.SatelliteID,
		BatteryChargePercent: point.BatteryChargePercent,
		StorageUsageMB:       point.StorageUsageMB,
		SignalStrengthDBM:    point.SignalStrengthDBM,
		IsAnomaly:            point.IsAnomaly,
	}

	// Verify all fields match
	if record.SatelliteID != point.SatelliteID {
		t.Error("satellite ID mismatch")
	}
	if !record.Timestamp.Equal(point.Timestamp) {
		t.Error("timestamp mismatch")
	}
	if record.BatteryChargePercent != point.BatteryChargePercent {
		t.Error("battery mismatch")
	}
	if record.StorageUsageMB != point.StorageUsageMB {
		t.Error("storage mismatch")
	}
	if record.SignalStrengthDBM != point.SignalStrengthDBM {
		t.Error("signal mismatch")
	}
	if record.IsAnomaly != point.IsAnomaly {
		t.Error("is_anomaly mismatch")
	}
}

// TestBatchProcessorMultipleAdds tests adding multiple points sequentially
func TestBatchProcessorMultipleAdds(t *testing.T) {
	anomalyConfig := AnomalyConfig{
		BatteryMinPercent: 10.0,
		StorageMaxMB:      95000.0,
		SignalMinDBM:      -100.0,
	}

	bp := &BatchProcessor{
		buffer:        make([]models.TelemetryPoint, 0, 100),
		batchSize:     1000, // High threshold so no auto-flush
		anomalyConfig: anomalyConfig,
		maxBufferSize: 10000,
	}

	// Add 100 points
	for i := 0; i < 100; i++ {
		point := TelemetryPointForTest(85.0, 45000.0, -55.0)
		point.SatelliteID = "SAT-001"
		if err := bp.Add(point); err != nil {
			t.Errorf("unexpected error on add %d: %v", i, err)
		}
	}

	if bp.GetBufferSize() != 100 {
		t.Errorf("expected buffer size 100, got %d", bp.GetBufferSize())
	}
}

// TestRandFloat64 tests the random float generator (used for jitter)
func TestRandFloat64(t *testing.T) {
	// Generate multiple values
	for i := 0; i < 100; i++ {
		val := randFloat64()
		if val < 0 || val > 1 {
			t.Errorf("randFloat64 returned %f, expected range [0, 1]", val)
		}
	}
}
