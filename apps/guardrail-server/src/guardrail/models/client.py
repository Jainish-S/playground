"""HTTP client for calling ML model services.

Handles:
- Connection pooling with httpx
- Timeout configuration
- Retry logic (disabled by default for latency)
- Error handling
"""

import httpx
from dataclasses import dataclass, field
from py_common.schemas import ModelPredictRequest, ModelPredictResponse

from guardrail.config import settings


@dataclass
class ModelClient:
    """Async HTTP client for calling model prediction endpoints."""
    
    model_name: str
    base_url: str
    timeout: float = field(default_factory=lambda: settings.MODEL_TIMEOUT_SECONDS)
    connect_timeout: float = field(default_factory=lambda: settings.MODEL_CONNECT_TIMEOUT)
    _client: httpx.AsyncClient | None = field(default=None, init=False)
    
    async def __aenter__(self) -> "ModelClient":
        """Create async client on context entry."""
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=httpx.Timeout(
                connect=self.connect_timeout,
                read=self.timeout,
                write=self.timeout,
                pool=self.timeout,
            ),
        )
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Close client on context exit."""
        if self._client:
            await self._client.aclose()
    
    async def predict(self, text: str, request_id: str) -> ModelPredictResponse:
        """Call the model's /predict endpoint.
        
        Args:
            text: Text to analyze
            request_id: Request ID for tracing
            
        Returns:
            ModelPredictResponse with prediction results
            
        Raises:
            httpx.TimeoutException: If request times out
            httpx.HTTPStatusError: If server returns error status
        """
        if not self._client:
            raise RuntimeError("Client not initialized. Use 'async with' context manager.")
        
        request = ModelPredictRequest(text=text, request_id=request_id)
        
        response = await self._client.post(
            "/predict",
            json=request.model_dump(),
        )
        response.raise_for_status()
        
        return ModelPredictResponse.model_validate(response.json())
    
    async def health_check(self) -> bool:
        """Check if model service is healthy.
        
        Returns:
            True if healthy, False otherwise
        """
        if not self._client:
            raise RuntimeError("Client not initialized")
        
        try:
            response = await self._client.get("/health")
            return response.status_code == 200
        except Exception:
            return False


# Shared client pool - one client per model
_client_pool: dict[str, httpx.AsyncClient] = {}


async def get_shared_client(model_name: str) -> httpx.AsyncClient:
    """Get or create a shared async client for a model.
    
    Uses connection pooling for efficiency.
    """
    if model_name not in _client_pool:
        base_url = settings.model_urls.get(model_name)
        if not base_url:
            raise ValueError(f"Unknown model: {model_name}")
        
        _client_pool[model_name] = httpx.AsyncClient(
            base_url=base_url,
            timeout=httpx.Timeout(
                connect=settings.MODEL_CONNECT_TIMEOUT,
                read=settings.MODEL_TIMEOUT_SECONDS,
                write=settings.MODEL_TIMEOUT_SECONDS,
                pool=settings.MODEL_TIMEOUT_SECONDS,
            ),
            limits=httpx.Limits(
                max_connections=100,
                max_keepalive_connections=20,
            ),
        )
    return _client_pool[model_name]


async def close_all_clients() -> None:
    """Close all shared clients (call on shutdown)."""
    for client in _client_pool.values():
        await client.aclose()
    _client_pool.clear()
