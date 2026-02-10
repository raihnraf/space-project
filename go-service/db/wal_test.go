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

// TestWALRecordWithPositionFields tests that position fields are preserved
func TestWALRecordWithPositionFields(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Create helper function to convert float64 to pointer
	toPtr := func(v float64) *float64 {
		return &v
	}

	original := WALRecord{
		Timestamp:            testTime,
		SatelliteID:          "SAT-ISS-001",
		BatteryChargePercent: 75.5,
		StorageUsageMB:       42000.0,
		SignalStrengthDBM:    -60.0,
		IsAnomaly:            false,
		Latitude:             toPtr(-33.8688), // Sydney, Australia
		Longitude:            toPtr(151.2093),
		AltitudeKM:           toPtr(408.5),
		VelocityKMPH:         toPtr(27576.5),
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

	// Verify all position fields
	if read.Latitude == nil {
		t.Error("expected latitude to be set, got nil")
	} else if *read.Latitude != -33.8688 {
		t.Errorf("expected latitude -33.8688, got %f", *read.Latitude)
	}

	if read.Longitude == nil {
		t.Error("expected longitude to be set, got nil")
	} else if *read.Longitude != 151.2093 {
		t.Errorf("expected longitude 151.2093, got %f", *read.Longitude)
	}

	if read.AltitudeKM == nil {
		t.Error("expected altitude_km to be set, got nil")
	} else if *read.AltitudeKM != 408.5 {
		t.Errorf("expected altitude_km 408.5, got %f", *read.AltitudeKM)
	}

	if read.VelocityKMPH == nil {
		t.Error("expected velocity_kmph to be set, got nil")
	} else if *read.VelocityKMPH != 27576.5 {
		t.Errorf("expected velocity_kmph 27576.5, got %f", *read.VelocityKMPH)
	}
}

// TestWALRecordWithoutPositionFields tests backward compatibility (nil position fields)
func TestWALRecordWithoutPositionFields(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Create record without position fields (backward compatibility)
	original := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-LEGACY-001",
		BatteryChargePercent: 85.0,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
		// Position fields left as nil (default for pointers)
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

	// Verify position fields are nil
	if read.Latitude != nil {
		t.Errorf("expected latitude to be nil, got %f", *read.Latitude)
	}
	if read.Longitude != nil {
		t.Errorf("expected longitude to be nil, got %f", *read.Longitude)
	}
	if read.AltitudeKM != nil {
		t.Errorf("expected altitude_km to be nil, got %f", *read.AltitudeKM)
	}
	if read.VelocityKMPH != nil {
		t.Errorf("expected velocity_kmph to be nil, got %f", *read.VelocityKMPH)
	}

	// Verify other fields are still preserved
	if read.SatelliteID != "SAT-LEGACY-001" {
		t.Errorf("expected satellite ID SAT-LEGACY-001, got %s", read.SatelliteID)
	}
}

// TestWALRecordPartialPositionFields tests records with only some position fields set
func TestWALRecordPartialPositionFields(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Create record with only latitude and longitude set
	original := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-PARTIAL-001",
		BatteryChargePercent: 85.0,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
		Latitude:             toPtr(40.7128), // New York City
		Longitude:            toPtr(-74.0060),
		// AltitudeKM and VelocityKMPH left as nil
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

	// Verify set fields
	if read.Latitude == nil || *read.Latitude != 40.7128 {
		t.Errorf("expected latitude 40.7128, got %v", read.Latitude)
	}
	if read.Longitude == nil || *read.Longitude != -74.0060 {
		t.Errorf("expected longitude -74.0060, got %v", read.Longitude)
	}

	// Verify nil fields
	if read.AltitudeKM != nil {
		t.Errorf("expected altitude_km to be nil, got %f", *read.AltitudeKM)
	}
	if read.VelocityKMPH != nil {
		t.Errorf("expected velocity_kmph to be nil, got %f", *read.VelocityKMPH)
	}
}

// TestWALRecordPositionFieldsPersistence tests position fields persist across close/reopen
func TestWALRecordPositionFieldsPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Create and write with position fields
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}

	original := WALRecord{
		Timestamp:            time.Date(2024, 2, 10, 15, 30, 0, 0, time.UTC),
		SatelliteID:          "SAT-PERSIST-001",
		BatteryChargePercent: 65.5,
		StorageUsageMB:       38000.0,
		SignalStrengthDBM:    -70.0,
		IsAnomaly:            true,
		Latitude:             toPtr(1.3521),    // Singapore
		Longitude:            toPtr(103.8198),
		AltitudeKM:           toPtr(550.0),
		VelocityKMPH:         toPtr(27600.0),
	}

	if err := wal.Write(original); err != nil {
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
		t.Fatalf("expected 1 record after reopen, got %d", len(records))
	}

	read := records[0]

	// Verify all position fields persisted
	if read.Latitude == nil || *read.Latitude != 1.3521 {
		t.Errorf("expected latitude 1.3521, got %v", read.Latitude)
	}
	if read.Longitude == nil || *read.Longitude != 103.8198 {
		t.Errorf("expected longitude 103.8198, got %v", read.Longitude)
	}
	if read.AltitudeKM == nil || *read.AltitudeKM != 550.0 {
		t.Errorf("expected altitude_km 550.0, got %v", read.AltitudeKM)
	}
	if read.VelocityKMPH == nil || *read.VelocityKMPH != 27600.0 {
		t.Errorf("expected velocity_kmph 27600.0, got %v", read.VelocityKMPH)
	}
}

