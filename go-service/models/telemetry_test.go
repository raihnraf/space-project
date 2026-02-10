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
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Format(time.RFC3339),
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
		Timestamp: now.Format(time.RFC3339),
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
	// Parse the timestamp string to verify it's valid
	parsedTime, err := time.Parse(time.RFC3339, unmarshaled.Timestamp)
	if err != nil {
		t.Errorf("failed to parse timestamp: %v", err)
	}
	// Verify the parsed time is close to now (within 1 second)
	if parsedTime.Sub(now) > time.Second || now.Sub(parsedTime) > time.Second {
		t.Errorf("expected Timestamp close to %v, got %v", now, parsedTime)
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

// ============================================================================
// Feature E: Position Tracking Tests
// ============================================================================

func TestTelemetryPointWithPositionFields(t *testing.T) {
	// Test JSON unmarshal with position fields
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 85.5,
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0,
		"timestamp": "2024-01-15T10:30:00Z",
		"latitude": -6.2088,
		"longitude": 106.8456,
		"altitude_km": 420.5,
		"velocity_kmph": 27543.21
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON with position fields: %v", err)
	}

	// Verify basic fields
	if point.SatelliteID != "SAT-0001" {
		t.Errorf("expected SatelliteID 'SAT-0001', got '%s'", point.SatelliteID)
	}

	// Verify position fields
	if point.Latitude == nil {
		t.Error("expected Latitude to be set, got nil")
	} else if *point.Latitude != -6.2088 {
		t.Errorf("expected Latitude -6.2088, got %f", *point.Latitude)
	}

	if point.Longitude == nil {
		t.Error("expected Longitude to be set, got nil")
	} else if *point.Longitude != 106.8456 {
		t.Errorf("expected Longitude 106.8456, got %f", *point.Longitude)
	}

	if point.AltitudeKM == nil {
		t.Error("expected AltitudeKM to be set, got nil")
	} else if *point.AltitudeKM != 420.5 {
		t.Errorf("expected AltitudeKM 420.5, got %f", *point.AltitudeKM)
	}

	if point.VelocityKMPH == nil {
		t.Error("expected VelocityKMPH to be set, got nil")
	} else if *point.VelocityKMPH != 27543.21 {
		t.Errorf("expected VelocityKMPH 27543.21, got %f", *point.VelocityKMPH)
	}
}

func TestTelemetryPointWithoutPositionFields(t *testing.T) {
	// Test backward compatibility: JSON without position fields
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
		t.Fatalf("failed to unmarshal JSON without position fields: %v", err)
	}

	// Verify basic fields work
	if point.SatelliteID != "SAT-0001" {
		t.Errorf("expected SatelliteID 'SAT-0001', got '%s'", point.SatelliteID)
	}

	// Verify position fields are nil (backward compatibility)
	if point.Latitude != nil {
		t.Errorf("expected Latitude to be nil for backward compatibility, got %v", *point.Latitude)
	}
	if point.Longitude != nil {
		t.Errorf("expected Longitude to be nil for backward compatibility, got %v", *point.Longitude)
	}
	if point.AltitudeKM != nil {
		t.Errorf("expected AltitudeKM to be nil for backward compatibility, got %v", *point.AltitudeKM)
	}
	if point.VelocityKMPH != nil {
		t.Errorf("expected VelocityKMPH to be nil for backward compatibility, got %v", *point.VelocityKMPH)
	}
}

func TestTelemetryPointPartialPositionFields(t *testing.T) {
	// Test with only some position fields (e.g., only latitude and longitude)
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 85.5,
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0,
		"latitude": -6.2088,
		"longitude": 106.8456
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON with partial position fields: %v", err)
	}

	// Verify provided position fields are set
	if point.Latitude == nil || *point.Latitude != -6.2088 {
		t.Errorf("expected Latitude -6.2088, got %v", point.Latitude)
	}
	if point.Longitude == nil || *point.Longitude != 106.8456 {
		t.Errorf("expected Longitude 106.8456, got %v", point.Longitude)
	}

	// Verify omitted position fields are nil
	if point.AltitudeKM != nil {
		t.Errorf("expected AltitudeKM to be nil when omitted, got %v", *point.AltitudeKM)
	}
	if point.VelocityKMPH != nil {
		t.Errorf("expected VelocityKMPH to be nil when omitted, got %v", *point.VelocityKMPH)
	}
}

