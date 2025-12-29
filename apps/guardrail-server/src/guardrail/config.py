"""Configuration settings for the Guardrail Server.

Uses pydantic-settings for environment variable management with validation.
"""

from pydantic_settings import BaseSettings, SettingsConfigDict
from pydantic import Field


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    # Server configuration
    HOST: str = Field(default="0.0.0.0", description="Server host")
    PORT: int = Field(default=8000, description="Server port")
    DEBUG: bool = Field(default=False, description="Debug mode")

    # Redis configuration
    REDIS_URL: str = Field(
        default="redis://localhost:6379/0",
        description="Redis connection URL",
    )
    REDIS_CACHE_TTL: int = Field(
        default=300,
        description="Cache TTL in seconds (default 5 min)",
    )

    # PostgreSQL configuration
    DATABASE_URL: str = Field(
        default="postgresql://postgres:postgres@localhost:5432/guardrails",
        description="PostgreSQL connection URL",
    )

    # Model service URLs
    MODEL_PROMPT_GUARD_URL: str = Field(
        default="http://model-prompt-guard:8000",
        description="Prompt guard model service URL",
    )
    MODEL_PII_DETECT_URL: str = Field(
        default="http://model-pii-detect:8000",
        description="PII detect model service URL",
    )
    MODEL_HATE_DETECT_URL: str = Field(
        default="http://model-hate-detect:8000",
        description="Hate detect model service URL",
    )
    MODEL_CONTENT_CLASS_URL: str = Field(
        default="http://model-content-class:8000",
        description="Content classification model service URL",
    )

    # Model call configuration
    MODEL_TIMEOUT_SECONDS: float = Field(
        default=0.08,
        description="Timeout for model calls in seconds (80ms default)",
    )
    MODEL_CONNECT_TIMEOUT: float = Field(
        default=0.02,
        description="Connection timeout for model calls (20ms default)",
    )

    # Circuit breaker configuration
    CB_FAILURE_THRESHOLD: int = Field(
        default=5,
        description="Number of failures before circuit opens",
    )
    CB_RECOVERY_TIMEOUT: float = Field(
        default=30.0,
        description="Seconds before attempting recovery (half-open)",
    )
    CB_SUCCESS_THRESHOLD: int = Field(
        default=3,
        description="Successes needed to close circuit from half-open",
    )

    # Note: Rate limiting is handled by Contour/Envoy at the ingress layer
    # See docs/CONTOUR_ARCHITECTURE.md for configuration

    @property
    def model_urls(self) -> dict[str, str]:
        """Return mapping of model names to URLs."""
        return {
            "prompt-guard": self.MODEL_PROMPT_GUARD_URL,
            "pii-detect": self.MODEL_PII_DETECT_URL,
            "hate-detect": self.MODEL_HATE_DETECT_URL,
            "content-class": self.MODEL_CONTENT_CLASS_URL,
        }


# Global settings instance
settings = Settings()
