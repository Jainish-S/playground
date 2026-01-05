// Model Prompt Guard - Main Application Entry Point
//
// This is the Go HTTP server that provides prompt injection detection.
// Uses keyword-based detection as a dummy implementation.
//
// IMPORTANT: This service processes ONE REQUEST AT A TIME per pod
// using a semaphore, simulating real ML which is CPU/GPU-bound.
//
// To run:
//
//	go run ./cmd/server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gocommon "github.com/playground/packages/go-common"

	"github.com/playground/apps/model-prompt-guard-go/internal/api"
	"github.com/playground/apps/model-prompt-guard-go/internal/config"
)

const modelName = "prompt-guard"

var shuttingDown = false

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("[%s] Starting with configuration:", modelName)
	log.Printf("  Host: %s", cfg.Host)
	log.Printf("  Port: %d", cfg.Port)
	log.Printf("  Inference Delay Enabled: %v", cfg.InferenceDelayEnabled)
	if cfg.InferenceDelayEnabled {
		log.Printf("  Inference Delay: %d-%dms", cfg.InferenceDelayMinMs, cfg.InferenceDelayMaxMs)
	}

	// Initialize metrics
	metrics := gocommon.NewModelMetrics(modelName)

	// Create HTTP handler
	handler := api.NewHandler(cfg, metrics, &shuttingDown)

	// Create ServeMux and register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Add metrics endpoint
	mux.Handle("GET /metrics", gocommon.MetricsHandler())

	// Wrap with metrics middleware
	wrappedMux := metrics.MetricsMiddleware(mux)

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      wrappedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[%s] Listening on %s", modelName, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("[%s] Shutdown initiated, draining requests...", modelName)
	shuttingDown = true

	// Brief drain period
	time.Sleep(500 * time.Millisecond)

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("[%s] Shutdown complete", modelName)
}
