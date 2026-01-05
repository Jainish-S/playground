// Package api provides HTTP routes for the model service.
package api

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	gocommon "github.com/playground/packages/go-common"

	"github.com/playground/apps/model-prompt-guard-go/internal/config"
	"github.com/playground/apps/model-prompt-guard-go/internal/inference"
)

const modelName = "prompt-guard"

// Handler holds the API dependencies.
type Handler struct {
	cfg          *config.Config
	metrics      *gocommon.Metrics
	shuttingDown *bool

	// Semaphore for single-request-at-a-time processing
	// This simulates real ML which can only process one request at a time per pod
	semaphore chan struct{}
}

// NewHandler creates a new API handler.
func NewHandler(cfg *config.Config, metrics *gocommon.Metrics, shuttingDown *bool) *Handler {
	return &Handler{
		cfg:          cfg,
		metrics:      metrics,
		shuttingDown: shuttingDown,
		semaphore:    make(chan struct{}, 1), // Capacity 1 = single request at a time
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /predict", h.handlePredict)
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /ready", h.handleReady)
}

// handlePredict handles POST /predict
func (h *Handler) handlePredict(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Parse request
	var req gocommon.ModelPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Acquire semaphore (blocks if another request is processing)
	// This ensures only one inference runs at a time per pod
	h.semaphore <- struct{}{}
	defer func() { <-h.semaphore }()

	// Simulate ML inference delay (blocking)
	if h.cfg.InferenceDelayEnabled {
		delayMs := h.cfg.InferenceDelayMinMs
		if h.cfg.InferenceDelayMaxMs > h.cfg.InferenceDelayMinMs {
			delayMs += rand.Intn(h.cfg.InferenceDelayMaxMs - h.cfg.InferenceDelayMinMs)
		}
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		log.Printf("[%s] Simulated delay: %dms for request %s", modelName, delayMs, req.RequestID)
	}

	// Run inference
	flagged, score, details := inference.DetectPromptInjection(req.Text)

	latencyMs := int(time.Since(startTime).Milliseconds())

	// Record metrics
	h.metrics.InferenceLatency.WithLabelValues(modelName).Observe(float64(latencyMs) / 1000.0)
	h.metrics.InferenceTotal.WithLabelValues(modelName, "success").Inc()

	// Send response
	h.writeJSON(w, http.StatusOK, gocommon.ModelPredictResponse{
		Flagged:   flagged,
		Score:     score,
		Details:   details,
		LatencyMs: latencyMs,
	})
}

// handleHealth handles GET /health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, gocommon.HealthResponse{
		Status: "healthy",
		Model:  modelName,
	})
}

// handleReady handles GET /ready
func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	if h.shuttingDown != nil && *h.shuttingDown {
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "draining",
			"model":  modelName,
		})
		return
	}

	h.writeJSON(w, http.StatusOK, gocommon.ReadyResponse{
		Status: "ready",
		Model:  modelName,
	})
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
