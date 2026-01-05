// Package gocommon provides shared schemas for the guardrails platform.
package gocommon

// ModelPredictRequest is the request schema for model prediction endpoints.
// Matches the Python ModelPredictRequest in py-common.
type ModelPredictRequest struct {
	// Text to analyze (max 50000 characters)
	Text string `json:"text"`
	// Request ID for tracing
	RequestID string `json:"request_id"`
}

// ModelPredictResponse is the response schema from model prediction endpoints.
// Matches the Python ModelPredictResponse in py-common.
type ModelPredictResponse struct {
	// Whether the text was flagged
	Flagged bool `json:"flagged"`
	// Confidence score (0.0 to 1.0)
	Score float64 `json:"score"`
	// Explanation details
	Details []string `json:"details"`
	// Inference latency in milliseconds
	LatencyMs int `json:"latency_ms"`
}

// ValidateRequest is the request schema for the main validation endpoint.
// Matches the Python ValidateRequest in py-common.
type ValidateRequest struct {
	// Optional client-provided request ID
	RequestID string `json:"request_id,omitempty"`
	// Project ID for config lookup
	ProjectID string `json:"project_id"`
	// Text to validate (max 50000 characters)
	Text string `json:"text"`
	// Input or output type (default: "input")
	Type string `json:"type"`
	// Optional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ModelResultResponse is the result from a single model.
// Matches the Python ModelResultResponse in py-common.
type ModelResultResponse struct {
	Flagged   bool     `json:"flagged"`
	Score     float64  `json:"score"`
	Details   []string `json:"details"`
	LatencyMs int      `json:"latency_ms"`
}

// ValidateResponse is the response schema for the main validation endpoint.
// Matches the Python ValidateResponse in py-common.
type ValidateResponse struct {
	// Request ID
	RequestID string `json:"request_id"`
	// Overall flag status
	Flagged bool `json:"flagged"`
	// Reasons for flagging
	FlagReasons []string `json:"flag_reasons"`
	// Per-model results
	ModelResults map[string]ModelResultResponse `json:"model_results"`
	// Some models failed
	PartialFailure bool `json:"partial_failure"`
	// Models that failed
	FailedModels []string `json:"failed_models"`
	// Total request latency in milliseconds
	LatencyMs int `json:"latency_ms"`
}

// HealthResponse is the response for health check endpoints.
type HealthResponse struct {
	Status string `json:"status"`
	Model  string `json:"model,omitempty"`
}

// ReadyResponse is the response for readiness check endpoints.
type ReadyResponse struct {
	Status          string   `json:"status"`
	AvailableModels []string `json:"available_models,omitempty"`
	Model           string   `json:"model,omitempty"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// CircuitBreakerStatus represents the status of a circuit breaker.
type CircuitBreakerStatus struct {
	Name            string  `json:"name"`
	State           string  `json:"state"`
	FailureCount    int     `json:"failure_count"`
	SuccessCount    int     `json:"success_count"`
	LastFailureTime float64 `json:"last_failure_time"`
}
