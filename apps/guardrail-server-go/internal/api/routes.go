// Package api provides HTTP routes for the guardrail server.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	gocommon "github.com/playground/packages/go-common"

	"github.com/playground/apps/guardrail-server-go/internal/circuitbreaker"
	"github.com/playground/apps/guardrail-server-go/internal/orchestrator"
)

// Handler holds the API dependencies.
type Handler struct {
	orchestrator *orchestrator.Orchestrator
	breakers     *circuitbreaker.Registry
	shuttingDown *bool
}

// NewHandler creates a new API handler.
func NewHandler(orch *orchestrator.Orchestrator, breakers *circuitbreaker.Registry, shuttingDown *bool) *Handler {
	return &Handler{
		orchestrator: orch,
		breakers:     breakers,
		shuttingDown: shuttingDown,
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Main API routes (Go 1.22+ pattern matching)
	mux.HandleFunc("POST /v1/validate", h.handleValidate)
	mux.HandleFunc("GET /v1/health", h.handleHealth)
	mux.HandleFunc("GET /v1/ready", h.handleReady)

	// Debug routes
	mux.HandleFunc("GET /debug/circuit-breakers", h.handleGetCircuitBreakers)
	mux.HandleFunc("POST /debug/circuit-breakers/{model}/close", h.handleForceCloseCircuitBreaker)
	mux.HandleFunc("POST /debug/circuit-breakers/{model}/open", h.handleForceOpenCircuitBreaker)

	// Root
	mux.HandleFunc("GET /", h.handleRoot)
}

// handleValidate handles POST /v1/validate
func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	// Check API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		h.writeError(w, http.StatusUnauthorized, "invalid_api_key", "API key required")
		return
	}

	// Parse request body
	var req gocommon.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body")
		return
	}

	// Validate required fields
	if req.ProjectID == "" {
		h.writeError(w, http.StatusBadRequest, "validation_error", "project_id is required")
		return
	}
	if req.Text == "" {
		h.writeError(w, http.StatusBadRequest, "validation_error", "text is required")
		return
	}
	if len(req.Text) > 50000 {
		h.writeError(w, http.StatusBadRequest, "validation_error", "text exceeds 50000 characters")
		return
	}

	// Default type
	if req.Type == "" {
		req.Type = "input"
	}

	// Validate text
	result, err := h.orchestrator.ValidateText(
		r.Context(),
		req.Text,
		nil, // All models
		orchestrator.StrategyAnyFlag,
		req.RequestID,
	)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}

	// Check if all models failed
	if result.PartialFailure && len(result.FailedModels) == 4 {
		h.writeError(w, http.StatusServiceUnavailable, "service_unavailable", "All model services unavailable")
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleHealth handles GET /v1/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, gocommon.HealthResponse{
		Status: "healthy",
	})
}

// handleReady handles GET /v1/ready
func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	// Check shutdown state
	if h.shuttingDown != nil && *h.shuttingDown {
		h.writeError(w, http.StatusServiceUnavailable, "draining", "Server shutting down, not accepting new requests")
		return
	}

	// Check circuit breakers
	breakers := h.breakers.GetAll()
	var availableModels []string

	for name, cb := range breakers {
		if cb.AllowRequest() {
			availableModels = append(availableModels, name)
		}
	}

	// If no breakers exist yet, we're ready
	if len(breakers) == 0 {
		h.writeJSON(w, http.StatusOK, gocommon.ReadyResponse{
			Status:          "ready",
			AvailableModels: []string{"all (not initialized)"},
		})
		return
	}

	if len(availableModels) > 0 {
		h.writeJSON(w, http.StatusOK, gocommon.ReadyResponse{
			Status:          "ready",
			AvailableModels: availableModels,
		})
	} else {
		h.writeError(w, http.StatusServiceUnavailable, "no_models_available", "No models available (all circuit breakers open)")
	}
}

// handleGetCircuitBreakers handles GET /debug/circuit-breakers
func (h *Handler) handleGetCircuitBreakers(w http.ResponseWriter, r *http.Request) {
	breakers := h.breakers.GetAll()
	result := make(map[string]gocommon.CircuitBreakerStatus)

	for name, cb := range breakers {
		result[name] = cb.GetStatus()
	}

	h.writeJSON(w, http.StatusOK, result)
}

// handleForceCloseCircuitBreaker handles POST /debug/circuit-breakers/{model}/close
func (h *Handler) handleForceCloseCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	modelName := r.PathValue("model")
	if modelName == "" {
		modelName = extractModelFromPath(r.URL.Path, "/debug/circuit-breakers/", "/close")
	}

	cb := h.breakers.Get(modelName)
	cb.ForceClose()

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Circuit breaker for " + modelName + " forced closed",
	})
}

// handleForceOpenCircuitBreaker handles POST /debug/circuit-breakers/{model}/open
func (h *Handler) handleForceOpenCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	modelName := r.PathValue("model")
	if modelName == "" {
		modelName = extractModelFromPath(r.URL.Path, "/debug/circuit-breakers/", "/open")
	}

	cb := h.breakers.Get(modelName)
	cb.ForceOpen()

	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Circuit breaker for " + modelName + " forced open",
	})
}

// handleRoot handles GET /
func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{
		"service": "guardrail-server-go",
		"version": "0.1.0",
		"docs":    "/docs",
		"health":  "/v1/health",
	})
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, gocommon.ErrorResponse{
		Error: gocommon.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// extractModelFromPath extracts model name from URL path
func extractModelFromPath(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return path
}
