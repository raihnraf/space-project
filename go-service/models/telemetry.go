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
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type TelemetryResponse struct {
	Status      string `json:"status"`
	SatelliteID string `json:"satellite_id,omitempty"`
	Count       int    `json:"count,omitempty"`
}
