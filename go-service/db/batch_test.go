package db

import (
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
