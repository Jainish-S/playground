// Package config handles configuration for the model service.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the model service.
type Config struct {
	Host string
	Port int

	InferenceDelayEnabled bool
	InferenceDelayMinMs   int
	InferenceDelayMaxMs   int
}

func Load() *Config {
	return &Config{
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnvInt("PORT", 8000),

		InferenceDelayEnabled: getEnvBool("INFERENCE_DELAY_ENABLED", true),
		InferenceDelayMinMs:   getEnvInt("INFERENCE_DELAY_MIN_MS", 10),
		InferenceDelayMaxMs:   getEnvInt("INFERENCE_DELAY_MAX_MS", 30),
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
