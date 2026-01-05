// Package orchestrator handles parallel model calls with fault tolerance.
//
// The orchestrator is the core of the guardrail server. It:
//  1. Fans out requests to all enabled models in parallel
//  2. Uses circuit breakers for fault tolerance
//  3. Aggregates results using configurable strategy
//  4. Handles partial failures gracefully
package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	gocommon "github.com/playground/packages/go-common"

	"github.com/playground/apps/guardrail-server-go/internal/circuitbreaker"
	"github.com/playground/apps/guardrail-server-go/internal/client"
	"github.com/playground/apps/guardrail-server-go/internal/config"
)

// AggregationStrategy defines how to aggregate model results.
type AggregationStrategy int

const (
	// StrategyAnyFlag flags if ANY model flags
	StrategyAnyFlag AggregationStrategy = iota
	// StrategyAllFlag flags only if ALL models flag
	StrategyAllFlag
	// StrategyMajority flags if majority (>50%) flag
	StrategyMajority
	// StrategyThreshold flags if weighted score exceeds threshold
	StrategyThreshold
)

// ModelCallResult holds the result from a single model call.
type ModelCallResult struct {
	ModelName string
	Success   bool
	Response  *gocommon.ModelPredictResponse
	Error     string
}

// Orchestrator coordinates model calls and aggregates results.
type Orchestrator struct {
	cfg        *config.Config
	clients    *client.Pool
	breakers   *circuitbreaker.Registry
	metrics    *gocommon.Metrics
	inFlight   int64
	inFlightMu sync.Mutex
}

// New creates a new orchestrator.
func New(cfg *config.Config, clients *client.Pool, breakers *circuitbreaker.Registry, metrics *gocommon.Metrics) *Orchestrator {
	return &Orchestrator{
		cfg:      cfg,
		clients:  clients,
		breakers: breakers,
		metrics:  metrics,
	}
}

// ValidateText validates text against all enabled models.
func (o *Orchestrator) ValidateText(ctx context.Context, text string, enabledModels []string, strategy AggregationStrategy, requestID string) (*gocommon.ValidateResponse, error) {
	startTime := time.Now()

	// Generate request ID if not provided
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Default to all models
	if len(enabledModels) == 0 {
		enabledModels = []string{"prompt-guard", "pii-detect", "hate-detect", "content-class"}
	}

	// Track in-flight requests
	o.inFlightMu.Lock()
	o.inFlight++
	o.inFlightMu.Unlock()
	hostname, _ := os.Hostname()
	o.metrics.InFlightRequests.WithLabelValues(hostname).Inc()

	defer func() {
		o.inFlightMu.Lock()
		o.inFlight--
		o.inFlightMu.Unlock()
		o.metrics.InFlightRequests.WithLabelValues(hostname).Dec()
	}()

	// Call all models in parallel
	results := o.callModelsParallel(ctx, text, requestID, enabledModels)

	// Process results
	modelResults := make(map[string]gocommon.ModelResultResponse)
	var failedModels []string
	var flagReasons []string

	for _, result := range results {
		if result.Success && result.Response != nil {
			modelResults[result.ModelName] = gocommon.ModelResultResponse{
				Flagged:   result.Response.Flagged,
				Score:     result.Response.Score,
				Details:   result.Response.Details,
				LatencyMs: result.Response.LatencyMs,
			}

			if result.Response.Flagged {
				flagReasons = append(flagReasons, result.ModelName+"_flagged")
			}
		} else {
			failedModels = append(failedModels, result.ModelName)
		}
	}

	// Aggregate results
	flagged := o.aggregateResults(modelResults, strategy)

	// Calculate latency
	latencyMs := int(time.Since(startTime).Milliseconds())

	// Record metrics
	o.metrics.RequestLatency.Observe(float64(latencyMs) / 1000.0)
	status := "success"
	if len(failedModels) > 0 {
		status = "partial"
	}
	o.metrics.RequestTotal.WithLabelValues(status, fmt.Sprintf("%t", flagged)).Inc()

	return &gocommon.ValidateResponse{
		RequestID:      requestID,
		Flagged:        flagged,
		FlagReasons:    flagReasons,
		ModelResults:   modelResults,
		PartialFailure: len(failedModels) > 0,
		FailedModels:   failedModels,
		LatencyMs:      latencyMs,
	}, nil
}

