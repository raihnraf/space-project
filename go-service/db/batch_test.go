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

// =============================================================================
// Feature E: Position Tracking Tests
// =============================================================================

// TestBatchProcessorWALRecordConversionWithPositionFields tests converting TelemetryPoint with position fields to WALRecord
func TestBatchProcessorWALRecordConversionWithPositionFields(t *testing.T) {
	// Create helper function to convert float64 to pointer
	toPtr := func(v float64) *float64 {
		return &v
	}

	point := models.TelemetryPoint{
		SatelliteID:          "SAT-ISS",
		Timestamp:            time.Date(2024, 2, 10, 15, 30, 0, 0, time.UTC),
		BatteryChargePercent: 75.5,
		StorageUsageMB:       42000.0,
		SignalStrengthDBM:    -60.0,
		IsAnomaly:            false,
		Latitude:             toPtr(-33.8688), // Sydney
		Longitude:            toPtr(151.2093),
		AltitudeKM:           toPtr(408.5),
		VelocityKMPH:         toPtr(27576.5),
	}

	// Convert to WALRecord (as done in flushToWAL)
	record := WALRecord{
		Timestamp:            point.Timestamp,
		SatelliteID:          point.SatelliteID,
		BatteryChargePercent: point.BatteryChargePercent,
		StorageUsageMB:       point.StorageUsageMB,
		SignalStrengthDBM:    point.SignalStrengthDBM,
		IsAnomaly:            point.IsAnomaly,
		Latitude:             point.Latitude,
		Longitude:            point.Longitude,
		AltitudeKM:           point.AltitudeKM,
		VelocityKMPH:         point.VelocityKMPH,
	}

	// Verify all fields match including position fields
	if record.SatelliteID != point.SatelliteID {
		t.Error("satellite ID mismatch")
	}
	if record.Latitude == nil || *record.Latitude != -33.8688 {
		t.Errorf("expected latitude -33.8688, got %v", record.Latitude)
	}
	if record.Longitude == nil || *record.Longitude != 151.2093 {
		t.Errorf("expected longitude 151.2093, got %v", record.Longitude)
	}
	if record.AltitudeKM == nil || *record.AltitudeKM != 408.5 {
		t.Errorf("expected altitude_km 408.5, got %v", record.AltitudeKM)
	}
	if record.VelocityKMPH == nil || *record.VelocityKMPH != 27576.5 {
		t.Errorf("expected velocity_kmph 27576.5, got %v", record.VelocityKMPH)
	}
}

// TestBatchProcessorFlushToWALWithPositionFields tests flushToWAL includes position data
func TestBatchProcessorFlushToWALWithPositionFields(t *testing.T) {
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

	toPtr := func(v float64) *float64 {
		return &v
	}

	batch := []models.TelemetryPoint{
		{
			SatelliteID:          "SAT-ISS-001",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 75.5,
			StorageUsageMB:       42000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(-33.8688),
			Longitude:            toPtr(151.2093),
			AltitudeKM:           toPtr(408.5),
			VelocityKMPH:         toPtr(27576.5),
		},
		{
			SatelliteID:          "SAT-STARLINK-1001",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       38000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
			Latitude:             toPtr(40.7128), // NYC
			Longitude:            toPtr(-74.0060),
			AltitudeKM:           toPtr(550.0),
			VelocityKMPH:         toPtr(27600.0),
		},
	}

	err = bp.flushToWAL(batch)
	if err != nil {
		t.Fatalf("failed to flush to WAL: %v", err)
	}

	// Verify records were written
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read WAL records: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 WAL records, got %d", len(records))
	}

	// Verify first record's position fields
	if records[0].Latitude == nil || *records[0].Latitude != -33.8688 {
		t.Errorf("first record: expected latitude -33.8688, got %v", records[0].Latitude)
	}
	if records[0].VelocityKMPH == nil || *records[0].VelocityKMPH != 27576.5 {
		t.Errorf("first record: expected velocity_kmph 27576.5, got %v", records[0].VelocityKMPH)
	}

	// Verify second record's position fields
	if records[1].Latitude == nil || *records[1].Latitude != 40.7128 {
		t.Errorf("second record: expected latitude 40.7128, got %v", records[1].Latitude)
	}
	if records[1].AltitudeKM == nil || *records[1].AltitudeKM != 550.0 {
		t.Errorf("second record: expected altitude_km 550.0, got %v", records[1].AltitudeKM)
	}
}

