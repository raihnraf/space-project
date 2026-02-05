package test

import (
	"orbitstream/models"
	"sync"
	"time"
)

// MockBatchProcessor is a mock implementation of the batch processor for testing
type MockBatchProcessor struct {
	mu            sync.Mutex
	addedPoints   []models.TelemetryPoint
	flushCount    int
	addCallCount  int
	shouldError   bool
	anomalyResult bool
}

// NewMockBatchProcessor creates a new mock batch processor
func NewMockBatchProcessor() *MockBatchProcessor {
	return &MockBatchProcessor{
		addedPoints: make([]models.TelemetryPoint, 0),
	}
}

// Add simulates adding a point to the batch
func (m *MockBatchProcessor) Add(point models.TelemetryPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addCallCount++
	m.addedPoints = append(m.addedPoints, point)
}

// GetAddedPoints returns all points that were added
func (m *MockBatchProcessor) GetAddedPoints() []models.TelemetryPoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]models.TelemetryPoint{}, m.addedPoints...)
}

// GetAddCallCount returns the number of times Add was called
func (m *MockBatchProcessor) GetAddCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addCallCount
}

// Clear clears all added points
func (m *MockBatchProcessor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addedPoints = make([]models.TelemetryPoint, 0)
}

// SetShouldError sets whether Add should simulate an error
func (m *MockBatchProcessor) SetShouldError(shouldError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
}

// SetAnomalyResult sets the anomaly detection result
func (m *MockBatchProcessor) SetAnomalyResult(anomaly bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.anomalyResult = anomaly
}

// Start is a no-op for the mock
func (m *MockBatchProcessor) Start() {}

// Stop is a no-op for the mock
func (m *MockBatchProcessor) Stop() {}

// GetFlushCount returns the number of times flush was called
func (m *MockBatchProcessor) GetFlushCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.flushCount
}

// Flush simulates a flush operation
func (m *MockBatchProcessor) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushCount++
	m.addedPoints = make([]models.TelemetryPoint, 0)
}

// NewTestTelemetryPoint creates a test telemetry point with default values
func NewTestTelemetryPoint() models.TelemetryPoint {
	return models.TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		Timestamp:            time.Now().UTC(),
		IsAnomaly:            false,
	}
}

// NewTestTelemetryPointWithSatelliteID creates a test telemetry point with a specific satellite ID
func NewTestTelemetryPointWithSatelliteID(id string) models.TelemetryPoint {
	point := NewTestTelemetryPoint()
	point.SatelliteID = id
	return point
}

// NewTestTelemetryPointWithBattery creates a test telemetry point with a specific battery level
func NewTestTelemetryPointWithBattery(battery float64) models.TelemetryPoint {
	point := NewTestTelemetryPoint()
	point.BatteryChargePercent = battery
	return point
}

// NewAnomalousTelemetryPoint creates a test telemetry point that should be flagged as an anomaly
func NewAnomalousTelemetryPoint() models.TelemetryPoint {
	return models.TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 5.0,
		StorageUsageMB:       98000.0,
		SignalStrengthDBM:    -115.0,
		Timestamp:            time.Now().UTC(),
		IsAnomaly:            true,
	}
}