func TestTelemetryPointPositionFieldsRoundTrip(t *testing.T) {
	// Test JSON marshal/unmarshal round-trip with position fields
	lat := -6.2088
	lon := 106.8456
	alt := 420.5
	vel := 27543.21

	point := TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		Timestamp:            time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		IsAnomaly:            false,
		Latitude:             &lat,
		Longitude:            &lon,
		AltitudeKM:           &alt,
		VelocityKMPH:         &vel,
	}

	data, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("failed to marshal JSON with position fields: %v", err)
	}

	var unmarshaled TelemetryPoint
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal marshaled JSON with position fields: %v", err)
	}

	// Verify all fields round-trip correctly
	if unmarshaled.SatelliteID != point.SatelliteID {
		t.Errorf("SatelliteID round-trip failed: expected '%s', got '%s'", point.SatelliteID, unmarshaled.SatelliteID)
	}
	if unmarshaled.BatteryChargePercent != point.BatteryChargePercent {
		t.Errorf("BatteryChargePercent round-trip failed")
	}

	// Verify position fields
	if unmarshaled.Latitude == nil || *unmarshaled.Latitude != lat {
		t.Errorf("Latitude round-trip failed: expected %f, got %v", lat, unmarshaled.Latitude)
	}
	if unmarshaled.Longitude == nil || *unmarshaled.Longitude != lon {
		t.Errorf("Longitude round-trip failed: expected %f, got %v", lon, unmarshaled.Longitude)
	}
	if unmarshaled.AltitudeKM == nil || *unmarshaled.AltitudeKM != alt {
		t.Errorf("AltitudeKM round-trip failed: expected %f, got %v", alt, unmarshaled.AltitudeKM)
	}
	if unmarshaled.VelocityKMPH == nil || *unmarshaled.VelocityKMPH != vel {
		t.Errorf("VelocityKMPH round-trip failed: expected %f, got %v", vel, unmarshaled.VelocityKMPH)
	}
}

func TestTelemetryPointPositionValueRanges(t *testing.T) {
	// Test that position values are within expected ranges for LEO satellites
	lat := -6.2088   // Valid: -90 to 90
	lon := 106.8456  // Valid: -180 to 180
	alt := 420.5     // Valid: 300-2000 km for LEO
	vel := 27543.21  // Valid: ~27000 km/h for orbital velocity

	point := TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		Latitude:             &lat,
		Longitude:            &lon,
		AltitudeKM:           &alt,
		VelocityKMPH:         &vel,
	}

	// Verify position value ranges
	if point.Latitude != nil && (*point.Latitude < -90 || *point.Latitude > 90) {
		t.Errorf("Latitude out of valid range [-90, 90]: %f", *point.Latitude)
	}
	if point.Longitude != nil && (*point.Longitude < -180 || *point.Longitude > 180) {
		t.Errorf("Longitude out of valid range [-180, 180]: %f", *point.Longitude)
	}
	// LEO satellites typically 300-2000 km altitude
	if point.AltitudeKM != nil && (*point.AltitudeKM < 0 || *point.AltitudeKM > 50000) {
		t.Errorf("AltitudeKM seems unrealistic: %f", *point.AltitudeKM)
	}
	// Orbital velocity typically ~27000 km/h
	if point.VelocityKMPH != nil && (*point.VelocityKMPH < 0 || *point.VelocityKMPH > 50000) {
		t.Errorf("VelocityKMPH seems unrealistic: %f", *point.VelocityKMPH)
	}
}

func TestTelemetryPointPositionZeroValues(t *testing.T) {
	// Test zero/null position values are handled correctly
	jsonData := `{
		"satellite_id": "SAT-0001",
		"battery_charge_percent": 85.5,
		"storage_usage_mb": 45000.0,
		"signal_strength_dbm": -55.0,
		"latitude": 0.0,
		"longitude": 0.0,
		"altitude_km": 0.0,
		"velocity_kmph": 0.0
	}`

	var point TelemetryPoint
	err := json.Unmarshal([]byte(jsonData), &point)

	if err != nil {
		t.Fatalf("failed to unmarshal JSON with zero position values: %v", err)
	}

	// Zero values are valid (e.g., equator, zero altitude reference)
	if point.Latitude == nil || *point.Latitude != 0.0 {
		t.Errorf("expected Latitude 0.0, got %v", point.Latitude)
	}
	if point.Longitude == nil || *point.Longitude != 0.0 {
		t.Errorf("expected Longitude 0.0, got %v", point.Longitude)
	}
	if point.AltitudeKM == nil || *point.AltitudeKM != 0.0 {
		t.Errorf("expected AltitudeKM 0.0, got %v", point.AltitudeKM)
	}
	if point.VelocityKMPH == nil || *point.VelocityKMPH != 0.0 {
		t.Errorf("expected VelocityKMPH 0.0, got %v", point.VelocityKMPH)
	}
}
