FROM python:3.12-slim

WORKDIR /app

# Install uv
RUN pip install uv

# Copy workspace files first (for layer caching)
COPY pyproject.toml uv.lock* ./
COPY packages/py-common/pyproject.toml ./packages/py-common/
COPY packages/py-common/src ./packages/py-common/src
COPY apps/guardrail-server/pyproject.toml ./apps/guardrail-server/

# Install dependencies
RUN uv sync --frozen --no-dev 2>/dev/null || uv sync --no-dev

# Copy application code
COPY apps/guardrail-server/src ./apps/guardrail-server/src

# Set working directory
WORKDIR /app/apps/guardrail-server

# Expose port
EXPOSE 8000

# Run the application
CMD ["uv", "run", "uvicorn", "guardrail.main:app", "--host", "0.0.0.0", "--port", "8000"]
