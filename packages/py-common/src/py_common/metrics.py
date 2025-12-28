"""Prometheus metrics for model services."""

from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
from fastapi import Response


# Model inference metrics
INFERENCE_LATENCY = Histogram(
    "model_inference_latency_seconds",
    "Model inference latency in seconds",
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
    async def track_in_flight(request, call_next):
        IN_FLIGHT.labels(model_name=model_name).inc()
        try:
            response = await call_next(request)
            return response
        finally:
            IN_FLIGHT.labels(model_name=model_name).dec()
