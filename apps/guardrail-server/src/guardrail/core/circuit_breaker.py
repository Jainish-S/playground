"""Circuit breaker implementation for fault-tolerant model calls.

Implements the circuit breaker pattern to prevent cascade failures:
- CLOSED: Normal operation, requests pass through
- OPEN: After N failures, all requests fail immediately
- HALF_OPEN: After recovery timeout, allow probe requests

Reference: https://martinfowler.com/bliki/CircuitBreaker.html
"""

import asyncio
import time
from enum import Enum
from dataclasses import dataclass, field
from prometheus_client import Gauge

from guardrail.config import settings


class CircuitState(Enum):
    """Circuit breaker states."""
    CLOSED = 0  # Normal operation
    OPEN = 1    # Rejecting all requests
    HALF_OPEN = 2  # Testing if service recovered


# Prometheus metric for circuit breaker state
CIRCUIT_STATE_GAUGE = Gauge(
    "guardrail_circuit_breaker_state",
    "Circuit breaker state (0=closed, 1=open, 2=half_open)",
    ["model_name"],
)


@dataclass
class CircuitBreaker:
    """Circuit breaker for a single model service.
    
    Usage:
        cb = CircuitBreaker("prompt-guard")
        
        if not cb.allow_request():
            raise CircuitOpenError()
        
        try:
            result = await call_model()
            cb.record_success()
        except Exception:
            cb.record_failure()
    """
    
    name: str
    failure_threshold: int = field(default_factory=lambda: settings.CB_FAILURE_THRESHOLD)
    recovery_timeout: float = field(default_factory=lambda: settings.CB_RECOVERY_TIMEOUT)
    success_threshold: int = field(default_factory=lambda: settings.CB_SUCCESS_THRESHOLD)
    
    # Internal state
    _state: CircuitState = field(default=CircuitState.CLOSED, init=False)
    _failure_count: int = field(default=0, init=False)
    _success_count: int = field(default=0, init=False)
    _last_failure_time: float = field(default=0.0, init=False)
    _lock: asyncio.Lock = field(default_factory=asyncio.Lock, init=False)
    
    def __post_init__(self):
        """Initialize metrics after dataclass creation."""
        CIRCUIT_STATE_GAUGE.labels(model_name=self.name).set(self._state.value)
    
    @property
    def state(self) -> CircuitState:
        """Get current circuit state, checking for timeout recovery."""
        if self._state == CircuitState.OPEN:
            # Check if recovery timeout has passed
            if time.monotonic() - self._last_failure_time >= self.recovery_timeout:
                self._transition_to(CircuitState.HALF_OPEN)
        return self._state
    
    def _transition_to(self, new_state: CircuitState) -> None:
        """Transition to a new state and update metrics."""
        old_state = self._state
        self._state = new_state
        CIRCUIT_STATE_GAUGE.labels(model_name=self.name).set(new_state.value)
        
        # Reset counters on state change
        if new_state == CircuitState.CLOSED:
            self._failure_count = 0
        elif new_state == CircuitState.HALF_OPEN:
            self._success_count = 0
    
    def allow_request(self) -> bool:
        """Check if a request should be allowed.
        
        Returns:
            True if request can proceed, False if circuit is open
        """
        current_state = self.state  # Triggers timeout check
        
        if current_state == CircuitState.CLOSED:
            return True
        elif current_state == CircuitState.HALF_OPEN:
            # Allow probe requests in half-open state
            return True
        else:  # OPEN
            return False
    
    async def record_success(self) -> None:
        """Record a successful request."""
        async with self._lock:
            if self._state == CircuitState.HALF_OPEN:
                self._success_count += 1
                if self._success_count >= self.success_threshold:
                    self._transition_to(CircuitState.CLOSED)
            elif self._state == CircuitState.CLOSED:
                # Reset failure count on success
                self._failure_count = 0
    
    async def record_failure(self) -> None:
        """Record a failed request."""
        async with self._lock:
            self._failure_count += 1
            self._last_failure_time = time.monotonic()
            
            if self._state == CircuitState.HALF_OPEN:
                # Any failure in half-open goes back to open
                self._transition_to(CircuitState.OPEN)
            elif self._state == CircuitState.CLOSED:
                if self._failure_count >= self.failure_threshold:
                    self._transition_to(CircuitState.OPEN)
    
    def force_close(self) -> None:
        """Force the circuit to close (for debugging/recovery)."""
        self._transition_to(CircuitState.CLOSED)
    
    def force_open(self) -> None:
        """Force the circuit to open (for testing)."""
        self._last_failure_time = time.monotonic()
        self._transition_to(CircuitState.OPEN)
    
    def get_status(self) -> dict:
        """Get current circuit breaker status."""
        return {
            "name": self.name,
            "state": self.state.name,
            "failure_count": self._failure_count,
            "success_count": self._success_count,
            "last_failure_time": self._last_failure_time,
        }


class CircuitOpenError(Exception):
    """Raised when circuit breaker is open."""
    
    def __init__(self, model_name: str):
        self.model_name = model_name
        super().__init__(f"Circuit breaker open for model: {model_name}")


# Global circuit breakers for each model
_circuit_breakers: dict[str, CircuitBreaker] = {}


def get_circuit_breaker(model_name: str) -> CircuitBreaker:
    """Get or create a circuit breaker for a model."""
    if model_name not in _circuit_breakers:
        _circuit_breakers[model_name] = CircuitBreaker(name=model_name)
    return _circuit_breakers[model_name]


def get_all_circuit_breakers() -> dict[str, CircuitBreaker]:
    """Get all circuit breakers (for debugging)."""
    return _circuit_breakers
