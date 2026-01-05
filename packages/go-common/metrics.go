// Package gocommon provides Prometheus metrics for the guardrails platform.
package gocommon

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Pre-defined histogram buckets for latency metrics
var (
	// HTTPLatencyBuckets are latency buckets for full HTTP request/response cycle
	HTTPLatencyBuckets = []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0}

	// InferenceLatencyBuckets are latency buckets for ML inference only
	InferenceLatencyBuckets = []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0}

	// GuardrailLatencyBuckets are latency buckets for guardrail requests
	GuardrailLatencyBuckets = []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.5}

	// ModelCallLatencyBuckets are latency buckets for downstream model calls
	ModelCallLatencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0}
)

// Metrics holds all Prometheus metrics for a service.
type Metrics struct {
	// HTTPRequestDuration tracks full HTTP request duration
	HTTPRequestDuration *prometheus.HistogramVec

	// InferenceLatency tracks ML inference latency (model services only)
	InferenceLatency *prometheus.HistogramVec

	// InferenceTotal tracks total inference count
	InferenceTotal *prometheus.CounterVec

	// InFlightRequests tracks currently processing requests
	InFlightRequests *prometheus.GaugeVec

	// RequestLatency tracks guardrail request latency (orchestrator only)
	RequestLatency prometheus.Histogram

	// RequestTotal tracks total guardrail requests
	RequestTotal *prometheus.CounterVec

	// ModelCallLatency tracks downstream model call latency
	ModelCallLatency *prometheus.HistogramVec

	// ModelCallRetries tracks retry attempts
	ModelCallRetries *prometheus.CounterVec

	// CircuitBreakerState tracks circuit breaker states
	CircuitBreakerState *prometheus.GaugeVec

	// ServiceName is the name of this service
	ServiceName string
}

// NewModelMetrics creates metrics for a model service.
func NewModelMetrics(serviceName string) *Metrics {
	m := &Metrics{
		ServiceName: serviceName,
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds (full request/response cycle)",
				Buckets: HTTPLatencyBuckets,
			},
			[]string{"model_name", "method", "endpoint", "status_code"},
		),
		InferenceLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "model_inference_latency_seconds",
				Help:    "Model inference latency in seconds (ML execution only)",
				Buckets: InferenceLatencyBuckets,
			},
			[]string{"model_name"},
		),
		InferenceTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "model_inference_total",
				Help: "Total model inferences",
			},
			[]string{"model_name", "status"},
		),
		InFlightRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "model_in_flight_requests",
				Help: "Number of in-flight requests",
			},
			[]string{"model_name", "pod"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.HTTPRequestDuration,
		m.InferenceLatency,
		m.InferenceTotal,
		m.InFlightRequests,
	)

	// Get hostname (pod name)
	hostname, _ := os.Hostname()

	// Pre-initialize labels
	m.InferenceLatency.WithLabelValues(serviceName)
	m.InferenceTotal.WithLabelValues(serviceName, "success")
	m.InferenceTotal.WithLabelValues(serviceName, "error")
	// Initialize to 0 so it's exposed immediately (crucial for HPA)
	m.InFlightRequests.WithLabelValues(serviceName, hostname).Set(0)

	return m
}

// NewGuardrailMetrics creates metrics for the guardrail orchestrator.
func NewGuardrailMetrics(serviceName string) *Metrics {
	modelNames := []string{"prompt-guard", "pii-detect", "hate-detect", "content-class"}

	m := &Metrics{
		ServiceName: serviceName,
		RequestLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "guardrail_request_latency_seconds",
				Help:    "Total request latency",
				Buckets: GuardrailLatencyBuckets,
			},
		),
		RequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "guardrail_request_total",
				Help: "Total requests",
			},
			[]string{"status", "flagged"},
		),
		InFlightRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "guardrail_in_flight_requests",
				Help: "Number of in-flight requests",
			},
			[]string{"pod"},
		),
		ModelCallLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "guardrail_model_call_latency_seconds",
				Help:    "Latency of downstream model calls",
				Buckets: ModelCallLatencyBuckets,
			},
			[]string{"model_name"},
		),
		ModelCallRetries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "guardrail_model_call_retries_total",
				Help: "Total number of retries for model calls",
			},
			[]string{"model_name", "retry_number"},
		),
		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "guardrail_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half_open)",
			},
			[]string{"model_name"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.RequestLatency,
		m.RequestTotal,
		m.InFlightRequests,
		m.ModelCallLatency,
		m.ModelCallRetries,
		m.CircuitBreakerState,
	)

	// Get hostname (pod name)
	hostname, _ := os.Hostname()

	// Initialize InFlightRequests to 0 (crucial for HPA)
	m.InFlightRequests.WithLabelValues(hostname).Set(0)

	// Pre-initialize labels for all models
	for _, name := range modelNames {
		m.ModelCallLatency.WithLabelValues(name)
		m.CircuitBreakerState.WithLabelValues(name).Set(0) // CLOSED
	}

	return m
}

// MetricsHandler returns an http.Handler for the /metrics endpoint.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware returns middleware that tracks HTTP request metrics.
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics collection for /metrics endpoint
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		hostname, _ := os.Hostname()

		// Helper to invoke WithLabelValues correct based on labels
		// We have to hack this a bit since Metrics struct is shared but labels differ
		// Ideally we would split the struct, but for now we check if ModelCallLatency exists (only in Guardrail)
		isGuardrail := m.ModelCallLatency != nil

		if isGuardrail {
			m.InFlightRequests.WithLabelValues(hostname).Inc()
		} else {
			m.InFlightRequests.WithLabelValues(m.ServiceName, hostname).Inc()
		}

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		defer func() {
			duration := time.Since(start).Seconds()
			if isGuardrail {
				m.InFlightRequests.WithLabelValues(hostname).Dec()
			} else {
				m.InFlightRequests.WithLabelValues(m.ServiceName, hostname).Dec()
			}

			m.HTTPRequestDuration.WithLabelValues(
				m.ServiceName,
				r.Method,
				r.URL.Path,
				strconv.Itoa(wrapped.statusCode),
			).Observe(duration)
		}()

		next.ServeHTTP(wrapped, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
