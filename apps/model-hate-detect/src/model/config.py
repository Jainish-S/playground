"""Model configuration settings.

Manages environment variables for the model service including
simulated ML inference latency configuration.
"""

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class ModelSettings(BaseSettings):
    """Configuration for model service behavior."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Simulated inference latency
    INFERENCE_DELAY_ENABLED: bool = Field(
        default=True,
        description="Enable simulated ML inference delay",
    )
    INFERENCE_DELAY_MIN_MS: int = Field(
        default=40,
        description="Minimum simulated inference delay in milliseconds",
    )
    INFERENCE_DELAY_MAX_MS: int = Field(
        default=60,
        description="Maximum simulated inference delay in milliseconds",
    )


settings = ModelSettings()
