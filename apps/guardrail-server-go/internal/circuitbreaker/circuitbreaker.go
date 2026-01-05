// Package circuitbreaker implements the circuit breaker pattern for fault-tolerant model calls.
//
// Implements the circuit breaker pattern to prevent cascade failures:
//   - CLOSED: Normal operation, requests pass through
//   - OPEN: After N failures, all requests fail immediately
//   - HALF_OPEN: After recovery timeout, allow probe requests
//
// Reference: https://martinfowler.com/bliki/CircuitBreaker.html
package circuitbreaker

import (
	"sync"
	"time"

	gocommon "github.com/playground/packages/go-common"
	"github.com/prometheus/client_golang/prometheus"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed is normal operation
	StateClosed State = 0
	// StateOpen is rejecting all requests
	StateOpen State = 1
	// StateHalfOpen is testing if service recovered
	StateHalfOpen State = 2
)

// String returns the state name.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker provides circuit breaker functionality for a single model service.
type CircuitBreaker struct {
	name             string
	failureThreshold int
	recoveryTimeout  time.Duration
	successThreshold int

	mu              sync.Mutex
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time

	// Prometheus gauge for state
	stateGauge *prometheus.GaugeVec
}

// New creates a new circuit breaker.
func New(name string, failureThreshold, successThreshold int, recoveryTimeout time.Duration, stateGauge *prometheus.GaugeVec) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		recoveryTimeout:  recoveryTimeout,
		state:            StateClosed,
		stateGauge:       stateGauge,
	}

	// Initialize metric
	if stateGauge != nil {
		stateGauge.WithLabelValues(name).Set(float64(StateClosed))
	}

	return cb
}

// AllowRequest checks if a request should be allowed.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check for recovery timeout
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.recoveryTimeout {
			cb.transitionTo(StateHalfOpen)
		}
	}

	switch cb.state {
	case StateClosed, StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		cb.failureCount = 0
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		}
	}
}

// transitionTo transitions to a new state. Must be called with lock held.
func (cb *CircuitBreaker) transitionTo(newState State) {
	cb.state = newState

	if cb.stateGauge != nil {
		cb.stateGauge.WithLabelValues(cb.name).Set(float64(newState))
	}

	// Reset counters on state change
	switch newState {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.successCount = 0
	}
}

// ForceClose forces the circuit to close.
func (cb *CircuitBreaker) ForceClose() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}

// ForceOpen forces the circuit to open.
func (cb *CircuitBreaker) ForceOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.lastFailureTime = time.Now()
	cb.transitionTo(StateOpen)
}

// GetStatus returns the current status.
func (cb *CircuitBreaker) GetStatus() gocommon.CircuitBreakerStatus {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check for recovery timeout
	currentState := cb.state
	if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.recoveryTimeout {
		currentState = StateHalfOpen
	}

	return gocommon.CircuitBreakerStatus{
		Name:            cb.name,
		State:           currentState.String(),
		FailureCount:    cb.failureCount,
		SuccessCount:    cb.successCount,
		LastFailureTime: float64(cb.lastFailureTime.Unix()),
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check for recovery timeout
	if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.recoveryTimeout {
		cb.transitionTo(StateHalfOpen)
	}

	return cb.state
}

// Registry holds all circuit breakers.
type Registry struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex

	// Config for new breakers
	failureThreshold int
	successThreshold int
	recoveryTimeout  time.Duration
	stateGauge       *prometheus.GaugeVec
}

// NewRegistry creates a new circuit breaker registry.
func NewRegistry(failureThreshold, successThreshold int, recoveryTimeout time.Duration, stateGauge *prometheus.GaugeVec) *Registry {
	return &Registry{
		breakers:         make(map[string]*CircuitBreaker),
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		recoveryTimeout:  recoveryTimeout,
		stateGauge:       stateGauge,
	}
}

// Get returns the circuit breaker for a model, creating it if necessary.
func (r *Registry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, exists := r.breakers[name]
	r.mu.RUnlock()

	if exists {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = r.breakers[name]; exists {
		return cb
	}

	cb = New(name, r.failureThreshold, r.successThreshold, r.recoveryTimeout, r.stateGauge)
	r.breakers[name] = cb
	return cb
}

// GetAll returns all circuit breakers.
func (r *Registry) GetAll() map[string]*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(r.breakers))
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}
