"""Guardrail Server - Main Application Entry Point.

This is the FastAPI application that orchestrates ML model calls
for LLM guardrail validation.

To run:
    uv run uvicorn guardrail.main:app --reload --host 0.0.0.0 --port 8000
"""

import asyncio
from contextlib import asynccontextmanager
import structlog

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from guardrail.config import settings
from guardrail.api.routes import router, debug_router
from guardrail.models.client import close_all_clients
from py_common.metrics import setup_metrics

# Global shutdown state
_shutting_down = False


# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer(),
    ],
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager.

    Startup:
        - Log configuration
        - (Future) Initialize Redis connection pool
        - (Future) Initialize PostgreSQL connection pool

    Shutdown:
        - Set shutdown flag
        - Drain in-flight requests
        - Close HTTP clients
        - (Future) Close database connections
    """
    global _shutting_down

    # Startup
    logger.info(
        "guardrail_server_starting",
        host=settings.HOST,
        port=settings.PORT,
        model_urls=settings.model_urls,
        model_timeout=settings.MODEL_TIMEOUT_SECONDS,
        cb_failure_threshold=settings.CB_FAILURE_THRESHOLD,
        retry_enabled=settings.RETRY_ENABLED,
        retry_max_attempts=settings.RETRY_MAX_ATTEMPTS,
    )

    yield

    # Shutdown sequence
    _shutting_down = True
    logger.info("guardrail_server_shutdown_initiated")

    # Wait for in-flight requests to drain (max 5s)
    max_wait = 5.0
    poll_interval = 0.1
    elapsed = 0.0

    while elapsed < max_wait:
        # Import here to avoid circular dependency
        from guardrail.core.orchestrator import IN_FLIGHT

        in_flight = IN_FLIGHT._value._value  # Access internal counter
        if in_flight == 0:
            break
        await asyncio.sleep(poll_interval)
        elapsed += poll_interval

    logger.info(
        "guardrail_server_shutting_down",
        in_flight_drained=(in_flight == 0),
        wait_time_seconds=elapsed,
    )

    await close_all_clients()


# Create FastAPI application
app = FastAPI(
    title="Guardrail API",
    description="Real-time LLM safety validation service",
    version="0.1.0",
    lifespan=lifespan,
)

# Setup metrics (must be before other middleware to track all requests)
setup_metrics(app, "guardrail-server")

# CORS middleware (configure properly in production)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # TODO: Restrict in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(router)
app.include_router(debug_router)


# Root endpoint
@app.get("/")
async def root():
    """Root endpoint with service info."""
    return {
        "service": "guardrail-server",
        "version": "0.1.0",
        "docs": "/docs",
        "health": "/v1/health",
    }
