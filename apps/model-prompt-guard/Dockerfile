FROM python:3.12-slim

WORKDIR /app

# Install uv
RUN pip install uv

# Copy workspace files first (for layer caching)
COPY pyproject.toml uv.lock* ./
COPY packages/py-common/pyproject.toml ./packages/py-common/
COPY packages/py-common/src ./packages/py-common/src
COPY apps/model-prompt-guard/pyproject.toml ./apps/model-prompt-guard/

# Install dependencies
RUN uv sync --frozen --no-dev 2>/dev/null || uv sync --no-dev

# Copy application code
COPY apps/model-prompt-guard/src ./apps/model-prompt-guard/src

# Set working directory to app
WORKDIR /app/apps/model-prompt-guard

# Expose port
EXPOSE 8000

# Run the application
CMD ["uv", "run", "uvicorn", "model.main:app", "--host", "0.0.0.0", "--port", "8000"]
