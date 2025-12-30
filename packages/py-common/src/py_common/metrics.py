"""Prometheus metrics for model services."""

import time
from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
from fastapi import Response


# HTTP request metrics (end-to-end)
HTTP_REQUEST_DURATION = Histogram(
    "http_request_duration_seconds",
    "HTTP request duration in seconds (full request/response cycle)",
    ["model_name", "method", "endpoint", "status_code"],
    buckets=[0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0],
)

# Model inference metrics (just ML execution)
INFERENCE_LATENCY = Histogram(
    "model_inference_latency_seconds",
    "Model inference latency in seconds (ML execution only)",
    ["model_name"],
    buckets=[0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0],
)

INFERENCE_TOTAL = Counter(
    "model_inference_total",
    "Total model inferences",
    ["model_name", "status"],
)

IN_FLIGHT = Gauge(
    "model_in_flight_requests",
    "Number of in-flight requests",
    ["model_name"],
)


def setup_metrics(app, model_name: str):
    """Add metrics endpoint to FastAPI app."""

    @app.get("/metrics")
    async def metrics():
        return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)

    @app.middleware("http")
    async def track_requests(request, call_next):
        """Track in-flight requests and full HTTP request duration."""
        # Skip metrics collection for the /metrics endpoint itself
        if request.url.path == "/metrics":
            return await call_next(request)

        start_time = time.perf_counter()
        IN_FLIGHT.labels(model_name=model_name).inc()

        try:
            response = await call_next(request)

            # Record full HTTP request duration
            duration = time.perf_counter() - start_time
            HTTP_REQUEST_DURATION.labels(
                model_name=model_name,
                method=request.method,
                endpoint=request.url.path,
                status_code=response.status_code,
            ).observe(duration)

            return response
        finally:
            IN_FLIGHT.labels(model_name=model_name).dec()
