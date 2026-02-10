package models

import "time"

type TelemetryPoint struct {
	SatelliteID          string    `json:"satellite_id" db:"satellite_id"`
	BatteryChargePercent float64   `json:"battery_charge_percent" db:"battery_charge_percent"`
	StorageUsageMB       float64   `json:"storage_usage_mb" db:"storage_usage_mb"`
	SignalStrengthDBM    float64   `json:"signal_strength_dbm" db:"signal_strength_dbm"`
	Timestamp            time.Time `json:"timestamp,omitempty" db:"time"`
	IsAnomaly            bool      `json:"is_anomaly,omitempty" db:"is_anomaly"`
}

type HealthResponse struct {
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
	DatabaseStatus string `json:"database_status,omitempty"`
	WALSizeBytes   int64  `json:"wal_size_bytes,omitempty"`
	WALRecordCount int    `json:"wal_record_count,omitempty"`
	BufferSize     int    `json:"buffer_size,omitempty"`
	CircuitBreaker string `json:"circuit_breaker,omitempty"`
}

type TelemetryResponse struct {
	Status      string `json:"status"`
	SatelliteID string `json:"satellite_id,omitempty"`
	Count       int    `json:"count,omitempty"`
}
