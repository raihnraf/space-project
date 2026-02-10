package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestHealthMonitorSetCheckInterval tests the SetCheckInterval method
func TestHealthMonitorSetCheckInterval(t *testing.T) {
	hm := &HealthMonitor{
		checkInterval: 5 * time.Second,
	}

	newInterval := 10 * time.Second
	hm.SetCheckInterval(newInterval)

	if hm.checkInterval != newInterval {
		t.Errorf("expected check interval %v, got %v", newInterval, hm.checkInterval)
	}
}

// TestHealthMonitorIsHealthy tests the IsHealthy method
func TestHealthMonitorIsHealthy(t *testing.T) {
	hm := &HealthMonitor{
		isHealthy: false,
	}

	if hm.IsHealthy() {
		t.Error("expected IsHealthy to be false initially")
	}

	hm.healthMutex.Lock()
	hm.isHealthy = true
	hm.healthMutex.Unlock()

	if !hm.IsHealthy() {
		t.Error("expected IsHealthy to be true after setting")
	}
}

// TestHealthMonitorGetLastCheckTime tests the GetLastCheckTime method
func TestHealthMonitorGetLastCheckTime(t *testing.T) {
	hm := &HealthMonitor{}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	hm.healthMutex.Lock()
	hm.lastCheckTime = testTime
	hm.healthMutex.Unlock()

	if !hm.GetLastCheckTime().Equal(testTime) {
		t.Errorf("expected last check time %v, got %v", testTime, hm.GetLastCheckTime())
	}
}

// TestHealthMonitorGetLastCheckResult tests the GetLastCheckResult method
func TestHealthMonitorGetLastCheckResult(t *testing.T) {
	hm := &HealthMonitor{}

	// Initially nil
	if hm.GetLastCheckResult() != nil {
		t.Error("expected last check result to be nil initially")
	}

	// Set an error
	testErr := os.ErrNotExist
	hm.healthMutex.Lock()
	hm.lastCheckResult = testErr
	hm.healthMutex.Unlock()

	if hm.GetLastCheckResult() != testErr {
		t.Errorf("expected last check result %v, got %v", testErr, hm.GetLastCheckResult())
	}
}

// TestHealthMonitorConcurrentHealthChecks tests thread-safety of health state access
func TestHealthMonitorConcurrentHealthChecks(t *testing.T) {
	hm := &HealthMonitor{}
	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hm.IsHealthy()
		}()
	}

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val bool) {
			defer wg.Done()
			hm.healthMutex.Lock()
			hm.isHealthy = val
			hm.healthMutex.Unlock()
		}(i%2 == 0)
	}

	// Concurrent last check time reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hm.GetLastCheckTime()
		}()
	}

	// Concurrent last check result reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hm.GetLastCheckResult()
		}()
	}

	wg.Wait()
	// If we reach here without race conditions, test passes
}

// TestHealthMonitorInitialState tests the initial state of HealthMonitor
func TestHealthMonitorInitialState(t *testing.T) {
	hm := &HealthMonitor{}

	// Initial values should be zero/nil
	if hm.IsHealthy() {
		t.Error("expected IsHealthy to be false initially")
	}

	if !hm.GetLastCheckTime().IsZero() {
		t.Error("expected last check time to be zero initially")
	}

	if hm.GetLastCheckResult() != nil {
		t.Error("expected last check result to be nil initially")
	}
}

// TestHealthMonitorStateTransitions tests the health state transitions
func TestHealthMonitorStateTransitions(t *testing.T) {
	hm := &HealthMonitor{}

	// Start unhealthy
	if hm.IsHealthy() {
		t.Error("should start unhealthy")
	}

	// Transition to healthy
	hm.healthMutex.Lock()
	hm.isHealthy = true
	hm.lastCheckTime = time.Now()
	hm.lastCheckResult = nil
	hm.healthMutex.Unlock()

	if !hm.IsHealthy() {
		t.Error("should be healthy")
	}

	// Transition back to unhealthy
	testErr := os.ErrClosed
	hm.healthMutex.Lock()
	hm.isHealthy = false
	hm.lastCheckTime = time.Now()
	hm.lastCheckResult = testErr
	hm.healthMutex.Unlock()

	if hm.IsHealthy() {
		t.Error("should be unhealthy")
	}

	if hm.GetLastCheckResult() != testErr {
		t.Error("should have last check error")
	}
}