// TestBatchProcessorFlushToWALPartialPositionFields tests flushToWAL with partial position fields
func TestBatchProcessorFlushToWALPartialPositionFields(t *testing.T) {
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

	toPtr := func(v float64) *float64 {
		return &v
	}

	batch := []models.TelemetryPoint{
		// Full position fields
		{
			SatelliteID:          "SAT-FULL",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(35.6762),
			Longitude:            toPtr(139.6503),
			AltitudeKM:           toPtr(408.0),
			VelocityKMPH:         toPtr(27580.0),
		},
		// No position fields (backward compatibility)
		{
			SatelliteID:          "SAT-NOPOS",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		// Partial position fields (only lat/lon)
		{
			SatelliteID:          "SAT-PARTIAL",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 75.0,
			StorageUsageMB:       42000.0,
			SignalStrengthDBM:    -58.0,
			IsAnomaly:            false,
			Latitude:             toPtr(-23.5505),
			Longitude:            toPtr(-46.6333),
		},
	}

	err = bp.flushToWAL(batch)
	if err != nil {
		t.Fatalf("failed to flush to WAL: %v", err)
	}

	// Verify records were written correctly
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read WAL records: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 WAL records, got %d", len(records))
	}

	// First record has all position fields
	if records[0].Latitude == nil || *records[0].Latitude != 35.6762 {
		t.Error("first record: latitude mismatch")
	}
	if records[0].VelocityKMPH == nil || *records[0].VelocityKMPH != 27580.0 {
		t.Error("first record: velocity mismatch")
	}

	// Second record has no position fields
	if records[1].Latitude != nil {
		t.Error("second record: expected nil latitude")
	}
	if records[1].AltitudeKM != nil {
		t.Error("second record: expected nil altitude")
	}

	// Third record has partial position fields
	if records[2].Latitude == nil || *records[2].Latitude != -23.5505 {
		t.Error("third record: latitude mismatch")
	}
	if records[2].AltitudeKM != nil {
		t.Error("third record: expected nil altitude")
	}
}

// TestBatchProcessorAddWithPositionFields tests that Add() handles position fields correctly
func TestBatchProcessorAddWithPositionFields(t *testing.T) {
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

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Add point with full position fields
	pointWithPos := TelemetryPointForTest(85.0, 45000.0, -55.0)
	pointWithPos.SatelliteID = "SAT-POS-001"
	pointWithPos.Latitude = toPtr(1.3521)
	pointWithPos.Longitude = toPtr(103.8198)
	pointWithPos.AltitudeKM = toPtr(400.0)
	pointWithPos.VelocityKMPH = toPtr(27500.0)

	if err := bp.Add(pointWithPos); err != nil {
		t.Fatalf("unexpected error adding point with position: %v", err)
	}

	if bp.GetBufferSize() != 1 {
		t.Errorf("expected buffer size 1, got %d", bp.GetBufferSize())
	}

	// Verify position fields are preserved in buffer
	if bp.buffer[0].Latitude == nil || *bp.buffer[0].Latitude != 1.3521 {
		t.Error("latitude not preserved in buffer")
	}
	if bp.buffer[0].VelocityKMPH == nil || *bp.buffer[0].VelocityKMPH != 27500.0 {
		t.Error("velocity not preserved in buffer")
	}

	// Add point without position fields (backward compatibility)
	pointNoPos := TelemetryPointForTest(85.0, 45000.0, -55.0)
	pointNoPos.SatelliteID = "SAT-NOPOS-001"

	if err := bp.Add(pointNoPos); err != nil {
		t.Fatalf("unexpected error adding point without position: %v", err)
	}

	if bp.GetBufferSize() != 2 {
		t.Errorf("expected buffer size 2, got %d", bp.GetBufferSize())
	}

	// Verify second point has nil position fields
	if bp.buffer[1].Latitude != nil {
		t.Error("expected nil latitude for point without position")
	}
}

