// Package config handles configuration for the model service.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Host                  string
	Port                  int
	InferenceDelayEnabled bool
	InferenceDelayMinMs   int
	InferenceDelayMaxMs   int
}

func Load() *Config {
	return &Config{
		Host:                  getEnv("HOST", "0.0.0.0"),
		Port:                  getEnvInt("PORT", 8000),
		InferenceDelayEnabled: getEnvBool("INFERENCE_DELAY_ENABLED", true),
		InferenceDelayMinMs:   getEnvInt("INFERENCE_DELAY_MIN_MS", 10),
		InferenceDelayMaxMs:   getEnvInt("INFERENCE_DELAY_MAX_MS", 30),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}