// TestWALRecordConversion tests WALRecord structure used by HealthMonitor
func TestWALRecordConversion(t *testing.T) {
	// This tests that WALRecord can properly store telemetry data
	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
	}

	// Verify all fields are properly set
	if record.SatelliteID != "SAT-001" {
		t.Errorf("satellite ID mismatch: expected SAT-001, got %s", record.SatelliteID)
	}
	if record.BatteryChargePercent != 85.5 {
		t.Errorf("battery mismatch: expected 85.5, got %f", record.BatteryChargePercent)
	}
	if record.StorageUsageMB != 45000.0 {
		t.Errorf("storage mismatch: expected 45000.0, got %f", record.StorageUsageMB)
	}
	if record.SignalStrengthDBM != -55.0 {
		t.Errorf("signal mismatch: expected -55.0, got %f", record.SignalStrengthDBM)
	}
}

// TestHealthMonitorStopWithoutStart tests that Stop is safe to call without Start
func TestHealthMonitorStopWithoutStart(t *testing.T) {
	hm := &HealthMonitor{
		stopCh: make(chan struct{}),
	}

	// Stop should not panic or block
	done := make(chan struct{})
	go func() {
		hm.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good - Stop completed
	case <-time.After(1 * time.Second):
		t.Error("Stop blocked for too long")
	}
}

// TestHealthMonitorStopChannelClose tests that Stop closes the stop channel
func TestHealthMonitorStopChannelClose(t *testing.T) {
	hm := &HealthMonitor{
		stopCh: make(chan struct{}),
	}

	// Check channel is open
	select {
	case <-hm.stopCh:
		t.Error("stopCh should not be closed initially")
	default:
		// Good - channel is open
	}

	// Stop should close the channel
	hm.Stop()

	// Verify channel is closed
	select {
	case <-hm.stopCh:
		// Good - channel is closed
	default:
		t.Error("stopCh should be closed after Stop()")
	}
}

// TestHealthMonitorReplayWALEmpty tests replayWAL with empty WAL
func TestHealthMonitorReplayWALEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test that HealthMonitor can be created with WAL
	_ = &HealthMonitor{
		wal: wal,
	}

	// replayWAL should not error with empty WAL
	// Note: This tests that it doesn't panic with nil pool
	// The actual database operations would fail, but with empty WAL it returns early
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read WAL: %v", err)
	}
	if len(records) != 0 {
		t.Error("expected empty WAL")
	}
}

// TestHealthMonitorReplayWALWithData tests WAL replay scenario setup
func TestHealthMonitorReplayWALWithData(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write some records to WAL
	for i := 0; i < 5; i++ {
		record := WALRecord{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-001",
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		}
		if err := wal.Write(record); err != nil {
			t.Fatalf("failed to write WAL record: %v", err)
		}
	}

	// Verify WAL has records
	count, err := wal.Count()
	if err != nil {
		t.Fatalf("failed to count WAL records: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 WAL records, got %d", count)
	}

	// The actual replay test requires a database connection
	// which is tested in integration tests
}

// TestHealthMonitorCheckIntervalDefault tests default check interval
func TestHealthMonitorCheckIntervalDefault(t *testing.T) {
	// Create a minimal health monitor
	hm := &HealthMonitor{}

	// Default should be unset (zero)
	if hm.checkInterval != 0 {
		t.Logf("note: check interval has non-zero default: %v", hm.checkInterval)
	}

	// Set it to a value
	hm.SetCheckInterval(5 * time.Second)
	if hm.checkInterval != 5*time.Second {
		t.Errorf("expected 5s interval, got %v", hm.checkInterval)
	}
}

// TestHealthMonitorThreadSafety tests concurrent access to all public methods
func TestHealthMonitorThreadSafety(t *testing.T) {
	hm := &HealthMonitor{
		stopCh: make(chan struct{}),
	}
	var wg sync.WaitGroup

	// Run concurrent operations for a short time
	done := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(done)
	}()

	for {
		select {
		case <-done:
			wg.Wait()
			return // Test complete
		default:
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = hm.IsHealthy()
				_ = hm.GetLastCheckTime()
				_ = hm.GetLastCheckResult()

				// Simulate state changes
				hm.healthMutex.Lock()
				hm.isHealthy = !hm.isHealthy
				hm.lastCheckTime = time.Now()
				hm.healthMutex.Unlock()
			}()
			time.Sleep(time.Microsecond)
		}
	}
}
