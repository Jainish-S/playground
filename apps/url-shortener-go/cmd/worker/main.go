package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignore error if file doesn't exist - use system env vars)
	_ = godotenv.Load()

	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	database, err := db.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Connected to PostgreSQL with TimescaleDB")

	// Initialize Redis cache
	redisCache, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()
	log.Println("Connected to Redis")

	// Create analytics flusher
	flusher := worker.NewFlusher(redisCache, database, cfg)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the flusher worker
	log.Println("Starting Analytics Worker...")
	go flusher.Start(ctx)

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Analytics Worker...")
	cancel()

	// Give time for pending flushes to complete
	time.Sleep(2 * time.Second)
	log.Println("Analytics Worker stopped")
}

// ClickEvent represents a click event from Redis Stream
type ClickEvent struct {
	ShortCode string `json:"short_code"`
	IPHash    string `json:"ip_hash"`
	UserAgent string `json:"user_agent"`
	Referrer  string `json:"referrer"`
	Timestamp int64  `json:"timestamp"`
}

// ParseClickEvent parses a click event from JSON
func ParseClickEvent(data string) (*ClickEvent, error) {
	var event ClickEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}
	return &event, nil
}
