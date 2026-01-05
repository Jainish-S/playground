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

	"github.com/playground/apps/model-hate-detect-go/internal/api"
	"github.com/playground/apps/model-hate-detect-go/internal/config"
	gocommon "github.com/playground/packages/go-common"
)

const modelName = "hate-detect"

var shuttingDown = false

func main() {
	cfg := config.Load()
	log.Printf("[%s] Starting on %s:%d", modelName, cfg.Host, cfg.Port)

	metrics := gocommon.NewModelMetrics(modelName)
	handler := api.NewHandler(cfg, metrics, &shuttingDown)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	mux.Handle("GET /metrics", gocommon.MetricsHandler())

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{Addr: addr, Handler: metrics.MetricsMiddleware(mux), ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}

	go func() {
		log.Printf("[%s] Listening on %s", modelName, addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shuttingDown = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Printf("[%s] Shutdown complete", modelName)
}
