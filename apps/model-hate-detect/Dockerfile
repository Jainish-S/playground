FROM python:3.12-slim

WORKDIR /app

RUN pip install uv

COPY pyproject.toml uv.lock* ./
COPY packages/py-common/pyproject.toml ./packages/py-common/
COPY packages/py-common/src ./packages/py-common/src
COPY apps/model-hate-detect/pyproject.toml ./apps/model-hate-detect/

RUN uv sync --frozen --no-dev 2>/dev/null || uv sync --no-dev

COPY apps/model-hate-detect/src ./apps/model-hate-detect/src

WORKDIR /app/apps/model-hate-detect
EXPOSE 8000
CMD ["uv", "run", "uvicorn", "model.main:app", "--host", "0.0.0.0", "--port", "8000"]
