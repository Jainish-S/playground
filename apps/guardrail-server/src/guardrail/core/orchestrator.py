"""Orchestrator for parallel model calls with fault tolerance.

The orchestrator is the core of the guardrail server. It:
1. Fans out requests to all enabled models in parallel
2. Uses circuit breakers for fault tolerance
3. Aggregates results using configurable strategy
4. Handles partial failures gracefully

Latency budget (targeting <100ms total):
- Model calls: 30-60ms (parallel, slowest wins)
- Aggregation: 1-2ms
- Network overhead: 5-10ms
"""

import asyncio
import time
import uuid
from dataclasses import dataclass
from enum import Enum

import httpx
import structlog
from prometheus_client import Histogram, Counter, Gauge
from tenacity import (
    retry,
    stop_after_attempt,
    wait_fixed,
    retry_if_exception_type,
    RetryCallState,
)

from py_common.schemas import ModelPredictResponse, ValidateResponse, ModelResultResponse
from guardrail.config import settings
from guardrail.core.circuit_breaker import (
    get_circuit_breaker,
    CircuitOpenError,
    CircuitState,
)
from guardrail.models.client import get_shared_client

logger = structlog.get_logger()


class AggregationStrategy(Enum):
    """Strategy for aggregating model results."""
    ANY_FLAG = "any_flag"      # Flag if ANY model flags
    ALL_FLAG = "all_flag"      # Flag only if ALL models flag
    MAJORITY = "majority"       # Flag if majority (>50%) flag
    THRESHOLD = "threshold"     # Flag if weighted score exceeds threshold


# Prometheus metrics
REQUEST_LATENCY = Histogram(
    "guardrail_request_latency_seconds",
    "Total request latency",
    buckets=[0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.5],
)

REQUEST_TOTAL = Counter(
    "guardrail_request_total",
    "Total requests",
    ["status", "flagged"],
)

IN_FLIGHT = Gauge(
    "guardrail_in_flight_requests",
    "Number of in-flight requests",
)

MODEL_CALL_LATENCY = Histogram(
    "guardrail_model_call_latency_seconds",
    "Latency of downstream model calls",
    ["model_name"],
    buckets=[0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0],
)

RETRY_COUNT = Counter(
    "guardrail_model_call_retries_total",
    "Total number of retries for model calls",
    ["model_name", "retry_number"],
)

# Pre-initialize histogram labels for all models
# This ensures metrics are exposed even before first request
_MODEL_NAMES = ["prompt-guard", "pii-detect", "hate-detect", "content-class"]
for _name in _MODEL_NAMES:
    MODEL_CALL_LATENCY.labels(model_name=_name)

@dataclass
class ModelCallResult:
    """Result from a single model call."""
    model_name: str
    success: bool
    response: ModelPredictResponse | None = None
    error: str | None = None


def before_retry_log(retry_state: RetryCallState) -> None:
    """Log retry attempts and record metrics."""
    if retry_state.outcome and retry_state.outcome.failed:
        exception = retry_state.outcome.exception()
        # Extract model_name from the function arguments
        model_name = retry_state.args[0] if retry_state.args else "unknown"

        logger.warning(
            "model_call_retry",
            model_name=model_name,
            attempt=retry_state.attempt_number,
            exception=str(exception),
        )

        RETRY_COUNT.labels(
            model_name=model_name,
            retry_number=retry_state.attempt_number,
        ).inc()


async def call_model(
    model_name: str,
    text: str,
    request_id: str,
) -> ModelCallResult:
    """Call a single model with circuit breaker and retry protection.

    This function:
    1. Checks circuit breaker state
    2. Makes HTTP call with retry logic if enabled
    3. Records success/failure to circuit breaker

    Args:
        model_name: Name of the model to call
        text: Text to analyze
        request_id: Request ID for tracing

    Returns:
        ModelCallResult with success/failure info
    """
    cb = get_circuit_breaker(model_name)

    # Check circuit breaker - skip if open
    if not cb.allow_request():
        return ModelCallResult(
            model_name=model_name,
            success=False,
            error=f"Circuit breaker open for {model_name}",
        )

    # Inner function with retry logic
    @retry(
        stop=stop_after_attempt(settings.RETRY_MAX_ATTEMPTS),
        wait=wait_fixed(settings.RETRY_WAIT_MS / 1000),
        retry=retry_if_exception_type((
            httpx.TimeoutException,
            httpx.ConnectError,
        )),
        before_sleep=before_retry_log,
        reraise=True,
    )
    async def _call_with_retry():
        """Inner function with retry decorator."""
        start_time = time.perf_counter()

        client = await get_shared_client(model_name)

        response = await client.post(
            "/predict",
            json={"text": text, "request_id": request_id},
        )
        response.raise_for_status()

        # Record latency
        duration = time.perf_counter() - start_time
        MODEL_CALL_LATENCY.labels(model_name=model_name).observe(duration)

        return ModelPredictResponse.model_validate(response.json())

    # Execute with or without retry based on config
    try:
        if settings.RETRY_ENABLED:
            result = await _call_with_retry()
        else:
            # Call directly without retry if disabled
            result = await _call_with_retry.__wrapped__()

        # Record success to circuit breaker
        await cb.record_success()

        return ModelCallResult(
            model_name=model_name,
            success=True,
            response=result,
        )

    except httpx.TimeoutException:
        await cb.record_failure()
        return ModelCallResult(
            model_name=model_name,
            success=False,
            error=f"Timeout calling {model_name} (after {settings.RETRY_MAX_ATTEMPTS} attempts)",
        )
    except httpx.ConnectError:
        await cb.record_failure()
        return ModelCallResult(
            model_name=model_name,
            success=False,
            error=f"Connection error calling {model_name} (after {settings.RETRY_MAX_ATTEMPTS} attempts)",
        )
    except httpx.HTTPStatusError as e:
        await cb.record_failure()
        return ModelCallResult(
            model_name=model_name,
            success=False,
            error=f"HTTP error from {model_name}: {e.response.status_code}",
        )
    except Exception as e:
        await cb.record_failure()
        return ModelCallResult(
            model_name=model_name,
            success=False,
            error=f"Error calling {model_name}: {str(e)}",
        )