// TestWALRecordPositionValueRanges tests realistic position value ranges
func TestWALRecordPositionValueRanges(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	toPtr := func(v float64) *float64 {
		return &v
	}

	testCases := []struct {
		name      string
		record    WALRecord
		shouldErr bool
	}{
		{
			name: "valid equatorial position",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-EQ-001",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(0.0),
				Longitude:            toPtr(0.0),
				AltitudeKM:           toPtr(400.0),
				VelocityKMPH:         toPtr(27500.0),
			},
			shouldErr: false,
		},
		{
			name: "valid north pole position",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-NP-001",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(90.0),
				Longitude:            toPtr(0.0),
				AltitudeKM:           toPtr(500.0),
				VelocityKMPH:         toPtr(27000.0),
			},
			shouldErr: false,
		},
		{
			name: "valid south pole position",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-SP-001",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(-90.0),
				Longitude:            toPtr(180.0),
				AltitudeKM:           toPtr(450.0),
				VelocityKMPH:         toPtr(27800.0),
			},
			shouldErr: false,
		},
		{
			name: "valid international date line position",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-IDL-001",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(45.0),
				Longitude:            toPtr(-180.0),
				AltitudeKM:           toPtr(420.0),
				VelocityKMPH:         toPtr(27650.0),
			},
			shouldErr: false,
		},
		{
			name: "low LEO altitude",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-LEO-LOW",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(0.0),
				Longitude:            toPtr(0.0),
				AltitudeKM:           toPtr(300.0),
				VelocityKMPH:         toPtr(27300.0),
			},
			shouldErr: false,
		},
		{
			name: "high LEO altitude",
			record: WALRecord{
				Timestamp:            time.Now().UTC(),
				SatelliteID:          "SAT-LEO-HIGH",
				BatteryChargePercent: 80.0,
				StorageUsageMB:       40000.0,
				SignalStrengthDBM:    -60.0,
				IsAnomaly:            false,
				Latitude:             toPtr(0.0),
				Longitude:            toPtr(0.0),
				AltitudeKM:           toPtr(2000.0),
				VelocityKMPH:         toPtr(26000.0),
			},
			shouldErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wal.Write(tc.record)
			if (err != nil) != tc.shouldErr {
				t.Errorf("Write() error = %v, shouldErr %v", err, tc.shouldErr)
			}
		})
	}

	// Verify all records were written correctly
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(records) != len(testCases) {
		t.Errorf("expected %d records, got %d", len(testCases), len(records))
	}
}

