package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"orbitstream/db"
	"orbitstream/models"
)

// BatchProcessorInterface defines the interface for batch processing
// This allows for mocking in tests
type BatchProcessorInterface interface {
	Add(point models.TelemetryPoint)
}

type TelemetryHandler struct {
	batchProcessor BatchProcessorInterface
}

func NewTelemetryHandler(bp BatchProcessorInterface) *TelemetryHandler {
	return &TelemetryHandler{
		batchProcessor: bp,
	}
}

// NewTelemetryHandlerWithDB creates a handler with the real database batch processor
func NewTelemetryHandlerWithDB(bp *db.BatchProcessor) *TelemetryHandler {
	return &TelemetryHandler{
		batchProcessor: bp,
	}
}

// HandleTelemetry handles a single telemetry point
func (h *TelemetryHandler) HandleTelemetry(c *gin.Context) {
	var point models.TelemetryPoint

	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set timestamp if not provided
	if point.Timestamp.IsZero() {
		point.Timestamp = time.Now().UTC()
	}

	// Add to batch (async processing)
	h.batchProcessor.Add(point)

	// Return immediately
	c.JSON(http.StatusAccepted, models.TelemetryResponse{
		Status:      "accepted",
		SatelliteID: point.SatelliteID,
	})
}

// HandleTelemetryBatch handles a batch of telemetry points
func (h *TelemetryHandler) HandleTelemetryBatch(c *gin.Context) {
	var points []models.TelemetryPoint

	if err := c.ShouldBindJSON(&points); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC()
	for i := range points {
		if points[i].Timestamp.IsZero() {
			points[i].Timestamp = now
		}
		h.batchProcessor.Add(points[i])
	}

	c.JSON(http.StatusAccepted, models.TelemetryResponse{
		Status: "accepted",
		Count:  len(points),
	})
}

// HealthCheck returns the health status of the service
func (h *TelemetryHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
	})
}
