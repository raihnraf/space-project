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

	// Configure retry and circuit breaker
	batchProcessor.SetRetryConfig(cfg.MaxRetries, cfg.RetryDelay)
	circuitBreaker := db.NewCircuitBreaker(cfg.CircuitBreakerThreshold, 30*time.Second)
	batchProcessor.SetCircuitBreaker(circuitBreaker)
	batchProcessor.SetMaxBufferSize(cfg.MaxBufferSize)

	// Initialize WAL (Write Ahead Log)
	wal, err := db.NewWAL(cfg.WALPath)
	if err != nil {
		log.Printf("WARNING: Failed to initialize WAL: %v", err)
		log.Printf("Data may be lost if database becomes unavailable")
	} else {
		batchProcessor.SetWAL(wal)
		log.Printf("WAL initialized at: %s", cfg.WALPath)

		// Check for existing WAL records on startup
		if count, err := wal.Count(); err == nil && count > 0 {
			log.Printf("Found %d existing WAL records - will be replayed when DB is healthy", count)
		}
	}

	// Start batch processor background worker
	go batchProcessor.Start()

	// Initialize and start health monitor
	var healthMonitor *db.HealthMonitor
	if wal != nil {
		healthMonitor = db.NewHealthMonitor(pool, wal, batchProcessor)
		healthMonitor.SetCheckInterval(5 * time.Second)
		healthMonitor.Start()
		log.Println("Health monitor started")
		defer healthMonitor.Stop()
	}

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
		log.Printf("Configuration:")
		log.Printf("  Batch Size: %d", cfg.BatchSize)
		log.Printf("  Batch Timeout: %v", cfg.BatchTimeout)
		log.Printf("  Max Retries: %d", cfg.MaxRetries)
		log.Printf("  Circuit Breaker Threshold: %d", cfg.CircuitBreakerThreshold)
		log.Printf("  Max Buffer Size: %d", cfg.MaxBufferSize)
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

	// Stop health monitor first
	if healthMonitor != nil {
		healthMonitor.Stop()
		log.Println("Health monitor stopped")
	}

	// Stop batch processor (triggers final flush)
	batchProcessor.Stop()
	log.Println("Batch processor stopped")

	// Close WAL
	if wal != nil {
		if err := wal.Close(); err != nil {
			log.Printf("Error closing WAL: %v", err)
		}
		log.Println("WAL closed")
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

func setupRouter(batchProcessor *db.BatchProcessor) *gin.Engine {
	router := gin.Default()

	telemetryHandler := handlers.NewTelemetryHandlerWithDB(batchProcessor)

	// Health check
	router.GET("/health", telemetryHandler.HealthCheck)

	// Telemetry endpoints
	router.POST("/telemetry", telemetryHandler.HandleTelemetry)
	router.POST("/telemetry/batch", telemetryHandler.HandleTelemetryBatch)

	return router
}
