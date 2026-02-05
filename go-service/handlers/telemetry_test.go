package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"orbitstream/models"
	"orbitstream/test"
)

func init() {
	// Set Gin to test mode to suppress output
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(handler *TelemetryHandler) *gin.Engine {
	router := gin.New()
	router.POST("/telemetry", handler.HandleTelemetry)
	router.POST("/telemetry/batch", handler.HandleTelemetryBatch)
	router.GET("/health", handler.HealthCheck)
	return router
}

// HandleTelemetry Tests

func TestHandleTelemetryValid(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := test.NewTestTelemetryPoint()
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	var response models.TelemetryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "accepted" {
		t.Errorf("expected status 'accepted', got '%s'", response.Status)
	}
	if response.SatelliteID != point.SatelliteID {
		t.Errorf("expected satellite_id '%s', got '%s'", point.SatelliteID, response.SatelliteID)
	}

	if mockBP.GetAddCallCount() != 1 {
		t.Errorf("expected 1 call to Add, got %d", mockBP.GetAddCallCount())
	}
}

func TestHandleTelemetryInvalidJSON(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("expected error field in response")
	}

	if mockBP.GetAddCallCount() != 0 {
		t.Errorf("expected 0 calls to Add, got %d", mockBP.GetAddCallCount())
	}
}

func TestHandleTelemetryMissingFields(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	// Missing required fields - Gin's JSON binding uses zero values for missing fields
	// This test documents the current behavior: partial data is accepted
	incompleteData := map[string]interface{}{
		"satellite_id": "SAT-0001",
		// Missing battery, storage, signal - they will be zero values
	}
	jsonData, _ := json.Marshal(incompleteData)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Current behavior: accepted with zero values for missing numeric fields
	// Additional validation would need to be added for stricter checking
	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202 (current behavior), got %d", w.Code)
	}

	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 1 {
		t.Fatalf("expected 1 point added, got %d", len(addedPoints))
	}

	// Verify the point has zero values for missing fields
	if addedPoints[0].BatteryChargePercent != 0 {
		t.Errorf("expected BatteryChargePercent to be 0 (missing field), got %f", addedPoints[0].BatteryChargePercent)
	}
	if addedPoints[0].StorageUsageMB != 0 {
		t.Errorf("expected StorageUsageMB to be 0 (missing field), got %f", addedPoints[0].StorageUsageMB)
	}
	if addedPoints[0].SignalStrengthDBM != 0 {
		t.Errorf("expected SignalStrengthDBM to be 0 (missing field), got %f", addedPoints[0].SignalStrengthDBM)
	}
}

func TestHandleTelemetryZeroTimestamp(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	// Create point without timestamp (zero value)
	point := models.TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: 85.5,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
		// Timestamp is zero
	}
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	// Verify the point was added to the batch
	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 1 {
		t.Fatalf("expected 1 point added, got %d", len(addedPoints))
	}

	// The timestamp should be set by the handler
	if addedPoints[0].Timestamp.IsZero() {
		t.Error("expected timestamp to be set by handler")
	}
}

func TestHandleTelemetryAddsToBatch(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := test.NewTestTelemetryPoint()
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 1 {
		t.Fatalf("expected 1 point added to batch, got %d", len(addedPoints))
	}

	if addedPoints[0].SatelliteID != point.SatelliteID {
		t.Errorf("expected satellite_id '%s', got '%s'", point.SatelliteID, addedPoints[0].SatelliteID)
	}
}

func TestHandleTelemetryMultiple(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	satellites := []string{"SAT-0001", "SAT-0002", "SAT-0003"}

	for _, satID := range satellites {
		point := test.NewTestTelemetryPointWithSatelliteID(satID)
		jsonData, _ := json.Marshal(point)

		req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", w.Code)
		}
	}

	if mockBP.GetAddCallCount() != 3 {
		t.Errorf("expected 3 calls to Add, got %d", mockBP.GetAddCallCount())
	}

	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 3 {
		t.Fatalf("expected 3 points added, got %d", len(addedPoints))
	}
}

// HandleTelemetryBatch Tests

func TestHandleTelemetryBatchValid(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	points := []models.TelemetryPoint{
		test.NewTestTelemetryPointWithSatelliteID("SAT-0001"),
		test.NewTestTelemetryPointWithSatelliteID("SAT-0002"),
		test.NewTestTelemetryPointWithSatelliteID("SAT-0003"),
	}
	jsonData, _ := json.Marshal(points)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	var response models.TelemetryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "accepted" {
		t.Errorf("expected status 'accepted', got '%s'", response.Status)
	}
	if response.Count != 3 {
		t.Errorf("expected count 3, got %d", response.Count)
	}

	if mockBP.GetAddCallCount() != 3 {
		t.Errorf("expected 3 calls to Add, got %d", mockBP.GetAddCallCount())
	}
}

