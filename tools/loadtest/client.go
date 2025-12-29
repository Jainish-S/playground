package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

// ValidateRequest mirrors the guardrail API request schema
type ValidateRequest struct {
	RequestID string            `json:"request_id"`
	ProjectID string            `json:"project_id"`
	Text      string            `json:"text"`
	Type      string            `json:"type"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ValidateResponse mirrors the guardrail API response schema
type ValidateResponse struct {
	RequestID      string                 `json:"request_id"`
	Flagged        bool                   `json:"flagged"`
	FlagReasons    []string               `json:"flag_reasons"`
	ModelResults   map[string]ModelResult `json:"model_results"`
	PartialFailure bool                   `json:"partial_failure"`
	FailedModels   []string               `json:"failed_models"`
	LatencyMs      int                    `json:"latency_ms"`
}

// ModelResult represents a single model's result
type ModelResult struct {
	Flagged   bool     `json:"flagged"`
	Score     float64  `json:"score"`
	Details   []string `json:"details"`
	LatencyMs int      `json:"latency_ms"`
}

// Tenant represents a simulated tenant
type Tenant struct {
	ID        string
	APIKey    string
	ProjectID string
}

// Client handles HTTP requests to the guardrail API
type Client struct {
	httpClient *http.Client
	baseURL    string
	tenants    []Tenant
	tenantIdx  atomic.Uint64
	reqCounter atomic.Uint64
}

// NewClient creates a new guardrail API client
func NewClient(baseURL string, numTenants int) *Client {
	// Create HTTP client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	client := &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
		},
		baseURL: baseURL,
		tenants: make([]Tenant, numTenants),
	}

	// Generate simulated tenants
	for i := 0; i < numTenants; i++ {
		client.tenants[i] = Tenant{
			ID:        fmt.Sprintf("tenant-%d", i+1),
			APIKey:    fmt.Sprintf("api-key-%d-%s", i+1, randomString(8)),
			ProjectID: fmt.Sprintf("project-%d", i+1),
		}
	}

	return client
}

// RequestResult holds the result of a single request
type RequestResult struct {
	TenantID  string
	Latency   time.Duration
	Success   bool
	Timeout   bool
	Error     error
	Flagged   bool
	StatusCode int
}

// SendRequest sends a validation request to the guardrail API
func (c *Client) SendRequest(ctx context.Context) RequestResult {
	// Round-robin tenant selection
	idx := c.tenantIdx.Add(1) - 1
	tenant := c.tenants[idx%uint64(len(c.tenants))]

	// Generate request ID
	reqID := fmt.Sprintf("load-%d-%d", time.Now().UnixNano(), c.reqCounter.Add(1))

	// Create request body
	reqBody := ValidateRequest{
		RequestID: reqID,
		ProjectID: tenant.ProjectID,
		Text:      generateRandomText(),
		Type:      "input",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return RequestResult{
			TenantID: tenant.ID,
			Success:  false,
			Error:    err,
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/validate", bytes.NewReader(body))
	if err != nil {
		return RequestResult{
			TenantID: tenant.ID,
			Success:  false,
			Error:    err,
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", tenant.APIKey)

	// Send request and measure latency
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	latency := time.Since(start)

	result := RequestResult{
		TenantID: tenant.ID,
		Latency:  latency,
	}

	if err != nil {
		// Check if it's a timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.Timeout = true
		}
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return result
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true

		// Parse response to get flagged status
		var validateResp ValidateResponse
		if err := json.Unmarshal(respBody, &validateResp); err == nil {
			result.Flagged = validateResp.Flagged
		}
	} else if resp.StatusCode == 429 {
		result.Error = fmt.Errorf("rate limited")
	} else if resp.StatusCode >= 500 {
		result.Error = fmt.Errorf("server error: %d", resp.StatusCode)
	} else {
		result.Error = fmt.Errorf("client error: %d - %s", resp.StatusCode, string(respBody))
	}

	return result
}

// randomString generates a random alphanumeric string
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Text samples for load testing - mix of safe and potentially flagged content
var textSamples = []string{
	// Normal texts
	"Hello, I need help with my account settings. Can you assist me?",
	"What's the weather forecast for tomorrow in San Francisco?",
	"Can you explain how machine learning works in simple terms?",
	"I'm looking for a good recipe for chocolate chip cookies.",
	"What are the best practices for writing clean code?",
	"How do I reset my password for the application?",
	"Tell me about the history of the Roman Empire.",
	"What programming language should I learn first?",
	"Can you help me debug this Python script?",
	"What's the difference between REST and GraphQL APIs?",
	
	// Longer texts
	"I've been working on this project for several weeks now, and I'm running into some issues with the database queries. The response times are getting slower as the dataset grows. I've tried adding indexes but it doesn't seem to help much. Any suggestions for optimizing PostgreSQL performance?",
	"Our team is evaluating different cloud providers for our infrastructure. We need to consider factors like cost, scalability, and the learning curve for our developers. Currently we're on AWS but considering a move to GCP or Azure. What are the key differences we should be aware of?",
	
	// Edge cases with special characters
	"Test with émojis 🎉 and spëcial çharacters!",
	"SELECT * FROM users WHERE id = 1; DROP TABLE users;--",
	"<script>alert('test')</script>",
	
	// Mixed content
	"This is a normal message followed by some numbers: 123-45-6789 and an email: test@example.com",
}

// generateRandomText returns a random text sample for testing
func generateRandomText() string {
	return textSamples[rand.Intn(len(textSamples))]
}
