package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear all relevant environment variables
	unsetEnvVars()

	cfg := LoadConfig()

	// Verify default values
	if cfg.Port != "8080" {
		t.Errorf("expected Port to be '8080', got '%s'", cfg.Port)
	}
	if cfg.DBUrl != "postgres://postgres:postgres@timescaledb:5432/orbitstream?sslmode=disable" {
		t.Errorf("unexpected default DBUrl: %s", cfg.DBUrl)
	}
	if cfg.BatchSize != 1000 {
		t.Errorf("expected BatchSize to be 1000, got %d", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 1*time.Second {
		t.Errorf("expected BatchTimeout to be 1s, got %v", cfg.BatchTimeout)
	}
	if cfg.MaxConnections != 50 {
		t.Errorf("expected MaxConnections to be 50, got %d", cfg.MaxConnections)
	}
	if cfg.AnomalyThresholdBattery != 10.0 {
		t.Errorf("expected AnomalyThresholdBattery to be 10.0, got %f", cfg.AnomalyThresholdBattery)
	}
	if cfg.AnomalyThresholdStorage != 95000.0 {
		t.Errorf("expected AnomalyThresholdStorage to be 95000.0, got %f", cfg.AnomalyThresholdStorage)
	}
	if cfg.AnomalyThresholdSignal != -100.0 {
		t.Errorf("expected AnomalyThresholdSignal to be -100.0, got %f", cfg.AnomalyThresholdSignal)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	unsetEnvVars()

	// Set environment variables
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db?sslmode=require")
	os.Setenv("BATCH_SIZE", "500")
	os.Setenv("BATCH_TIMEOUT", "2s")
	os.Setenv("MAX_CONNECTIONS", "100")
	os.Setenv("ANOMALY_THRESHOLD_BATTERY", "5.0")
	os.Setenv("ANOMALY_THRESHOLD_STORAGE", "90000.0")
	os.Setenv("ANOMALY_THRESHOLD_SIGNAL", "-90.0")

	cfg := LoadConfig()

	if cfg.Port != "9090" {
		t.Errorf("expected Port to be '9090', got '%s'", cfg.Port)
	}
	if cfg.DBUrl != "postgres://user:pass@host:5432/db?sslmode=require" {
		t.Errorf("unexpected DBUrl: %s", cfg.DBUrl)
	}
	if cfg.BatchSize != 500 {
		t.Errorf("expected BatchSize to be 500, got %d", cfg.BatchSize)
	}
	if cfg.BatchTimeout != 2*time.Second {
		t.Errorf("expected BatchTimeout to be 2s, got %v", cfg.BatchTimeout)
	}
	if cfg.MaxConnections != 100 {
		t.Errorf("expected MaxConnections to be 100, got %d", cfg.MaxConnections)
	}
	if cfg.AnomalyThresholdBattery != 5.0 {
		t.Errorf("expected AnomalyThresholdBattery to be 5.0, got %f", cfg.AnomalyThresholdBattery)
	}
	if cfg.AnomalyThresholdStorage != 90000.0 {
		t.Errorf("expected AnomalyThresholdStorage to be 90000.0, got %f", cfg.AnomalyThresholdStorage)
	}
	if cfg.AnomalyThresholdSignal != -90.0 {
		t.Errorf("expected AnomalyThresholdSignal to be -90.0, got %f", cfg.AnomalyThresholdSignal)
	}

	unsetEnvVars()
}

func TestGetEnv(t *testing.T) {
	// Test when env var is not set
	result := getEnv("NONEXISTENT_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("expected 'default_value', got '%s'", result)
	}

	// Test when env var is set
	os.Setenv("TEST_GET_ENV", "custom_value")
	result = getEnv("TEST_GET_ENV", "default_value")
	if result != "custom_value" {
		t.Errorf("expected 'custom_value', got '%s'", result)
	}
	os.Unsetenv("TEST_GET_ENV")

	// Test empty string env var - returns empty string (code behavior)
	os.Setenv("TEST_GET_ENV", "")
	result = getEnv("TEST_GET_ENV", "default_value")
	// Note: The actual behavior of getEnv is to return the value even if empty
	// because os.Getenv returns "" for both unset and empty string values
	if result != "default_value" {
		t.Errorf("getEnv returns default for empty string (unset and empty both return ''), got '%s'", result)
	}
	os.Unsetenv("TEST_GET_ENV")
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		defaultValue  int
		expectedValue int
	}{
		{
			name:          "env var not set",
			envValue:      "",
			defaultValue:  42,
			expectedValue: 42,
		},
		{
			name:          "valid integer",
			envValue:      "100",
			defaultValue:  42,
			expectedValue: 100,
		},
		{
			name:          "negative integer",
			envValue:      "-50",
			defaultValue:  42,
			expectedValue: -50,
		},
		{
			name:          "zero",
			envValue:      "0",
			defaultValue:  42,
			expectedValue: 0,
		},
		{
			name:          "invalid integer - returns default",
			envValue:      "not_a_number",
			defaultValue:  42,
			expectedValue: 42,
		},
		{
			name:          "float string - returns default",
			envValue:      "12.5",
			defaultValue:  42,
			expectedValue: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_INT_VAR", tt.envValue)
			} else {
				os.Unsetenv("TEST_INT_VAR")
			}
			result := getEnvInt("TEST_INT_VAR", tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("expected %d, got %d", tt.expectedValue, result)
			}
			os.Unsetenv("TEST_INT_VAR")
		})
	}
}

