// Guardrail Server - Main Application Entry Point
//
// This is the Go HTTP server that orchestrates ML model calls
// for LLM guardrail validation.
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

	"github.com/playground/apps/guardrail-server-go/internal/api"
	"github.com/playground/apps/guardrail-server-go/internal/circuitbreaker"
	"github.com/playground/apps/guardrail-server-go/internal/client"
	"github.com/playground/apps/guardrail-server-go/internal/config"
	"github.com/playground/apps/guardrail-server-go/internal/orchestrator"
)

var shuttingDown = false

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("[guardrail-server-go] Starting with configuration:")
	log.Printf("  Host: %s", cfg.Host)
	log.Printf("  Port: %d", cfg.Port)
	log.Printf("  Model URLs: %v", cfg.ModelURLs())
	log.Printf("  Model Timeout: %v", cfg.ModelTimeout)
	log.Printf("  CB Failure Threshold: %d", cfg.CBFailureThreshold)
	log.Printf("  Retry Enabled: %v", cfg.RetryEnabled)
	log.Printf("  Retry Max Attempts: %d", cfg.RetryMaxAttempts)

	// Initialize metrics
	metrics := gocommon.NewGuardrailMetrics("guardrail-server")

	// Initialize client pool
	clients := client.NewPool(cfg)

	// Initialize circuit breaker registry
	breakers := circuitbreaker.NewRegistry(
		cfg.CBFailureThreshold,
		cfg.CBSuccessThreshold,
		cfg.CBRecoveryTimeout,
		metrics.CircuitBreakerState,
	)

	// Initialize orchestrator
	orch := orchestrator.New(cfg, clients, breakers, metrics)

	// Create HTTP handler
	handler := api.NewHandler(orch, breakers, &shuttingDown)

	// Create ServeMux and register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Add metrics endpoint
	mux.Handle("GET /metrics", gocommon.MetricsHandler())

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[guardrail-server-go] Listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[guardrail-server-go] Shutdown initiated")
	shuttingDown = true

	// Wait for in-flight requests to drain (max 5s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Clean up
	clients.CloseAll()

	log.Println("[guardrail-server-go] Shutdown complete")
}