// callModelsParallel calls all models in parallel using goroutines.
func (o *Orchestrator) callModelsParallel(ctx context.Context, text, requestID string, models []string) []ModelCallResult {
	results := make([]ModelCallResult, len(models))
	var wg sync.WaitGroup

	for i, model := range models {
		wg.Add(1)
		go func(idx int, modelName string) {
			defer wg.Done()
			results[idx] = o.callModel(ctx, modelName, text, requestID)
		}(i, model)
	}

	wg.Wait()
	return results
}

// callModel calls a single model with circuit breaker and retry protection.
func (o *Orchestrator) callModel(ctx context.Context, modelName, text, requestID string) ModelCallResult {
	cb := o.breakers.Get(modelName)

	// Check circuit breaker
	if !cb.AllowRequest() {
		return ModelCallResult{
			ModelName: modelName,
			Success:   false,
			Error:     fmt.Sprintf("Circuit breaker open for %s", modelName),
		}
	}

	// Retry loop
	var lastErr error
	for attempt := 1; attempt <= o.cfg.RetryMaxAttempts; attempt++ {
		if attempt > 1 {
			// Record retry metric
			o.metrics.ModelCallRetries.WithLabelValues(modelName, fmt.Sprintf("%d", attempt)).Inc()
			log.Printf("[%s] Retry attempt %d", modelName, attempt)

			// Wait before retry
			time.Sleep(time.Duration(o.cfg.RetryWaitMs) * time.Millisecond)
		}

		result, err := o.doModelCall(ctx, modelName, text, requestID)
		if err == nil {
			cb.RecordSuccess()
			return result
		}

		lastErr = err

		// Don't retry for certain errors
		if !o.cfg.RetryEnabled {
			break
		}
	}

	// All retries failed
	cb.RecordFailure()
	return ModelCallResult{
		ModelName: modelName,
		Success:   false,
		Error:     fmt.Sprintf("Error calling %s: %v", modelName, lastErr),
	}
}

// doModelCall performs the actual HTTP call to a model.
func (o *Orchestrator) doModelCall(ctx context.Context, modelName, text, requestID string) (ModelCallResult, error) {
	startTime := time.Now()

	baseURL := o.clients.GetBaseURL(modelName)
	if baseURL == "" {
		return ModelCallResult{}, fmt.Errorf("unknown model: %s", modelName)
	}

	// Prepare request
	reqBody := gocommon.ModelPredictRequest{
		Text:      text,
		RequestID: requestID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return ModelCallResult{}, fmt.Errorf("marshal error: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/predict", bytes.NewReader(body))
	if err != nil {
		return ModelCallResult{}, fmt.Errorf("request creation error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make request
	client := o.clients.Get(modelName)
	resp, err := client.Do(req)
	if err != nil {
		return ModelCallResult{}, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	// Record latency
	duration := time.Since(startTime).Seconds()
	o.metrics.ModelCallLatency.WithLabelValues(modelName).Observe(duration)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return ModelCallResult{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelCallResult{}, fmt.Errorf("read error: %w", err)
	}

	var modelResp gocommon.ModelPredictResponse
	if err := json.Unmarshal(respBody, &modelResp); err != nil {
		return ModelCallResult{}, fmt.Errorf("unmarshal error: %w", err)
	}

	return ModelCallResult{
		ModelName: modelName,
		Success:   true,
		Response:  &modelResp,
	}, nil
}

// aggregateResults aggregates model results based on strategy.
func (o *Orchestrator) aggregateResults(results map[string]gocommon.ModelResultResponse, strategy AggregationStrategy) bool {
	if len(results) == 0 {
		return false
	}

	flags := make([]bool, 0, len(results))
	var totalScore float64

	for _, r := range results {
		flags = append(flags, r.Flagged)
		totalScore += r.Score
	}

	switch strategy {
	case StrategyAnyFlag:
		for _, f := range flags {
			if f {
				return true
			}
		}
		return false

	case StrategyAllFlag:
		for _, f := range flags {
			if !f {
				return false
			}
		}
		return true

	case StrategyMajority:
		count := 0
		for _, f := range flags {
			if f {
				count++
			}
		}
		return float64(count) > float64(len(flags))/2

	case StrategyThreshold:
		avgScore := totalScore / float64(len(results))
		return avgScore > 0.5

	default:
		return false
	}
}

// GetInFlight returns the current number of in-flight requests.
func (o *Orchestrator) GetInFlight() int64 {
	o.inFlightMu.Lock()
	defer o.inFlightMu.Unlock()
	return o.inFlight
}
