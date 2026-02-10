package db

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestWALWriteAndRead tests basic write and read operations
func TestWALWriteAndRead(t *testing.T) {
	// Create temporary directory for WAL
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write a record
	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
	}

	if err := wal.Write(record); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}

	// Read records
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}

	if records[0].SatelliteID != "SAT-001" {
		t.Errorf("expected satellite ID SAT-001, got %s", records[0].SatelliteID)
	}

	if records[0].BatteryChargePercent != 85.5 {
		t.Errorf("expected battery 85.5, got %f", records[0].BatteryChargePercent)
	}
}

// TestWALMultipleWrites tests writing multiple records
func TestWALMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write 10 records
	for i := 0; i < 10; i++ {
		record := WALRecord{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-001",
			BatteryChargePercent: float64(i) * 10.0,
			StorageUsageMB:       float64(i) * 1000.0,
			SignalStrengthDBM:    -50.0 - float64(i),
			IsAnomaly:            i%2 == 0,
		}
		if err := wal.Write(record); err != nil {
			t.Fatalf("failed to write record %d: %v", i, err)
		}
	}

	// Read all records
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(records) != 10 {
		t.Errorf("expected 10 records, got %d", len(records))
	}
}

// TestWALClear tests clearing the WAL
func TestWALClear(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write records
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
			t.Fatalf("failed to write record: %v", err)
		}
	}

	// Verify records exist
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}
	if len(records) != 5 {
		t.Errorf("expected 5 records before clear, got %d", len(records))
	}

	// Clear WAL
	if err := wal.Clear(); err != nil {
		t.Fatalf("failed to clear WAL: %v", err)
	}

	// Verify records are gone
	records, err = wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records after clear: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records after clear, got %d", len(records))
	}
}

// TestWALSize tests the Size method
func TestWALSize(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Initial size should be 0
	initialSize := wal.Size()
	if initialSize != 0 {
		t.Errorf("expected initial size 0, got %d", initialSize)
	}

	// Write a record
	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
	}
	if err := wal.Write(record); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}

	// Size should increase
	newSize := wal.Size()
	if newSize <= initialSize {
		t.Errorf("expected size to increase after write, got %d (was %d)", newSize, initialSize)
	}

	// Clear and verify size is back to 0
	wal.Clear()
	clearedSize := wal.Size()
	if clearedSize != 0 {
		t.Errorf("expected size 0 after clear, got %d", clearedSize)
	}
}

// TestWALCount tests the Count method
func TestWALCount(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Initial count should be 0
	count, err := wal.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected initial count 0, got %d", count)
	}

	// Write 3 records
	for i := 0; i < 3; i++ {
		record := WALRecord{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-001",
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		}
		if err := wal.Write(record); err != nil {
			t.Fatalf("failed to write record: %v", err)
		}
	}

	// Count should be 3
	count, err = wal.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

// TestWALPersistence tests that WAL data persists across close/reopen
func TestWALPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	// Create and write
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
	}
	if err := wal.Write(record); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}
	wal.Close()

	// Reopen and read
	wal2, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to reopen WAL: %v", err)
	}
	defer wal2.Close()

	records, err := wal2.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record after reopen, got %d", len(records))
	}

	if records[0].SatelliteID != "SAT-001" {
		t.Errorf("expected satellite ID SAT-001, got %s", records[0].SatelliteID)
	}
}

// TestWALConcurrentWrites tests thread-safety with concurrent writes
func TestWALConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Write from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	recordsPerGoroutine := 10

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < recordsPerGoroutine; i++ {
				record := WALRecord{
					Timestamp:            time.Now().UTC(),
					SatelliteID:          "SAT-001",
					BatteryChargePercent: float64(goroutineID*100 + i),
					StorageUsageMB:       float64(goroutineID*100 + i),
					SignalStrengthDBM:    -55.0,
					IsAnomaly:            false,
				}
				if err := wal.Write(record); err != nil {
					t.Errorf("goroutine %d failed to write: %v", goroutineID, err)
				}
			}
		}(g)
	}

	wg.Wait()

	// Verify all records were written
	count, err := wal.Count()
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}

	expectedCount := numGoroutines * recordsPerGoroutine
	if count != expectedCount {
		t.Errorf("expected %d records, got %d", expectedCount, count)
	}
}

// TestWALNewDirectory tests that NewWAL creates the directory if it doesn't exist
func TestWALNewDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "subdir", "nested", "test.wal")

	// Directory shouldn't exist yet
	if _, err := os.Stat(filepath.Dir(walPath)); !os.IsNotExist(err) {
		t.Fatalf("expected directory to not exist initially")
	}

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL with new directory: %v", err)
	}
	defer wal.Close()

	// Directory should now exist
	if _, err := os.Stat(filepath.Dir(walPath)); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}

	// File should exist
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		t.Error("expected WAL file to be created")
	}
}

// TestWALEmptyFile tests reading from an empty WAL file
func TestWALEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Read without writing
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read from empty WAL: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records from empty WAL, got %d", len(records))
	}
}

// TestWALRecordFields tests that all record fields are preserved
func TestWALRecordFields(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Create record with specific timestamp for consistent testing
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	original := WALRecord{
		Timestamp:            testTime,
		SatelliteID:          "SAT-TEST-999",
		BatteryChargePercent: 12.34,
		StorageUsageMB:       56789.01,
		SignalStrengthDBM:    -67.89,
		IsAnomaly:            true,
	}

	if err := wal.Write(original); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}

	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	read := records[0]

	if !read.Timestamp.Equal(original.Timestamp) {
		t.Errorf("timestamp mismatch: expected %v, got %v", original.Timestamp, read.Timestamp)
	}
	if read.SatelliteID != original.SatelliteID {
		t.Errorf("satellite ID mismatch: expected %s, got %s", original.SatelliteID, read.SatelliteID)
	}
	if read.BatteryChargePercent != original.BatteryChargePercent {
		t.Errorf("battery mismatch: expected %f, got %f", original.BatteryChargePercent, read.BatteryChargePercent)
	}
	if read.StorageUsageMB != original.StorageUsageMB {
		t.Errorf("storage mismatch: expected %f, got %f", original.StorageUsageMB, read.StorageUsageMB)
	}
	if read.SignalStrengthDBM != original.SignalStrengthDBM {
		t.Errorf("signal mismatch: expected %f, got %f", original.SignalStrengthDBM, read.SignalStrengthDBM)
	}
	if read.IsAnomaly != original.IsAnomaly {
		t.Errorf("is_anomaly mismatch: expected %v, got %v", original.IsAnomaly, read.IsAnomaly)
	}
}

// TestWALClose tests that Close properly closes the file
func TestWALClose(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	// Write before close
	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-001",
		BatteryChargePercent: 85.0,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
	}
	if err := wal.Write(record); err != nil {
		t.Fatalf("failed to write record: %v", err)
	}

	// Close should not error
	if err := wal.Close(); err != nil {
		t.Errorf("failed to close WAL: %v", err)
	}

	// Second close returns error because file is already closed
	// This is expected behavior - we just verify it doesn't panic
	_ = wal.Close()
}
