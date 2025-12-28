"""Prompt Guard Model Service - FastAPI Application.

Detects prompt injection attacks in LLM inputs.
Currently uses dummy keyword-based detection - will be replaced with ML model.
"""

import time
from contextlib import asynccontextmanager

from fastapi import FastAPI
from py_common.schemas import ModelPredictRequest, ModelPredictResponse
from py_common.metrics import setup_metrics, INFERENCE_LATENCY, INFERENCE_TOTAL

from model.inference import detect_prompt_injection

MODEL_NAME = "prompt-guard"


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan - model loading would happen here."""
    # In production: Load actual ML model here
    print(f"[{MODEL_NAME}] Starting with dummy inference (keyword-based)")
    yield
    print(f"[{MODEL_NAME}] Shutting down")


app = FastAPI(
    title="Prompt Guard Model",
    description="Detects prompt injection attacks",
    version="0.1.0",
    lifespan=lifespan,
)

# Setup Prometheus metrics
setup_metrics(app, MODEL_NAME)


@app.post("/predict", response_model=ModelPredictResponse)
async def predict(request: ModelPredictRequest) -> ModelPredictResponse:
    """Run prompt injection detection on input text."""
    start_time = time.perf_counter()

    try:
        flagged, score, details = detect_prompt_injection(request.text)

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
    """Readiness probe."""
    return {"status": "ready", "model": MODEL_NAME}
