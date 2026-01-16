package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"orbitstream/config"
	"orbitstream/db"
	"orbitstream/handlers"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database connection pool
	pool, err := db.NewConnectionPool(cfg.DBUrl, cfg.MaxConnections)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	// Initialize batch processor
	anomalyConfig := db.AnomalyConfig{
		BatteryMinPercent: cfg.AnomalyThresholdBattery,
		StorageMaxMB:      cfg.AnomalyThresholdStorage,
		SignalMinDBM:      cfg.AnomalyThresholdSignal,
	}

	batchProcessor := db.NewBatchProcessor(
		pool,
		cfg.BatchSize,
		cfg.BatchTimeout,
		anomalyConfig,
	)

	// Start batch processor background worker
	go batchProcessor.Start()

	// Setup HTTP router
	router := setupRouter(batchProcessor)

	// Configure HTTP server
	server := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server with graceful shutdown
	go func() {
		log.Printf("Starting OrbitStream ingestion service on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batchProcessor.Stop()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

func setupRouter(batchProcessor *db.BatchProcessor) *gin.Engine {
	router := gin.Default()

	telemetryHandler := handlers.NewTelemetryHandler(batchProcessor)

	// Health check
	router.GET("/health", telemetryHandler.HealthCheck)

	// Telemetry endpoints
	router.POST("/telemetry", telemetryHandler.HandleTelemetry)
	router.POST("/telemetry/batch", telemetryHandler.HandleTelemetryBatch)

	return router
}
