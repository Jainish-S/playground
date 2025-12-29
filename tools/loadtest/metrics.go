package main

import (
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// Metrics collects and aggregates load test metrics
type Metrics struct {
	mu sync.Mutex

	startTime time.Time
	endTime   time.Time

	// Global histogram (microseconds for precision)
	histogram *hdrhistogram.Histogram

	// Per-tenant histograms
	tenantHistograms map[string]*hdrhistogram.Histogram
	tenantSuccess    map[string]int64
	tenantTotal      map[string]int64

	// Counters
	totalRequests int64
	successCount  int64
	timeoutCount  int64
	errorCount    int64
	flaggedCount  int64
	rateLimited   int64
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		// HDR Histogram: 1us to 60s range, 3 significant figures
		histogram:        hdrhistogram.New(1, 60_000_000, 3),
		tenantHistograms: make(map[string]*hdrhistogram.Histogram),
		tenantSuccess:    make(map[string]int64),
		tenantTotal:      make(map[string]int64),
	}
}

// Start marks the beginning of the test
func (m *Metrics) Start() {
	m.mu.Lock()
	m.startTime = time.Now()
	m.mu.Unlock()
}

// Stop marks the end of the test
func (m *Metrics) Stop() {
	m.mu.Lock()
	m.endTime = time.Now()
	m.mu.Unlock()
}

// Record adds a request result to the metrics
func (m *Metrics) Record(result RequestResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++

	// Record latency in microseconds
	latencyUs := result.Latency.Microseconds()
	if latencyUs > 0 {
		m.histogram.RecordValue(latencyUs)

		// Per-tenant histogram
		if _, ok := m.tenantHistograms[result.TenantID]; !ok {
			m.tenantHistograms[result.TenantID] = hdrhistogram.New(1, 60_000_000, 3)
		}
		m.tenantHistograms[result.TenantID].RecordValue(latencyUs)
	}

	m.tenantTotal[result.TenantID]++

	if result.Success {
		m.successCount++
		m.tenantSuccess[result.TenantID]++
		if result.Flagged {
			m.flaggedCount++
		}
	} else if result.Timeout {
		m.timeoutCount++
	} else if result.Error != nil && result.Error.Error() == "rate limited" {
		m.rateLimited++
	} else {
		m.errorCount++
	}
}

// Results represents the final test results
type Results struct {
	Duration      time.Duration `json:"duration"`
	TargetRPS     int           `json:"target_rps"`
	AchievedRPS   float64       `json:"achieved_rps"`
	TotalRequests int64         `json:"total_requests"`

	// Latency percentiles in milliseconds
	LatencyP50 float64 `json:"latency_p50_ms"`
	LatencyP90 float64 `json:"latency_p90_ms"`
	LatencyP95 float64 `json:"latency_p95_ms"`
	LatencyP99 float64 `json:"latency_p99_ms"`
	LatencyMax float64 `json:"latency_max_ms"`
	LatencyMin float64 `json:"latency_min_ms"`
	LatencyAvg float64 `json:"latency_avg_ms"`

	// Counts
	SuccessCount int64 `json:"success_count"`
	TimeoutCount int64 `json:"timeout_count"`
	ErrorCount   int64 `json:"error_count"`
	FlaggedCount int64 `json:"flagged_count"`
	RateLimited  int64 `json:"rate_limited_count"`

	// Per-tenant results
	TenantResults []TenantResult `json:"tenant_results,omitempty"`
}

// TenantResult holds per-tenant metrics
type TenantResult struct {
	TenantID    string  `json:"tenant_id"`
	Requests    int64   `json:"requests"`
	SuccessRate float64 `json:"success_rate"`
	LatencyP50  float64 `json:"latency_p50_ms"`
	LatencyP99  float64 `json:"latency_p99_ms"`
}

// GetResults computes the final results
func (m *Metrics) GetResults(targetRPS int) *Results {
	m.mu.Lock()
	defer m.mu.Unlock()

	duration := m.endTime.Sub(m.startTime)
	if duration == 0 {
		duration = time.Second // Avoid division by zero
	}

	results := &Results{
		Duration:      duration,
		TargetRPS:     targetRPS,
		AchievedRPS:   float64(m.totalRequests) / duration.Seconds(),
		TotalRequests: m.totalRequests,

		// Convert microseconds to milliseconds
		LatencyP50: float64(m.histogram.ValueAtPercentile(50)) / 1000.0,
		LatencyP90: float64(m.histogram.ValueAtPercentile(90)) / 1000.0,
		LatencyP95: float64(m.histogram.ValueAtPercentile(95)) / 1000.0,
		LatencyP99: float64(m.histogram.ValueAtPercentile(99)) / 1000.0,
		LatencyMax: float64(m.histogram.Max()) / 1000.0,
		LatencyMin: float64(m.histogram.Min()) / 1000.0,
		LatencyAvg: m.histogram.Mean() / 1000.0,

		SuccessCount: m.successCount,
		TimeoutCount: m.timeoutCount,
		ErrorCount:   m.errorCount,
		FlaggedCount: m.flaggedCount,
		RateLimited:  m.rateLimited,
	}

	// Per-tenant results
	for tenantID, hist := range m.tenantHistograms {
		total := m.tenantTotal[tenantID]
		success := m.tenantSuccess[tenantID]
		successRate := float64(0)
		if total > 0 {
			successRate = float64(success) / float64(total)
		}

		results.TenantResults = append(results.TenantResults, TenantResult{
			TenantID:    tenantID,
			Requests:    total,
			SuccessRate: successRate,
			LatencyP50:  float64(hist.ValueAtPercentile(50)) / 1000.0,
			LatencyP99:  float64(hist.ValueAtPercentile(99)) / 1000.0,
		})
	}

	return results
}
