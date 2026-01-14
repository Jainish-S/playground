package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new database connection pool
func New(cfg *config.Config) (*DB, error) {
	// Parse connection string
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.PostgresMaxConns)
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// HealthCheck performs a database health check
func (db *DB) HealthCheck(ctx context.Context) error {
	// Ping database
	if err := db.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check if TimescaleDB is installed
	var version string
	err := db.Pool.QueryRow(ctx, `
		SELECT extversion
		FROM pg_extension
		WHERE extname = 'timescaledb'
	`).Scan(&version)
	if err != nil {
		return fmt.Errorf("timescaledb extension not found: %w", err)
	}

	return nil
}

// Stats returns database pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
