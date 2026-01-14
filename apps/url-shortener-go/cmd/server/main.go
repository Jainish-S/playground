package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/api"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/gofiber/fiber/v2"
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

	// Initialize handlers and register routes
	handlers := api.NewHandlers(redisCache, database, cfg)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "URL Shortener API",
		ServerHeader: "url-shortener",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ReadBufferSize: 16384,
	})

	// Middleware - imported from fiber/v2/middleware/*
	// Note: recover and logger middlewares should be added for production
	// For now, using minimal middleware setup

	// Register health endpoints before other routes
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	app.Get("/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		checks := fiber.Map{}
		ready := true

		// Check database
		if err := database.HealthCheck(ctx); err != nil {
			checks["postgres"] = fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			ready = false
		} else {
			stats := database.Stats()
			checks["postgres"] = fiber.Map{
				"status":         "healthy",
				"total_conns":    stats.TotalConns(),
				"idle_conns":     stats.IdleConns(),
				"acquired_conns": stats.AcquiredConns(),
			}
		}

		// Check Redis
		if err := redisCache.HealthCheck(ctx); err != nil {
			checks["redis"] = fiber.Map{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			ready = false
		} else {
			stats, _ := redisCache.Stats(ctx)
			checks["redis"] = fiber.Map{
				"status":     "healthy",
				"hits":       stats.Hits,
				"misses":     stats.Misses,
				"idle_conns": stats.IdleConns,
			}
		}

		status := "ready"
		if !ready {
			status = "not_ready"
		}

		return c.JSON(fiber.Map{
			"status": status,
			"checks": checks,
		})
	})

	// Register all API routes (includes Auth0 middleware for /v1/* routes)
	api.RegisterRoutes(app, handlers, cfg)

	// Start server in goroutine
	go func() {
		addr := cfg.Host + ":" + cfg.Port
		log.Printf("Starting URL Shortener API on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exiting")
}
