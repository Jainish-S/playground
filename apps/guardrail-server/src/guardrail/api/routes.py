"""API routes for the Guardrail Server.

Endpoints:
- POST /v1/validate - Main validation endpoint
- GET /v1/health - Liveness probe
- GET /v1/ready - Readiness probe
- GET /metrics - Prometheus metrics
- GET /debug/circuit-breakers - Circuit breaker status (debug)
"""

from fastapi import APIRouter, Header, HTTPException, Response
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

from py_common.schemas import ValidateRequest, ValidateResponse
from guardrail.core.orchestrator import validate_text, AggregationStrategy
from guardrail.core.circuit_breaker import get_all_circuit_breakers, get_circuit_breaker


router = APIRouter()


@router.post("/v1/validate", response_model=ValidateResponse)
async def validate(
    request: ValidateRequest,
    x_api_key: str = Header(..., alias="X-API-Key"),
) -> ValidateResponse:
    """Validate text against guardrail models.
    
    This endpoint:
    1. Validates the API key (placeholder - no auth currently)
    2. Calls all enabled models in parallel
    3. Returns aggregated results
    
    Headers:
        X-API-Key: API key for authentication (placeholder)
        
    Request body:
        project_id: Project ID for config lookup
        text: Text to validate
        type: "input" or "output"
        
    Returns:
        ValidateResponse with flag status and model results
    """
    # TODO: Validate API key against database/cache
    # For now, accept any key
    if not x_api_key:
        raise HTTPException(status_code=401, detail="API key required")
    
    # TODO: Get project config for enabled models and thresholds
    # For now, use all models with default strategy
    enabled_models = None  # None = all models
    strategy = AggregationStrategy.ANY_FLAG
    
    result = await validate_text(
        text=request.text,
        enabled_models=enabled_models,
        strategy=strategy,
        request_id=request.request_id,
    )
    
    # If all models failed, return 503
    if result.partial_failure and len(result.failed_models) == 4:
        raise HTTPException(
            status_code=503,
            detail="All model services unavailable",
        )
    
    return result


@router.get("/v1/health")
async def health():
    """Liveness probe - is the process alive?"""
    return {"status": "healthy"}


@router.get("/v1/ready")
async def ready():
    """Readiness probe - can handle traffic?
    
    Checks that at least one model circuit breaker is closed.
    """
    circuit_breakers = get_all_circuit_breakers()
    
    # Check if at least one model is available
    available_models = []
    for name, cb in circuit_breakers.items():
        if cb.allow_request():
            available_models.append(name)
    
    # If no circuit breakers exist yet, we're ready (they'll be created on first request)
    if not circuit_breakers:
        return {"status": "ready", "available_models": "all (not initialized)"}
    
    if available_models:
        return {"status": "ready", "available_models": available_models}
    else:
        raise HTTPException(
            status_code=503,
            detail="No models available (all circuit breakers open)",
        )


@router.get("/metrics")
async def metrics():
    """Prometheus metrics endpoint."""
    return Response(content=generate_latest(), media_type=CONTENT_TYPE_LATEST)


# Debug endpoints (should be protected in production)
debug_router = APIRouter(prefix="/debug", tags=["debug"])


@debug_router.get("/circuit-breakers")
async def get_circuit_breaker_status():
    """Get status of all circuit breakers."""
    circuit_breakers = get_all_circuit_breakers()
    return {
        name: cb.get_status()
        for name, cb in circuit_breakers.items()
    }


@debug_router.post("/circuit-breakers/{model_name}/close")
async def force_close_circuit_breaker(model_name: str):
    """Force close a circuit breaker (for recovery)."""
    cb = get_circuit_breaker(model_name)
    cb.force_close()
    return {"message": f"Circuit breaker for {model_name} forced closed"}


@debug_router.post("/circuit-breakers/{model_name}/open")
async def force_open_circuit_breaker(model_name: str):
    """Force open a circuit breaker (for testing)."""
    cb = get_circuit_breaker(model_name)
    cb.force_open()
    return {"message": f"Circuit breaker for {model_name} forced open"}
