"""Prompt Guard Model Service - FastAPI Application.

Detects prompt injection attacks in LLM inputs.
Currently uses dummy keyword-based detection - will be replaced with ML model.
"""

import asyncio
import random
import time
from contextlib import asynccontextmanager
from concurrent.futures import ThreadPoolExecutor

from fastapi import FastAPI, HTTPException
from py_common.schemas import ModelPredictRequest, ModelPredictResponse
from py_common.metrics import setup_metrics, INFERENCE_LATENCY, INFERENCE_TOTAL

from model.config import settings
from model.inference import detect_prompt_injection

MODEL_NAME = "prompt-guard"

# Global shutdown state
_shutting_down = False

# Thread pool for blocking inference (max_workers=1 for serial processing)
# Keeps event loop responsive for /metrics, /health, /ready endpoints
_executor = ThreadPoolExecutor(max_workers=1, thread_name_prefix="inference")

# Semaphore to queue requests when inference is busy
_inference_semaphore = asyncio.Semaphore(1)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan - model loading would happen here."""
    global _shutting_down

    # Startup
    print(f"[{MODEL_NAME}] Starting with dummy inference (keyword-based)")
    if settings.INFERENCE_DELAY_ENABLED:
        print(f"[{MODEL_NAME}] Inference delay enabled: {settings.INFERENCE_DELAY_MIN_MS}-{settings.INFERENCE_DELAY_MAX_MS}ms")

    yield

    # Shutdown sequence
    _shutting_down = True
    print(f"[{MODEL_NAME}] Shutdown initiated, draining requests...")
    await asyncio.sleep(0.5)  # Brief drain period
    print(f"[{MODEL_NAME}] Shutdown complete")


app = FastAPI(
    title="Prompt Guard Model",
    description="Detects prompt injection attacks",
    version="0.1.0",
    lifespan=lifespan,
)

# Setup Prometheus metrics
setup_metrics(app, MODEL_NAME)


def _blocking_inference(text: str, request_id: str) -> tuple[bool, float, str]:
    """CPU-bound inference that blocks the calling thread.

    Runs in thread pool to keep event loop responsive for metrics.
    """
    # Simulate ML inference latency (BLOCKING like real ML)
    if settings.INFERENCE_DELAY_ENABLED:
        delay_ms = random.randint(
            settings.INFERENCE_DELAY_MIN_MS,
            settings.INFERENCE_DELAY_MAX_MS,
        )
        time.sleep(delay_ms / 1000)  # Blocking sleep
        print(f"[{MODEL_NAME}] Simulated delay: {delay_ms}ms for request {request_id}")

    # Actual inference logic
    return detect_prompt_injection(text)


@app.post("/predict", response_model=ModelPredictResponse)
async def predict(request: ModelPredictRequest) -> ModelPredictResponse:
    """Run prompt injection detection on input text.

    Uses thread pool to run blocking inference while keeping event loop
    responsive for /metrics, /health, /ready endpoints.
    """
    start_time = time.perf_counter()

    try:
        # Queue if inference is busy (max 1 concurrent)
        async with _inference_semaphore:
            # Run blocking inference in thread pool (doesn't block event loop!)
            flagged, score, details = await asyncio.to_thread(
                _blocking_inference,
                request.text,
                request.request_id,
            )

        latency_ms = int((time.perf_counter() - start_time) * 1000)

        # Record metrics
        INFERENCE_LATENCY.labels(model_name=MODEL_NAME).observe(latency_ms / 1000)
        INFERENCE_TOTAL.labels(model_name=MODEL_NAME, status="success").inc()

        return ModelPredictResponse(
            flagged=flagged,
            score=score,
            details=details,
            latency_ms=latency_ms,
        )
    except Exception as e:
        INFERENCE_TOTAL.labels(model_name=MODEL_NAME, status="error").inc()
        raise


@app.get("/health")
async def health():
    """Liveness probe."""
    return {"status": "healthy", "model": MODEL_NAME}


@app.get("/ready")
async def ready():
    """Readiness probe - indicates if service can accept traffic.

    Returns 503 during shutdown to remove pod from load balancer.
    """
    if _shutting_down:
        raise HTTPException(
            status_code=503,
            detail={"status": "draining", "model": MODEL_NAME},
        )
    return {"status": "ready", "model": MODEL_NAME}