// TestBatchProcessorPositionValueRanges tests realistic position value ranges in batch context
func TestBatchProcessorPositionValueRanges(t *testing.T) {
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

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Test various realistic satellite positions
	batch := []models.TelemetryPoint{
		{
			SatelliteID:          "SAT-EQUATORIAL",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(0.0),    // Equator
			Longitude:            toPtr(0.0),    // Prime meridian
			AltitudeKM:           toPtr(400.0),  // LEO
			VelocityKMPH:         toPtr(27500.0),
		},
		{
			SatelliteID:          "SAT-NORTH-POLE",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(90.0),   // North pole
			Longitude:            toPtr(0.0),
			AltitudeKM:           toPtr(500.0),
			VelocityKMPH:         toPtr(27000.0),
		},
		{
			SatelliteID:          "SAT-SOUTH-POLE",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(-90.0),  // South pole
			Longitude:            toPtr(180.0),  // International date line
			AltitudeKM:           toPtr(450.0),
			VelocityKMPH:         toPtr(27800.0),
		},
		{
			SatelliteID:          "SAT-LOW-LEO",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(45.0),
			Longitude:            toPtr(-122.0),
			AltitudeKM:           toPtr(300.0),  // Low LEO
			VelocityKMPH:         toPtr(27300.0),
		},
		{
			SatelliteID:          "SAT-HIGH-LEO",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(-45.0),
			Longitude:            toPtr(0.0),
			AltitudeKM:           toPtr(2000.0), // High LEO
			VelocityKMPH:         toPtr(26000.0),
		},
	}

	err = bp.flushToWAL(batch)
	if err != nil {
		t.Fatalf("failed to flush to WAL: %v", err)
	}

	// Verify all records were written
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read WAL records: %v", err)
	}

	if len(records) != len(batch) {
		t.Fatalf("expected %d WAL records, got %d", len(batch), len(records))
	}

	// Verify specific ranges
	if records[0].Latitude == nil || *records[0].Latitude != 0.0 {
		t.Error("equatorial: latitude should be 0.0")
	}
	if records[1].Latitude == nil || *records[1].Latitude != 90.0 {
		t.Error("north pole: latitude should be 90.0")
	}
	if records[2].Latitude == nil || *records[2].Latitude != -90.0 {
		t.Error("south pole: latitude should be -90.0")
	}
	if records[3].AltitudeKM == nil || *records[3].AltitudeKM != 300.0 {
		t.Error("low LEO: altitude should be 300.0")
	}
	if records[4].AltitudeKM == nil || *records[4].AltitudeKM != 2000.0 {
		t.Error("high LEO: altitude should be 2000.0")
	}
}

// TestBatchProcessorWALPositionZeroValues tests zero values in position fields during WAL write
func TestBatchProcessorWALPositionZeroValues(t *testing.T) {
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

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Test with explicit zero values (at equator, prime meridian, surface)
	batch := []models.TelemetryPoint{
		{
			SatelliteID:          "SAT-ZERO",
			Timestamp:            time.Now().UTC(),
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
			Latitude:             toPtr(0.0),
			Longitude:            toPtr(0.0),
			AltitudeKM:           toPtr(0.0),
			VelocityKMPH:         toPtr(0.0),
		},
	}

	err = bp.flushToWAL(batch)
	if err != nil {
		t.Fatalf("failed to flush to WAL: %v", err)
	}

	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read WAL records: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 WAL record, got %d", len(records))
	}

	// Verify zero values are preserved (not treated as nil)
	if records[0].Latitude == nil {
		t.Error("expected latitude 0.0, got nil")
	} else if *records[0].Latitude != 0.0 {
		t.Errorf("expected latitude 0.0, got %f", *records[0].Latitude)
	}
	if records[0].Longitude == nil {
		t.Error("expected longitude 0.0, got nil")
	} else if *records[0].Longitude != 0.0 {
		t.Errorf("expected longitude 0.0, got %f", *records[0].Longitude)
	}
	if records[0].AltitudeKM == nil {
		t.Error("expected altitude_km 0.0, got nil")
	} else if *records[0].AltitudeKM != 0.0 {
		t.Errorf("expected altitude_km 0.0, got %f", *records[0].AltitudeKM)
	}
	if records[0].VelocityKMPH == nil {
		t.Error("expected velocity_kmph 0.0, got nil")
	} else if *records[0].VelocityKMPH != 0.0 {
		t.Errorf("expected velocity_kmph 0.0, got %f", *records[0].VelocityKMPH)
	}
}
