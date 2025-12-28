"""Common Pydantic schemas for ML model services."""

from pydantic import BaseModel, Field


class ModelPredictRequest(BaseModel):
    """Request schema for model prediction endpoints."""

    text: str = Field(..., description="Text to analyze", max_length=50000)
    request_id: str = Field(..., description="Request ID for tracing")


class ModelPredictResponse(BaseModel):
    """Response schema from model prediction endpoints."""

    flagged: bool = Field(..., description="Whether the text was flagged")
    score: float = Field(..., ge=0.0, le=1.0, description="Confidence score")
    details: list[str] = Field(default_factory=list, description="Explanation details")
    latency_ms: int = Field(..., ge=0, description="Inference latency in milliseconds")


class ValidateRequest(BaseModel):
    """Request schema for the main validation endpoint."""

    request_id: str | None = Field(default=None, description="Optional client-provided request ID")
    project_id: str = Field(..., description="Project ID for config lookup")
    text: str = Field(..., max_length=50000, description="Text to validate")
    type: str = Field(default="input", pattern="^(input|output)$", description="Input or output")
    metadata: dict | None = Field(default=None, description="Optional metadata")


class ModelResultResponse(BaseModel):
    """Result from a single model."""

    flagged: bool
    score: float
    details: list[str]
    latency_ms: int


class ValidateResponse(BaseModel):
    """Response schema for the main validation endpoint."""

    request_id: str = Field(..., description="Request ID")
    flagged: bool = Field(..., description="Overall flag status")
    flag_reasons: list[str] = Field(default_factory=list, description="Reasons for flagging")
    model_results: dict[str, ModelResultResponse] = Field(
        default_factory=dict, description="Per-model results"
    )
    partial_failure: bool = Field(default=False, description="Some models failed")
    failed_models: list[str] = Field(default_factory=list, description="Models that failed")
    latency_ms: int = Field(..., ge=0, description="Total request latency")