// TestWALRecordPositionZeroValues tests zero values in position fields
func TestWALRecordPositionZeroValues(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Test with explicit zero values (not nil)
	record := WALRecord{
		Timestamp:            time.Now().UTC(),
		SatelliteID:          "SAT-ZERO-001",
		BatteryChargePercent: 85.0,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		IsAnomaly:            false,
		Latitude:             toPtr(0.0), // Explicit zero at equator
		Longitude:            toPtr(0.0), // Explicit zero at prime meridian
		AltitudeKM:           toPtr(0.0), // Explicit zero (surface level)
		VelocityKMPH:         toPtr(0.0), // Explicit zero (stationary)
	}

	if err := wal.Write(record); err != nil {
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

	// Verify zero values are preserved (not treated as nil)
	if read.Latitude == nil {
		t.Error("expected latitude to be 0.0, got nil")
	} else if *read.Latitude != 0.0 {
		t.Errorf("expected latitude 0.0, got %f", *read.Latitude)
	}

	if read.Longitude == nil {
		t.Error("expected longitude to be 0.0, got nil")
	} else if *read.Longitude != 0.0 {
		t.Errorf("expected longitude 0.0, got %f", *read.Longitude)
	}

	if read.AltitudeKM == nil {
		t.Error("expected altitude_km to be 0.0, got nil")
	} else if *read.AltitudeKM != 0.0 {
		t.Errorf("expected altitude_km 0.0, got %f", *read.AltitudeKM)
	}

	if read.VelocityKMPH == nil {
		t.Error("expected velocity_kmph to be 0.0, got nil")
	} else if *read.VelocityKMPH != 0.0 {
		t.Errorf("expected velocity_kmph 0.0, got %f", *read.VelocityKMPH)
	}
}

// TestWALMultipleRecordsWithPositionFields tests multiple records with various position configurations
func TestWALMultipleRecordsWithPositionFields(t *testing.T) {
	tmpDir := t.TempDir()
	walPath := filepath.Join(tmpDir, "test.wal")

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("failed to create WAL: %v", err)
	}
	defer wal.Close()

	toPtr := func(v float64) *float64 {
		return &v
	}

	// Write a mix of records: with positions, without positions, partial positions
	records := []WALRecord{
		{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-FULL-POS",
			BatteryChargePercent: 80.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -60.0,
			IsAnomaly:            false,
			Latitude:             toPtr(35.6762),
			Longitude:            toPtr(139.6503),
			AltitudeKM:           toPtr(408.0),
			VelocityKMPH:         toPtr(27580.0),
		},
		{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-NO-POS",
			BatteryChargePercent: 85.0,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
			IsAnomaly:            false,
		},
		{
			Timestamp:            time.Now().UTC(),
			SatelliteID:          "SAT-PARTIAL-POS",
			BatteryChargePercent: 75.0,
			StorageUsageMB:       42000.0,
			SignalStrengthDBM:    -58.0,
			IsAnomaly:            false,
			Latitude:             toPtr(-23.5505),
			Longitude:            toPtr(-46.6333),
		},
	}

	for i, record := range records {
		if err := wal.Write(record); err != nil {
			t.Fatalf("failed to write record %d: %v", i, err)
		}
	}

	readRecords, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("failed to read records: %v", err)
	}

	if len(readRecords) != len(records) {
		t.Fatalf("expected %d records, got %d", len(records), len(readRecords))
	}

	// Verify first record has all position fields
	if readRecords[0].Latitude == nil || *readRecords[0].Latitude != 35.6762 {
		t.Error("first record: latitude mismatch")
	}
	if readRecords[0].VelocityKMPH == nil || *readRecords[0].VelocityKMPH != 27580.0 {
		t.Error("first record: velocity mismatch")
	}

	// Verify second record has no position fields (backward compatibility)
	if readRecords[1].Latitude != nil {
		t.Error("second record: expected nil latitude")
	}

	// Verify third record has partial position fields
	if readRecords[2].Latitude == nil || *readRecords[2].Latitude != -23.5505 {
		t.Error("third record: latitude mismatch")
	}
	if readRecords[2].AltitudeKM != nil {
		t.Error("third record: expected nil altitude")
	}
}
