package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"orbitstream/db"
	"orbitstream/models"
)

// BatchProcessorInterface defines the interface for batch processing
// This allows for mocking in tests
type BatchProcessorInterface interface {
	Add(point models.TelemetryPoint) error
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
	if err := h.batchProcessor.Add(point); err != nil {
		// Buffer full - return 503 Service Unavailable
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("Buffer full: %v", err),
		})
		return
	}

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
	acceptedCount := 0
	for i := range points {
		if points[i].Timestamp.IsZero() {
			points[i].Timestamp = now
		}
		if err := h.batchProcessor.Add(points[i]); err != nil {
			// Log error but continue processing other points
			fmt.Printf("Error adding point %d: %v\n", i, err)
		} else {
			acceptedCount++
		}
	}

	c.JSON(http.StatusAccepted, models.TelemetryResponse{
		Status: "accepted",
		Count:  acceptedCount,
	})
}

// HealthCheck returns the health status of the service
// It checks database connectivity and WAL status
func (h *TelemetryHandler) HealthCheck(c *gin.Context) {
	status := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Type assert to get access to the real batch processor methods
	realBatchProcessor, ok := h.batchProcessor.(*db.BatchProcessor)
	httpStatus := http.StatusOK

	if ok {
		// Check database connectivity
		ctx, cancel := context.WithTimeout(c, 1*time.Second)
		defer cancel()

		pool := realBatchProcessor.GetPool()
		if pool != nil {
			err := pool.Ping(ctx)
			if err != nil {
				status.Status = "degraded"
				status.DatabaseStatus = "down"
				httpStatus = http.StatusServiceUnavailable
			} else {
				status.DatabaseStatus = "up"
			}
		}

		// Get WAL stats
		wal := realBatchProcessor.GetWAL()
		if wal != nil {
			status.WALSizeBytes = wal.Size()
			if count, err := wal.Count(); err == nil {
				status.WALRecordCount = count
			}
		}

		// Get buffer size
		status.BufferSize = realBatchProcessor.GetBufferSize()

		// Get circuit breaker state
		cb := realBatchProcessor.GetCircuitBreaker()
		if cb != nil {
			status.CircuitBreaker = cb.State().String()
		}
	}

	c.JSON(httpStatus, status)
}
