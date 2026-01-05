// Package config handles configuration for the guardrail server.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the guardrail server.
type Config struct {
	// Server configuration
	Host  string
	Port  int
	Debug bool

	// Model service URLs
	ModelPromptGuardURL  string
	ModelPIIDetectURL    string
	ModelHateDetectURL   string
	ModelContentClassURL string

	// Model call configuration
	ModelTimeout        time.Duration
	ModelConnectTimeout time.Duration

	// Circuit breaker configuration
	CBFailureThreshold int
	CBRecoveryTimeout  time.Duration
	CBSuccessThreshold int

	// Retry configuration
	RetryEnabled     bool
	RetryMaxAttempts int
	RetryWaitMs      int
}

// ModelURLs returns a map of model names to their URLs.
func (c *Config) ModelURLs() map[string]string {
	return map[string]string{
		"prompt-guard":  c.ModelPromptGuardURL,
		"pii-detect":    c.ModelPIIDetectURL,
		"hate-detect":   c.ModelHateDetectURL,
		"content-class": c.ModelContentClassURL,
	}
}

// Load loads configuration from environment variables.
func Load() *Config {
	return &Config{
		// Server configuration
		Host:  getEnv("HOST", "0.0.0.0"),
		Port:  getEnvInt("PORT", 8000),
		Debug: getEnvBool("DEBUG", false),

		// Model service URLs
		ModelPromptGuardURL:  getEnv("MODEL_PROMPT_GUARD_URL", "http://model-prompt-guard:8000"),
		ModelPIIDetectURL:    getEnv("MODEL_PII_DETECT_URL", "http://model-pii-detect:8000"),
		ModelHateDetectURL:   getEnv("MODEL_HATE_DETECT_URL", "http://model-hate-detect:8000"),
		ModelContentClassURL: getEnv("MODEL_CONTENT_CLASS_URL", "http://model-content-class:8000"),

		// Model call configuration
		ModelTimeout:        getEnvDuration("MODEL_TIMEOUT_SECONDS", 80*time.Millisecond),
		ModelConnectTimeout: getEnvDuration("MODEL_CONNECT_TIMEOUT", 20*time.Millisecond),

		// Circuit breaker configuration
		CBFailureThreshold: getEnvInt("CB_FAILURE_THRESHOLD", 5),
		CBRecoveryTimeout:  getEnvDuration("CB_RECOVERY_TIMEOUT", 30*time.Second),
		CBSuccessThreshold: getEnvInt("CB_SUCCESS_THRESHOLD", 3),

		// Retry configuration
		RetryEnabled:     getEnvBool("RETRY_ENABLED", true),
		RetryMaxAttempts: getEnvInt("RETRY_MAX_ATTEMPTS", 2),
		RetryWaitMs:      getEnvInt("RETRY_WAIT_MS", 5),
	}
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable with a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable with a default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable.
// For MODEL_TIMEOUT_SECONDS, RECOVERY_TIMEOUT etc., expects seconds as float.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return time.Duration(floatVal * float64(time.Second))
		}
	}
	return defaultValue
}
