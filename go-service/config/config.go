package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                       string
	DBUrl                      string
	BatchSize                  int
	BatchTimeout               time.Duration
	MaxConnections             int
	AnomalyThresholdBattery    float64
	AnomalyThresholdStorage    float64
	AnomalyThresholdSignal     float64
}

func LoadConfig() Config {
	return Config{
		Port:                       getEnv("PORT", "8080"),
		DBUrl:                      getEnv("DATABASE_URL", "postgres://postgres:postgres@timescaledb:5432/orbitstream?sslmode=disable"),
		BatchSize:                  getEnvInt("BATCH_SIZE", 1000),
		BatchTimeout:               getEnvDuration("BATCH_TIMEOUT", 1*time.Second),
		MaxConnections:             getEnvInt("MAX_CONNECTIONS", 50),
		AnomalyThresholdBattery:    getEnvFloat("ANOMALY_THRESHOLD_BATTERY", 10.0),
		AnomalyThresholdStorage:    getEnvFloat("ANOMALY_THRESHOLD_STORAGE", 95000.0),
		AnomalyThresholdSignal:     getEnvFloat("ANOMALY_THRESHOLD_SIGNAL", -100.0),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
