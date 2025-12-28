"""Guardrail Server - Main Application Entry Point.

This is the FastAPI application that orchestrates ML model calls
for LLM guardrail validation.

To run:
    uv run uvicorn guardrail.main:app --reload --host 0.0.0.0 --port 8000
"""

from contextlib import asynccontextmanager
import structlog

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from guardrail.config import settings
from guardrail.api.routes import router, debug_router
from guardrail.models.client import close_all_clients


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
        - Close HTTP clients
        - (Future) Close database connections
    """
    # Startup
    logger.info(
        "guardrail_server_starting",
        host=settings.HOST,
        port=settings.PORT,
        model_urls=settings.model_urls,
        model_timeout=settings.MODEL_TIMEOUT_SECONDS,
        cb_failure_threshold=settings.CB_FAILURE_THRESHOLD,
    )
    
    yield
    
    # Shutdown
    logger.info("guardrail_server_shutting_down")
    await close_all_clients()


# Create FastAPI application
app = FastAPI(
    title="Guardrail API",
    description="Real-time LLM safety validation service",
    version="0.1.0",
    lifespan=lifespan,
)

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
