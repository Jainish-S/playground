package api

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/playground/apps/model-hate-detect-go/internal/config"
	"github.com/playground/apps/model-hate-detect-go/internal/inference"
	gocommon "github.com/playground/packages/go-common"
)

const modelName = "hate-detect"

type Handler struct {
	cfg          *config.Config
	metrics      *gocommon.Metrics
	shuttingDown *bool
	semaphore    chan struct{}
}

func NewHandler(cfg *config.Config, metrics *gocommon.Metrics, shuttingDown *bool) *Handler {
	return &Handler{cfg: cfg, metrics: metrics, shuttingDown: shuttingDown, semaphore: make(chan struct{}, 1)}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /predict", h.handlePredict)
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /ready", h.handleReady)
}

func (h *Handler) handlePredict(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	var req gocommon.ModelPredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	h.semaphore <- struct{}{}
	defer func() { <-h.semaphore }()

	if h.cfg.InferenceDelayEnabled {
		delayMs := h.cfg.InferenceDelayMinMs + rand.Intn(h.cfg.InferenceDelayMaxMs-h.cfg.InferenceDelayMinMs+1)
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		log.Printf("[%s] Simulated delay: %dms for request %s", modelName, delayMs, req.RequestID)
	}

	flagged, score, details := inference.DetectHateSpeech(req.Text)
	latencyMs := int(time.Since(startTime).Milliseconds())

	h.metrics.InferenceLatency.WithLabelValues(modelName).Observe(float64(latencyMs) / 1000.0)
	h.metrics.InferenceTotal.WithLabelValues(modelName, "success").Inc()

	h.writeJSON(w, http.StatusOK, gocommon.ModelPredictResponse{Flagged: flagged, Score: score, Details: details, LatencyMs: latencyMs})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, gocommon.HealthResponse{Status: "healthy", Model: modelName})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	if h.shuttingDown != nil && *h.shuttingDown {
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "draining"})
		return
	}
	h.writeJSON(w, http.StatusOK, gocommon.ReadyResponse{Status: "ready", Model: modelName})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