func TestGetEnvFloat(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		defaultValue  float64
		expectedValue float64
	}{
		{
			name:          "env var not set",
			envValue:      "",
			defaultValue:  10.5,
			expectedValue: 10.5,
		},
		{
			name:          "valid float",
			envValue:      "99.9",
			defaultValue:  10.5,
			expectedValue: 99.9,
		},
		{
			name:          "negative float",
			envValue:      "-100.0",
			defaultValue:  10.5,
			expectedValue: -100.0,
		},
		{
			name:          "integer string",
			envValue:      "42",
			defaultValue:  10.5,
			expectedValue: 42.0,
		},
		{
			name:          "scientific notation",
			envValue:      "1.5e2",
			defaultValue:  10.5,
			expectedValue: 150.0,
		},
		{
			name:          "invalid float - returns default",
			envValue:      "not_a_number",
			defaultValue:  10.5,
			expectedValue: 10.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_FLOAT_VAR", tt.envValue)
			} else {
				os.Unsetenv("TEST_FLOAT_VAR")
			}
			result := getEnvFloat("TEST_FLOAT_VAR", tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("expected %f, got %f", tt.expectedValue, result)
			}
			os.Unsetenv("TEST_FLOAT_VAR")
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		defaultValue  time.Duration
		expectedValue time.Duration
	}{
		{
			name:          "env var not set",
			envValue:      "",
			defaultValue:  5 * time.Second,
			expectedValue: 5 * time.Second,
		},
		{
			name:          "seconds",
			envValue:      "10s",
			defaultValue:  5 * time.Second,
			expectedValue: 10 * time.Second,
		},
		{
			name:          "milliseconds",
			envValue:      "500ms",
			defaultValue:  5 * time.Second,
			expectedValue: 500 * time.Millisecond,
		},
		{
			name:          "minutes",
			envValue:      "2m",
			defaultValue:  5 * time.Second,
			expectedValue: 2 * time.Minute,
		},
		{
			name:          "hours",
			envValue:      "1h",
			defaultValue:  5 * time.Second,
			expectedValue: 1 * time.Hour,
		},
		{
			name:          "combined units",
			envValue:      "1m30s",
			defaultValue:  5 * time.Second,
			expectedValue: 90 * time.Second,
		},
		{
			name:          "invalid duration - returns default",
			envValue:      "not_a_duration",
			defaultValue:  5 * time.Second,
			expectedValue: 5 * time.Second,
		},
		{
			name:          "negative duration - Go parses negative durations",
			envValue:      "-5s",
			defaultValue:  5 * time.Second,
			expectedValue: -5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("TEST_DURATION_VAR", tt.envValue)
			} else {
				os.Unsetenv("TEST_DURATION_VAR")
			}
			result := getEnvDuration("TEST_DURATION_VAR", tt.defaultValue)
			if result != tt.expectedValue {
				t.Errorf("expected %v, got %v", tt.expectedValue, result)
			}
			os.Unsetenv("TEST_DURATION_VAR")
		})
	}
}

// Helper function to clear all environment variables used by config
func unsetEnvVars() {
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("BATCH_TIMEOUT")
	os.Unsetenv("MAX_CONNECTIONS")
	os.Unsetenv("ANOMALY_THRESHOLD_BATTERY")
	os.Unsetenv("ANOMALY_THRESHOLD_STORAGE")
	os.Unsetenv("ANOMALY_THRESHOLD_SIGNAL")
}
