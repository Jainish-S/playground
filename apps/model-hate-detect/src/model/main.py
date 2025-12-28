"""Hate Detect Model Service - FastAPI Application.

Detects hate speech and toxic content.
Currently uses dummy keyword-based detection - will be replaced with BERT.
"""

import time
from contextlib import asynccontextmanager

from fastapi import FastAPI
from py_common.schemas import ModelPredictRequest, ModelPredictResponse
from py_common.metrics import setup_metrics, INFERENCE_LATENCY, INFERENCE_TOTAL

from model.inference import detect_hate

MODEL_NAME = "hate-detect"


@asynccontextmanager
async def lifespan(app: FastAPI):
    print(f"[{MODEL_NAME}] Starting with dummy inference (keyword-based)")
    yield
    print(f"[{MODEL_NAME}] Shutting down")


app = FastAPI(
    title="Hate Detect Model",
    description="Detects hate speech and toxic content",
    version="0.1.0",
    lifespan=lifespan,
)

setup_metrics(app, MODEL_NAME)


@app.post("/predict", response_model=ModelPredictResponse)
async def predict(request: ModelPredictRequest) -> ModelPredictResponse:
    start_time = time.perf_counter()

    try:
        flagged, score, details = detect_hate(request.text)
        latency_ms = int((time.perf_counter() - start_time) * 1000)

        INFERENCE_LATENCY.labels(model_name=MODEL_NAME).observe(latency_ms / 1000)
        INFERENCE_TOTAL.labels(model_name=MODEL_NAME, status="success").inc()

        return ModelPredictResponse(
            flagged=flagged, score=score, details=details, latency_ms=latency_ms
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
