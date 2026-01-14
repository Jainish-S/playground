package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Host string
	Port string

	// Base URL for short links
	BaseURL string

	// Database configuration
	PostgresHost     string
	PostgresPort     string
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresMaxConns int

	// Redis configuration
	RedisHost string
	RedisPort string
	RedisDB   int

	// Cache TTLs
	URLCacheTTL time.Duration
	QRCacheTTL  time.Duration

	// Rate limiting
	RateLimitCreatePerMinute   int
	RateLimitRedirectPerSecond int

	// Auth0 configuration
	Auth0Domain   string
	Auth0Audience string

	// GeoIP configuration
	GeoIPDBPath string

	// Short code configuration
	ShortCodeMinLength int
	DefaultTTLDays     int

	// Security
	IPHashSalt string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		// Server
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnv("PORT", "8000"),

		// Base URL
		BaseURL: getEnv("BASE_URL", "http://localhost:8000"),

		// Database
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:       getEnv("POSTGRES_DB", "urlshortener"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresMaxConns: getEnvInt("POSTGRES_MAX_CONNECTIONS", 20),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		// Cache TTLs
		URLCacheTTL: time.Duration(getEnvInt("URL_CACHE_TTL", 3600)) * time.Second,
		QRCacheTTL:  time.Duration(getEnvInt("QR_CACHE_TTL", 86400)) * time.Second,

		// Rate limiting
		RateLimitCreatePerMinute:   getEnvInt("RATE_LIMIT_CREATE_PER_MINUTE", 10),
		RateLimitRedirectPerSecond: getEnvInt("RATE_LIMIT_REDIRECT_PER_SECOND", 100),

		// Auth0
		Auth0Domain:   getEnv("AUTH0_DOMAIN", ""),
		Auth0Audience: getEnv("AUTH0_AUDIENCE", ""),

		// GeoIP
		GeoIPDBPath: getEnv("GEOIP_DB_PATH", "/data/GeoLite2-City.mmdb"),

		// Short codes
		ShortCodeMinLength: getEnvInt("SHORT_CODE_MIN_LENGTH", 6),
		DefaultTTLDays:     getEnvInt("DEFAULT_TTL_DAYS", 365),

		// Security
		IPHashSalt: getEnv("IP_HASH_SALT", "change-this-in-production"),
	}
}

// DatabaseURL returns the PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	return "postgres://" + c.PostgresUser + ":" + c.PostgresPassword +
		"@" + c.PostgresHost + ":" + c.PostgresPort +
		"/" + c.PostgresDB + "?sslmode=disable"
}

// RedisAddr returns the Redis address
func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
