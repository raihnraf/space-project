package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTelemetryPointJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 85.5,
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0,
		"timestamp": "2024-01-15T10:30:00Z"
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if point.SatelliteID != "SAT-0001" {
		t.Errorf("expected SatelliteID 'SAT-0001', got '%s'", point.SatelliteID)
	}
	if point.BatteryChargePercent != 85.5 {
		t.Errorf("expected BatteryChargePercent 85.5, got %f", point.BatteryChargePercent)
	}
	if point.StorageUsageMB != 45000.0 {
		t.Errorf("expected StorageUsageMB 45000.0, got %f", point.StorageUsageMB)
	}
	if point.SignalStrengthDBM != -55.0 {
		t.Errorf("expected SignalStrengthDBM -55.0, got %f", point.SignalStrengthDBM)
	}
}

func TestTelemetryPointJSONMarshal(t *testing.T) {
	point := TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		Timestamp:            time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		IsAnomaly:            false,
	}

	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var unmarshaled TelemetryPoint
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal marshaled JSON: %v", err)
	}

	if unmarshaled.SatelliteID != point.SatelliteID {
		t.Errorf("round-trip failed: expected SatelliteID '%s', got '%s'", point.SatelliteID, unmarshaled.SatelliteID)
	}
	if unmarshaled.BatteryChargePercent != point.BatteryChargePercent {
		t.Errorf("round-trip failed: expected BatteryChargePercent %f, got %f", point.BatteryChargePercent, unmarshaled.BatteryChargePercent)
	}
}

func TestTelemetryPointWithoutTimestamp(t *testing.T) {
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 85.5,
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if point.SatelliteID != "SAT-0001" {
		t.Errorf("expected SatelliteID 'SAT-0001', got '%s'", point.SatelliteID)
	}

	// Timestamp should be zero when not provided
	if !point.Timestamp.IsZero() {
		t.Errorf("expected zero timestamp, got %v", point.Timestamp)
	}
}

func TestTelemetryPointWithAnomaly(t *testing.T) {
	point := TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 5.0,
		StorageUsageMB:       98000.0,
		SignalStrengthDBM:    -115.0,
		Timestamp:            time.Now(),
		IsAnomaly:            true,
	}

	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var unmarshaled TelemetryPoint
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal marshaled JSON: %v", err)
	}

	if !unmarshaled.IsAnomaly {
		t.Errorf("expected IsAnomaly to be true, got false")
	}
}

func TestTelemetryPointNegativeValues(t *testing.T) {
	// Test that negative values can be parsed (even if semantically invalid)
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": -10.0,
		"storage_usage_mb": -100.0,
		"signal_strength_dbm": -55.0
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON with negative values: %v", err)
	}

	if point.BatteryChargePercent != -10.0 {
		t.Errorf("expected BatteryChargePercent -10.0, got %f", point.BatteryChargePercent)
	}
	if point.StorageUsageMB != -100.0 {
		t.Errorf("expected StorageUsageMB -100.0, got %f", point.StorageUsageMB)
	}
}

func TestTelemetryPointExtremeValues(t *testing.T) {
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 999.99,
		"storage_usage_mb": 999999.99,
		"signal_strength_dbm": -999.99
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON with extreme values: %v", err)
	}

	if point.BatteryChargePercent != 999.99 {
		t.Errorf("expected BatteryChargePercent 999.99, got %f", point.BatteryChargePercent)
	}
	if point.StorageUsageMB != 999999.99 {
		t.Errorf("expected StorageUsageMB 999999.99, got %f", point.StorageUsageMB)
	}
	if point.SignalStrengthDBM != -999.99 {
		t.Errorf("expected SignalStrengthDBM -999.99, got %f", point.SignalStrengthDBM)
	}
}

func TestTelemetryPointInvalidJSON(t *testing.T) {
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": "not_a_number",
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestHealthResponseStructure(t *testing.T) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if decoded["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", decoded["status"])
	}
	if _, ok := decoded["timestamp"]; !ok {
		t.Error("expected timestamp field in JSON")
	}
}

func TestHealthResponseJSON(t *testing.T) {
	now := time.Now().UTC()
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: now,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var unmarshaled HealthResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if unmarshaled.Status != "healthy" {
		t.Errorf("expected Status 'healthy', got '%s'", unmarshaled.Status)
	}
	if !unmarshaled.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, unmarshaled.Timestamp)
	}
}

func TestTelemetryResponseStructure(t *testing.T) {
	response := TelemetryResponse{
		Status:      "accepted",
		SatelliteID: "SAT-0001",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if decoded["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got %v", decoded["status"])
	}
	if decoded["satellite_id"] != "SAT-0001" {
		t.Errorf("expected satellite_id 'SAT-0001', got %v", decoded["satellite_id"])
	}
}

func TestTelemetryResponseWithCount(t *testing.T) {
	response := TelemetryResponse{
		Status: "accepted",
		Count:  100,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	var unmarshaled TelemetryResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if unmarshaled.Status != "accepted" {
		t.Errorf("expected Status 'accepted', got '%s'", unmarshaled.Status)
	}
	if unmarshaled.Count != 100 {
		t.Errorf("expected Count 100, got %d", unmarshaled.Count)
	}
	if unmarshaled.SatelliteID != "" {
		t.Errorf("expected empty SatelliteID, got '%s'", unmarshaled.SatelliteID)
	}
}

func TestTelemetryResponseJSONRoundTrip(t *testing.T) {
	testCases := []TelemetryResponse{
		{
			Status:      "accepted",
			SatelliteID: "SAT-0001",
		},
		{
			Status: "accepted",
			Count:  50,
		},
		{
			Status:      "accepted",
			SatelliteID: "SAT-9999",
			Count:       1,
		},
	}

	for _, tc := range testCases {
		data, err := json.Marshal(tc)
		if err != nil {
			t.Errorf("failed to marshal %v: %v", tc, err)
			continue
		}

		var unmarshaled TelemetryResponse
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Errorf("failed to unmarshal %v: %v", tc, err)
			continue
		}

		if unmarshaled.Status != tc.Status {
			t.Errorf("Status mismatch: expected '%s', got '%s'", tc.Status, unmarshaled.Status)
		}
		if unmarshaled.SatelliteID != tc.SatelliteID {
			t.Errorf("SatelliteID mismatch: expected '%s', got '%s'", tc.SatelliteID, unmarshaled.SatelliteID)
		}
		if unmarshaled.Count != tc.Count {
			t.Errorf("Count mismatch: expected %d, got %d", tc.Count, unmarshaled.Count)
		}
	}
}