func TestHandleTelemetryBatchEmpty(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	points := []models.TelemetryPoint{}
	jsonData, _ := json.Marshal(points)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	var response models.TelemetryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("expected count 0, got %d", response.Count)
	}
}

func TestHandleTelemetryBatchInvalidJSON(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleTelemetryBatchZeroTimestamps(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	// Create points without timestamps
	now := time.Now().UTC()
	points := []models.TelemetryPoint{
		{
			SatelliteID:          "SAT-0001",
			BatteryChargePercent: 85.5,
			StorageUsageMB:       45000.0,
			SignalStrengthDBM:    -55.0,
		},
		{
			SatelliteID:          "SAT-0002",
			BatteryChargePercent: 90.0,
			StorageUsageMB:       40000.0,
			SignalStrengthDBM:    -50.0,
		},
	}
	jsonData, _ := json.Marshal(points)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Record time before request to check timestamp range
	timeBeforeRequest := time.Now().UTC()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 2 {
		t.Fatalf("expected 2 points added, got %d", len(addedPoints))
	}

	// Verify timestamps were set (should be close to now)
	for i, point := range addedPoints {
		if point.Timestamp.Before(timeBeforeRequest) || point.Timestamp.After(now.Add(time.Second)) {
			t.Errorf("point %d: timestamp not set correctly: %v (expected around %v)", i, point.Timestamp, now)
		}
	}
}

func TestHandleTelemetryBatchLarge(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	// Create a batch of 100 points
	points := make([]models.TelemetryPoint, 100)
	for i := 0; i < 100; i++ {
		points[i] = test.NewTestTelemetryPointWithSatelliteID("SAT-" + string(rune('0'+i)))
	}
	jsonData, _ := json.Marshal(points)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	if mockBP.GetAddCallCount() != 100 {
		t.Errorf("expected 100 calls to Add, got %d", mockBP.GetAddCallCount())
	}
}

// HealthCheck Tests

func TestHealthCheckReturns200(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response.Status)
	}
}

func TestHealthCheckTimestampPresent(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Timestamp.IsZero() {
		t.Error("expected timestamp to be set, got zero")
	}

	// Verify timestamp is recent (within last second)
	if time.Since(response.Timestamp) > time.Second {
		t.Error("timestamp is not recent")
	}
}

func TestHealthCheckContentType(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
	}
}

// Edge Cases

func TestHandleTelemetryWithAnomalyFlag(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := test.NewAnomalousTelemetryPoint()
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	addedPoints := mockBP.GetAddedPoints()
	if len(addedPoints) != 1 {
		t.Fatalf("expected 1 point added, got %d", len(addedPoints))
	}

	// The anomaly flag should be preserved
	if !addedPoints[0].IsAnomaly {
		t.Error("expected IsAnomaly to be true")
	}
}

func TestHandleTelemetryWithNegativeBattery(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := models.TelemetryPoint{
		SatelliteID:          "SAT-0001",
		BatteryChargePercent: -10.0,
		StorageUsageMB:       45000.0,
		SignalStrengthDBM:    -55.0,
	}
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still be accepted (validation happens elsewhere)
	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}
}

func TestHandleTelemetryWithContentTypeHeader(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := test.NewTestTelemetryPoint()
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type 'application/json; charset=utf-8', got '%s'", contentType)
	}
}

func TestHandleTelemetryBatchSinglePoint(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	// Single point in batch endpoint
	point := test.NewTestTelemetryPoint()
	points := []models.TelemetryPoint{point}
	jsonData, _ := json.Marshal(points)

	req, _ := http.NewRequest("POST", "/telemetry/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}

	if mockBP.GetAddCallCount() != 1 {
		t.Errorf("expected 1 call to Add, got %d", mockBP.GetAddCallCount())
	}
}

func TestHandleTelemetryWithoutContentType(t *testing.T) {
	mockBP := test.NewMockBatchProcessor()
	handler := NewTelemetryHandler(mockBP)
	router := setupTestRouter(handler)

	point := test.NewTestTelemetryPoint()
	jsonData, _ := json.Marshal(point)

	req, _ := http.NewRequest("POST", "/telemetry", bytes.NewBuffer(jsonData))
	// No Content-Type header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Gin should still parse it correctly
	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}
}
