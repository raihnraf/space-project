package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WAL represents a Write Ahead Log for persistent buffering
// When the database is unavailable, telemetry data is written to the WAL
// and replayed when the database becomes available again.
type WAL struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

// WALRecord represents a single telemetry record in the WAL
// This is stored as JSON in the WAL file for easy inspection and debugging
type WALRecord struct {
	Timestamp            time.Time `json:"timestamp"`
	SatelliteID          string    `json:"satellite_id"`
	BatteryChargePercent float64   `json:"battery_charge_percent"`
	StorageUsageMB       float64   `json:"storage_usage_mb"`
	SignalStrengthDBM    float64   `json:"signal_strength_dbm"`
	IsAnomaly            bool      `json:"is_anomaly"`
	// Position tracking fields (nullable pointers for backward compatibility)
	Latitude             *float64  `json:"latitude,omitempty"`
	Longitude            *float64  `json:"longitude,omitempty"`
	AltitudeKM           *float64  `json:"altitude_km,omitempty"`
	VelocityKMPH         *float64  `json:"velocity_kmph,omitempty"`
}

// NewWAL creates a new WAL instance
// It creates the directory for the WAL file if it doesn't exist
// If the WAL file already exists, it will be opened and existing records can be read
func NewWAL(walPath string) (*WAL, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(walPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(walPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	return &WAL{
		filePath: walPath,
		file:     file,
	}, nil
}

// Write appends a record to the WAL in JSON format
// Each record is written as a single line for easy parsing
// Thread-safe: uses mutex to prevent concurrent writes
func (w *WAL) Write(record WALRecord) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Marshal record to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal WAL record: %w", err)
	}

	// Append newline and write to file
	data = append(data, '\n')
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write WAL record: %w", err)
	}

	// Sync to disk immediately for durability
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL file: %w", err)
	}

	return nil
}

// ReadAll reads all records from the WAL
// This opens the file in read-only mode and parses each line as JSON
// Thread-safe: uses mutex to prevent concurrent reads
func (w *WAL) ReadAll() ([]WALRecord, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Close existing file and reopen in read mode
	if w.file != nil {
		w.file.Close()
	}

	data, err := os.ReadFile(w.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []WALRecord{}, nil // No WAL file means no records
		}
		return nil, fmt.Errorf("failed to read WAL file: %w", err)
	}

	// Reopen file in append mode for future writes
	w.file, err = os.OpenFile(w.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to reopen WAL file: %w", err)
	}

	// Parse each line as a JSON record
	var records []WALRecord
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var record WALRecord
		if err := json.Unmarshal(line, &record); err != nil {
			// Log error but continue parsing other records
			fmt.Printf("Warning: failed to parse WAL record: %v\n", err)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// Clear removes all records from the WAL by truncating the file
// This should be called after successfully replaying all records to the database
// Thread-safe: uses mutex to prevent concurrent operations
func (w *WAL) Clear() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Close existing file
	if w.file != nil {
		w.file.Close()
	}

	// Truncate file
	if err := os.Truncate(w.filePath, 0); err != nil {
		return fmt.Errorf("failed to truncate WAL file: %w", err)
	}

	// Reopen file in append mode
	file, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen WAL file after clear: %w", err)
	}

	w.file = file
	return nil
}

// Size returns the current WAL file size in bytes
// This can be used to monitor WAL growth and trigger rotation if needed
func (w *WAL) Size() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(w.filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Count returns the number of records in the WAL
// This is a convenience method that calls ReadAll and counts the records
// For better performance with large WALs, consider maintaining an in-memory counter
func (w *WAL) Count() (int, error) {
	records, err := w.ReadAll()
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// Close closes the WAL file
// This should be called when shutting down the service
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// splitLines splits byte data into lines
// This is a helper function for ReadAll
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0

	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}

	// Add last line if it doesn't end with newline
	if start < len(data) {
		lines = append(lines, data[start:])
	}

	return lines
}