async def validate_text(
    text: str,
    enabled_models: list[str] | None = None,
    strategy: AggregationStrategy = AggregationStrategy.ANY_FLAG,
    request_id: str | None = None,
) -> ValidateResponse:
    """Validate text against all enabled models.
    
    This is the main entry point for validation. It:
    1. Fans out to all models in parallel
    2. Collects results (handling failures)
    3. Aggregates using the specified strategy
    
    Args:
        text: Text to validate
        enabled_models: List of model names to use (default: all)
        strategy: Aggregation strategy
        request_id: Optional request ID (generated if not provided)
        
    Returns:
        ValidateResponse with aggregated results
    """
    start_time = time.perf_counter()
    request_id = request_id or str(uuid.uuid4())
    
    # Default to all models
    if enabled_models is None:
        enabled_models = list(settings.model_urls.keys())
    
    # Track in-flight requests
    IN_FLIGHT.inc()
    
    try:
        # Call all models in parallel
        tasks = [
            call_model(model_name, text, request_id)
            for model_name in enabled_models
        ]
        
        results: list[ModelCallResult] = await asyncio.gather(*tasks)
        
        # Process results
        model_results: dict[str, ModelResultResponse] = {}
        failed_models: list[str] = []
        flag_reasons: list[str] = []
        
        for result in results:
            if result.success and result.response:
                model_results[result.model_name] = ModelResultResponse(
                    flagged=result.response.flagged,
                    score=result.response.score,
                    details=result.response.details,
                    latency_ms=result.response.latency_ms,
                )
                
                if result.response.flagged:
                    flag_reasons.append(f"{result.model_name}_flagged")
            else:
                failed_models.append(result.model_name)
        
        # Aggregate results based on strategy
        flagged = aggregate_results(model_results, strategy)
        
        # Calculate total latency
        latency_ms = int((time.perf_counter() - start_time) * 1000)
        
        # Record metrics
        REQUEST_LATENCY.observe(latency_ms / 1000)
        REQUEST_TOTAL.labels(
            status="success" if not failed_models else "partial",
            flagged=str(flagged).lower(),
        ).inc()
        
        return ValidateResponse(
            request_id=request_id,
            flagged=flagged,
            flag_reasons=flag_reasons,
            model_results=model_results,
            partial_failure=len(failed_models) > 0,
            failed_models=failed_models,
            latency_ms=latency_ms,
        )
        
    except Exception as e:
        # Record error metrics
        latency_ms = int((time.perf_counter() - start_time) * 1000)
        REQUEST_TOTAL.labels(status="error", flagged="false").inc()
        raise
        
    finally:
        IN_FLIGHT.dec()


def aggregate_results(
    model_results: dict[str, ModelResultResponse],
    strategy: AggregationStrategy,
) -> bool:
    """Aggregate model results into a single flag decision.
    
    Args:
        model_results: Results from each model
        strategy: Aggregation strategy to use
        
    Returns:
        True if content should be flagged
    """
    if not model_results:
        return False
    
    flags = [r.flagged for r in model_results.values()]
    
    if strategy == AggregationStrategy.ANY_FLAG:
        return any(flags)
    elif strategy == AggregationStrategy.ALL_FLAG:
        return all(flags)
    elif strategy == AggregationStrategy.MAJORITY:
        return sum(flags) > len(flags) / 2
    elif strategy == AggregationStrategy.THRESHOLD:
        # Use average score for threshold strategy
        avg_score = sum(r.score for r in model_results.values()) / len(model_results)
        return avg_score > 0.5
    
    return False
