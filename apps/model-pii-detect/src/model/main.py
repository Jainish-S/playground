"""PII Detect Model Service - FastAPI Application.

Detects personally identifiable information (PII) in text.
Currently uses dummy regex-based detection - will be replaced with Presidio.
"""

import time
from contextlib import asynccontextmanager

from fastapi import FastAPI
from py_common.schemas import ModelPredictRequest, ModelPredictResponse
from py_common.metrics import setup_metrics, INFERENCE_LATENCY, INFERENCE_TOTAL

from model.inference import detect_pii

MODEL_NAME = "pii-detect"


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan - model loading would happen here."""
    print(f"[{MODEL_NAME}] Starting with dummy inference (regex-based)")
    yield
    print(f"[{MODEL_NAME}] Shutting down")


app = FastAPI(
    title="PII Detect Model",
    description="Detects personally identifiable information",
    version="0.1.0",
    lifespan=lifespan,
)

setup_metrics(app, MODEL_NAME)


@app.post("/predict", response_model=ModelPredictResponse)
async def predict(request: ModelPredictRequest) -> ModelPredictResponse:
    """Run PII detection on input text."""
    start_time = time.perf_counter()

    try:
        flagged, score, details = detect_pii(request.text)
        latency_ms = int((time.perf_counter() - start_time) * 1000)

        INFERENCE_LATENCY.labels(model_name=MODEL_NAME).observe(latency_ms / 1000)
        INFERENCE_TOTAL.labels(model_name=MODEL_NAME, status="success").inc()

        return ModelPredictResponse(
            flagged=flagged,
            score=score,
            details=details,
            latency_ms=latency_ms,
        )
    except Exception:
        INFERENCE_TOTAL.labels(model_name=MODEL_NAME, status="error").inc()
        raise


@app.get("/health")
async def health():
    return {"status": "healthy", "model": MODEL_NAME}


@app.get("/ready")
async def ready():
    return {"status": "ready", "model": MODEL_NAME}
