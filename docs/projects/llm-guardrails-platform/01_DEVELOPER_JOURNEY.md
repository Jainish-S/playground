# LLM Guardrails Platform - Developer Journey

**A step-by-step guide to building this platform from scratch.**

This document outlines the exact sequence of work, from current repository state to production-ready platform. Each phase is self-contained and testable.

---

## Table of Contents

1. [Current State & Prerequisites](#current-state--prerequisites)
2. [Phase 1: Repository Structure Setup](#phase-1-repository-structure-setup-week-1)
3. [Phase 2: Guardrail Server MVP](#phase-2-guardrail-server-mvp-weeks-2-3)
4. [Phase 3: ML Model Services](#phase-3-ml-model-services-week-4)
5. [Phase 4: Platform API](#phase-4-platform-api-weeks-5-6)
6. [Phase 5: Analytics & Workers](#phase-5-analytics--workers-week-7)
7. [Phase 6: Web Dashboard](#phase-6-web-dashboard-week-8)
8. [Phase 7: Production Hardening](#phase-7-production-hardening-weeks-9-10)
9. [Phase 8: Beta & Launch](#phase-8-beta--launch-weeks-11-12)

---

## Current State & Prerequisites

### What We Have

✅ **Infrastructure**: OCI Kubernetes cluster with Twingate access
✅ **Terraform**: Network, OKE, observability stack
✅ **Documentation**: Architecture and design specs
✅ **Git workflow**: Rebase-only, trunk-based development

### What We Need

⬜ Moon workspace configuration
⬜ Python workspace (uv)
⬜ Go workspace (future)
⬜ CI/CD pipeline
⬜ All application code

### Prerequisites

Ensure you have completed: [docs/getting-started/README.md](../../getting-started/README.md)

Required tools:
- `moon` - Task runner
- `uv` - Python package manager
- `kubectl` - Kubernetes CLI
- `docker` - Container runtime

---

## Phase 1: Repository Structure Setup (Week 1)

**Goal**: Set up monorepo structure, moon configuration, and Python workspace.

### Step 1.1: Create Directory Structure

```bash
# From repository root
mkdir -p apps/{guardrail-server,platform-api,analytics-worker,web-dashboard}
mkdir -p packages/{api-contracts,py-common}
mkdir -p .moon
```

**Verify:**
```bash
tree -L 2 -d apps packages .moon
```

### Step 1.2: Initialize Moon Workspace

Create `.moon/workspace.yml`:

```yaml
# .moon/workspace.yml
$schema: 'https://moonrepo.dev/schemas/workspace.json'

# Project discovery
projects:
  globs:
    - 'apps/*'
    - 'packages/*'

# VCS configuration
vcs:
  manager: 'git'
  defaultBranch: 'main'

# Enforce project constraints (optional, can add later)
constraints:
  enforceProjectTypeRelationships: false
```

**Verify:**
```bash
moon query projects
# Should show: (no projects yet, that's expected)
```

### Step 1.3: Configure Moon Toolchain

Create `.moon/toolchain.yml`:

```yaml
# .moon/toolchain.yml
$schema: 'https://moonrepo.dev/schemas/toolchain.json'

# Python configuration
python:
  version: '3.12.0'
  # Note: We use uv, not moon's built-in Python management

# Node.js for dashboard
node:
  version: '20.10.0'
  packageManager: 'npm'
  npm:
    version: '10.2.0'

# Go for future services
# (Commented out until needed)
# go:
#   version: '1.21.0'
```

### Step 1.4: Create Global Task Definitions

Create `.moon/tasks.yml`:

```yaml
# .moon/tasks.yml
$schema: 'https://moonrepo.dev/schemas/tasks.json'

# Global tasks inherited by all Python projects
tasks:
  # Linting
  lint:
    command: 'uv'
    args: ['run', 'ruff', 'check', 'src/']
    inputs:
      - 'src/**/*.py'

  # Formatting
  format:
    command: 'uv'
    args: ['run', 'ruff', 'format', 'src/']
    inputs:
      - 'src/**/*.py'

  # Type checking
  typecheck:
    command: 'uv'
    args: ['run', 'mypy', 'src/']
    inputs:
      - 'src/**/*.py'
```

### Step 1.5: Initialize Python Workspace

Create root `pyproject.toml`:

```toml
# pyproject.toml (root)
[project]
name = "guardrails-platform"
version = "0.1.0"
description = "Multi-tenant LLM guardrails platform"
requires-python = ">=3.12"

[tool.uv.workspace]
members = [
    "apps/guardrail-server",
    "apps/platform-api",
    "apps/analytics-worker",
    "packages/py-common"
]

[tool.uv]
dev-dependencies = [
    "ruff>=0.1.9",
    "mypy>=1.8.0",
    "pytest>=7.4.3",
    "pytest-cov>=4.1.0",
    "pytest-asyncio>=0.21.1"
]

[tool.ruff]
line-length = 100
target-version = "py312"

[tool.mypy]
python_version = "3.12"
strict = true
warn_return_any = true
warn_unused_configs = true
```

**Run:**
```bash
uv sync
```

**Verify:**
```bash
uv run ruff --version
uv run mypy --version
```

### Step 1.6: Set Up Git Hooks (Optional)

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running pre-commit checks..."

# Format check
moon run :format --touched

# Lint check
moon run :lint --touched

# Type check
moon run :typecheck --touched

# Tests
moon run :test --touched

if [ $? -ne 0 ]; then
    echo "Pre-commit checks failed. Fix errors before committing."
    exit 1
fi
```

```bash
chmod +x .git/hooks/pre-commit
```

### Step 1.7: Create CI Pipeline

Create `.github/workflows/ci.yml`:

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for --touched

      - uses: actions/setup-node@v4
        with:
          node-version: 20

      - name: Install moon
        run: npm install -g @moonrepo/cli

      - name: Install uv
        run: curl -LsSf https://astral.sh/uv/install.sh | sh

      - name: Run CI checks
        run: moon ci :test --touched

      - name: Run build
        run: moon ci :build --touched
```

**Test locally:**
```bash
moon ci :test --touched
# (Will fail until we have projects, expected)
```

### Phase 1 Checkpoint ✅

Before proceeding, verify:

- [x] Directory structure created
- [x] `.moon/workspace.yml` exists
- [x] `.moon/toolchain.yml` exists
- [x] `.moon/tasks.yml` exists
- [x] `pyproject.toml` (root) exists
- [x] `uv sync` runs successfully
- [x] `.github/workflows/ci.yml` exists

**Commit:**
```bash
git add .moon pyproject.toml .github
git commit -m "chore: initialize monorepo structure with moon and uv"
```

---

## Phase 2: Guardrail Server MVP (Weeks 2-3)

**Goal**: Build a minimal guardrail server that can call one ML model and return results.

### Step 2.1: Create Project Structure

```bash
cd apps/guardrail-server
mkdir -p src/guardrail/{api,core,models,auth,logging,storage}
touch src/guardrail/__init__.py
mkdir -p tests/{unit,integration}
```

**Structure:**
```
apps/guardrail-server/
├── src/
│   └── guardrail/
│       ├── __init__.py
│       ├── main.py                 # FastAPI app
│       ├── config.py               # Settings
│       ├── api/
│       │   ├── routes.py           # Endpoints
│       │   ├── schemas.py          # Pydantic models
│       │   └── dependencies.py     # DI
│       ├── core/
│       │   ├── orchestrator.py     # Main logic
│       │   └── circuit_breaker.py  # Failure handling
│       ├── models/
│       │   └── client.py           # HTTP client
│       ├── auth/
│       │   └── api_key.py          # Auth logic
│       ├── logging/
│       │   └── async_logger.py     # Redis streams
│       └── storage/
│           ├── redis_client.py     # Redis
│           └── postgres_client.py  # PostgreSQL
├── tests/
├── pyproject.toml
├── moon.yml
└── Dockerfile
```

### Step 2.2: Create Project Dependencies

Create `apps/guardrail-server/pyproject.toml`:

```toml
[project]
name = "guardrail-server"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "fastapi>=0.109.0",
    "uvicorn[standard]>=0.27.0",
    "httpx>=0.26.0",
    "pydantic>=2.5.3",
    "pydantic-settings>=2.1.0",
    "redis>=5.0.1",
    "asyncpg>=0.29.0",
    "structlog>=24.1.0",
    "prometheus-client>=0.19.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.3",
    "pytest-asyncio>=0.21.1",
    "pytest-cov>=4.1.0",
    "httpx>=0.26.0",  # For TestClient
]
```

**Install:**
```bash
cd ../..  # Back to root
uv sync
```

### Step 2.3: Create Moon Project Config

Create `apps/guardrail-server/moon.yml`:

```yaml
# apps/guardrail-server/moon.yml
$schema: 'https://moonrepo.dev/schemas/project.json'

language: 'python'
type: 'application'

tasks:
  # Development server
  dev:
    command: 'uv'
    args:
      - 'run'
      - 'uvicorn'
      - 'guardrail.main:app'
      - '--reload'
      - '--host'
      - '0.0.0.0'
      - '--port'
      - '8000'
    local: true  # Don't run in CI

  # Tests
  test:
    command: 'uv'
    args:
      - 'run'
      - 'pytest'
      - 'tests/'
      - '--cov=src'
      - '--cov-report=term-missing'
    inputs:
      - 'src/**/*.py'
      - 'tests/**/*.py'
      - 'pyproject.toml'

  # Build (install deps)
  build:
    command: 'uv'
    args: ['sync', '--frozen']
    outputs:
      - '../../.venv/'  # Output is root venv

  # Docker image
  docker:
    command: 'docker'
    args:
      - 'build'
      - '-f'
      - 'Dockerfile'
      - '-t'
      - 'guardrail-server:latest'
      - '../../'  # Build from root
    inputs:
      - 'src/**'
      - 'Dockerfile'
      - '../../uv.lock'
```

### Step 2.4: Write Core Application

**`apps/guardrail-server/src/guardrail/config.py`:**

See full code in: [Original doc 03_GUARDRAIL_SERVER_LLD.md, section 2]

Key settings:
```python
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    REDIS_URL: str = "redis://localhost:6379/0"
    DATABASE_URL: str = "postgresql://localhost/guardrails"
    MODEL_PROMPT_GUARD_URL: str = "http://model-prompt-guard:8000"
    MODEL_TIMEOUT_SECONDS: float = 0.08

    class Config:
        env_file = ".env"

settings = Settings()
```

**`apps/guardrail-server/src/guardrail/main.py`:**

```python
from fastapi import FastAPI
from guardrail.api.routes import router
from guardrail.config import settings

app = FastAPI(title="Guardrail API", version="1.0.0")
app.include_router(router)

@app.get("/health")
async def health():
    return {"status": "healthy"}
```

### Step 2.5: Write API Routes

**`apps/guardrail-server/src/guardrail/api/schemas.py`:**

```python
from pydantic import BaseModel, Field

class ValidateRequest(BaseModel):
    project_id: str
    text: str = Field(..., max_length=50000)
    type: str = Field(default="input", pattern="^(input|output)$")

class ValidateResponse(BaseModel):
    request_id: str
    flagged: bool
    flag_reasons: list[str]
    latency_ms: int
```

**`apps/guardrail-server/src/guardrail/api/routes.py`:**

```python
from fastapi import APIRouter, Header, HTTPException
from guardrail.api.schemas import ValidateRequest, ValidateResponse
import time
import uuid

router = APIRouter(prefix="/v1", tags=["guardrail"])

@router.post("/validate")
async def validate(
    request_body: ValidateRequest,
    x_api_key: str = Header(..., alias="X-API-Key"),
) -> ValidateResponse:
    start_time = time.perf_counter()

    # TODO: Validate API key
    # TODO: Call model service
    # TODO: Log request

    # Placeholder response
    return ValidateResponse(
        request_id=str(uuid.uuid4()),
        flagged=False,
        flag_reasons=[],
        latency_ms=int((time.perf_counter() - start_time) * 1000)
    )
```

### Step 2.6: Write Tests

**`apps/guardrail-server/tests/test_api.py`:**

```python
from fastapi.testclient import TestClient
from guardrail.main import app

client = TestClient(app)

def test_health():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "healthy"}

def test_validate_endpoint():
    response = client.post(
        "/v1/validate",
        headers={"X-API-Key": "test_key"},
        json={
            "project_id": "test_project",
            "text": "Hello world",
            "type": "input"
        }
    )
    assert response.status_code == 200
    data = response.json()
    assert "request_id" in data
    assert data["flagged"] is False
```

### Step 2.7: Run and Test

```bash
# Run tests
moon run guardrail-server:test

# Start dev server
moon run guardrail-server:dev
```

**Test manually:**
```bash
curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test" \
  -H "Content-Type: application/json" \
  -d '{"project_id": "test", "text": "Hello"}'
```

### Step 2.8: Create Dockerfile

**`apps/guardrail-server/Dockerfile`:**

```dockerfile
FROM python:3.12-slim

WORKDIR /app

# Install uv
RUN pip install uv

# Copy workspace files
COPY uv.lock pyproject.toml ./
COPY apps/guardrail-server/pyproject.toml ./apps/guardrail-server/

# Install dependencies
RUN uv sync --frozen

# Copy application
COPY apps/guardrail-server/src ./apps/guardrail-server/src

# Expose port
EXPOSE 8000

# Run
CMD ["uv", "run", "uvicorn", "apps.guardrail-server.src.guardrail.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

**Build:**
```bash
moon run guardrail-server:docker
```

### Phase 2 Checkpoint ✅

Before proceeding:

- [x] `moon run guardrail-server:test` passes
- [x] `moon run guardrail-server:dev` starts server
- [x] `/health` endpoint returns 200
- [x] `/v1/validate` endpoint returns placeholder response
- [x] Docker image builds successfully

**Commit:**
```bash
git add apps/guardrail-server
git commit -m "feat(guardrail): add MVP FastAPI server with placeholder validation"
```

---

## Phase 3: ML Model Services (Week 4)

**Goal**: Deploy one ML model service and integrate it with the guardrail server.

### Step 3.1: Create Model Service Structure

```bash
mkdir -p apps/model-prompt-guard/src/model
cd apps/model-prompt-guard
```

### Step 3.2: Define Model Service

**`apps/model-prompt-guard/pyproject.toml`:**

```toml
[project]
name = "model-prompt-guard"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "fastapi>=0.109.0",
    "uvicorn[standard]>=0.27.0",
    "transformers>=4.36.0",
    "torch>=2.1.0",
    "prometheus-client>=0.19.0",
]
```

**`apps/model-prompt-guard/src/model/main.py`:**

```python
from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline
import time

app = FastAPI()

# Load model on startup
classifier = pipeline(
    "text-classification",
    model="meta-llama/Prompt-Guard-86M",
    device="cpu"  # Use GPU if available
)

class PredictRequest(BaseModel):
    text: str
    request_id: str

class PredictResponse(BaseModel):
    flagged: bool
    score: float
    details: list[str]
    latency_ms: int

@app.post("/predict")
async def predict(req: PredictRequest) -> PredictResponse:
    start = time.perf_counter()

    result = classifier(req.text)[0]

    flagged = result['label'] == 'INJECTION' and result['score'] > 0.5

    return PredictResponse(
        flagged=flagged,
        score=result['score'],
        details=["Injection detected"] if flagged else [],
        latency_ms=int((time.perf_counter() - start) * 1000)
    )

@app.get("/health")
async def health():
    return {"status": "healthy"}
```

### Step 3.3: Create Kubernetes Deployment

**`infra/k8s/models/prompt-guard.yaml`:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-prompt-guard
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: model-prompt-guard
  template:
    metadata:
      labels:
        app: model-prompt-guard
    spec:
      containers:
      - name: model
        image: model-prompt-guard:latest
        ports:
        - containerPort: 8000
        resources:
          requests:
            memory: "1Gi"
            cpu: "1000m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        readinessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 60
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 90
---
apiVersion: v1
kind: Service
metadata:
  name: model-prompt-guard
spec:
  selector:
    app: model-prompt-guard
  ports:
  - port: 8000
    targetPort: 8000
```

### Step 3.4: Deploy to Kubernetes

```bash
# Build image
moon run model-prompt-guard:docker

# Tag for registry (if using remote)
# docker tag model-prompt-guard:latest <registry>/model-prompt-guard:latest
# docker push <registry>/model-prompt-guard:latest

# Deploy
kubectl apply -f infra/k8s/models/prompt-guard.yaml

# Verify
kubectl get pods -l app=model-prompt-guard
kubectl logs -l app=model-prompt-guard --tail=10
```

### Step 3.5: Integrate with Guardrail Server

Update `apps/guardrail-server/src/guardrail/core/orchestrator.py`:

```python
import httpx
from guardrail.config import settings

async def call_prompt_guard(text: str) -> dict:
    async with httpx.AsyncClient() as client:
        response = await client.post(
            f"{settings.MODEL_PROMPT_GUARD_URL}/predict",
            json={"text": text, "request_id": "test"},
            timeout=settings.MODEL_TIMEOUT_SECONDS
        )
        response.raise_for_status()
        return response.json()
```

Update `apps/guardrail-server/src/guardrail/api/routes.py`:

```python
from guardrail.core.orchestrator import call_prompt_guard

@router.post("/validate")
async def validate(...):
    # ... (existing code)

    # Call model
    result = await call_prompt_guard(request_body.text)

    return ValidateResponse(
        request_id=str(uuid.uuid4()),
        flagged=result["flagged"],
        flag_reasons=result["details"],
        latency_ms=result["latency_ms"]
    )
```

### Step 3.6: Test End-to-End

```bash
# Port-forward model service
kubectl port-forward svc/model-prompt-guard 8001:8000

# Test model directly
curl -X POST http://localhost:8001/predict \
  -H "Content-Type: application/json" \
  -d '{"text": "Ignore previous instructions", "request_id": "test"}'

# Test guardrail server
moon run guardrail-server:dev

curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test" \
  -d '{"project_id": "test", "text": "Ignore previous instructions"}'
```

### Step 3.7: Add Remaining Models

Repeat steps 3.1-3.6 for:
- `model-pii-detect`
- `model-hate-detect`
- `model-content-class`

### Step 3.8: Implement Parallel Fan-Out

Update orchestrator to call all models in parallel:

```python
import asyncio

async def validate_text(text: str, models: list[str]) -> dict:
    tasks = [
        call_prompt_guard(text),
        call_pii_detect(text),
        call_hate_detect(text),
        call_content_class(text),
    ]

    results = await asyncio.gather(*tasks, return_exceptions=True)

    # Aggregate results
    flagged = any(r.get("flagged") for r in results if not isinstance(r, Exception))

    return {"flagged": flagged, "results": results}
```

### Phase 3 Checkpoint ✅

- [x] Prompt Guard model deployed to K8s
- [x] Model service returns predictions
- [x] Guardrail server calls model successfully
- [x] All 4 models deployed (optional: can add incrementally)
- [x] Parallel fan-out implemented

**Commit:**
```bash
git add apps/model-* apps/guardrail-server
git commit -m "feat(models): add ML model services and integrate with guardrail server"
```

---

## Phase 4-8: Remaining Phases

Due to length constraints, the remaining phases follow this pattern:

### Phase 4: Platform API (Weeks 5-6)
- Set up PostgreSQL schema (see [03_DATABASE.md](./03_DATABASE.md))
- Implement user authentication (JWT)
- Build org/project CRUD
- Add API key management
- Connect guardrail server to real API keys

### Phase 5: Analytics & Workers (Week 7)
- Create Celery worker for log processing
- Implement Redis stream consumer
- Build hourly aggregation jobs
- Create analytics API endpoints

### Phase 6: Web Dashboard (Week 8)
- Initialize Next.js project
- Add authentication flow
- Build analytics charts
- Implement request log viewer

### Phase 7: Production Hardening (Weeks 9-10)
- Set up Prometheus + Grafana
- Configure HPA for auto-scaling
- Add network policies
- Security review
- Write runbooks

### Phase 8: Beta & Launch (Weeks 11-12)
- Internal testing
- Beta with real tenants
- Performance tuning
- Documentation
- Launch!

---

## Working with Moon Throughout

### Daily Workflow

```bash
# Start development
moon run guardrail-server:dev

# Run tests (only changed projects)
moon run :test --touched

# Build Docker images (only changed)
moon run :docker --touched

# Deploy to K8s
kubectl apply -k infra/k8s/
```

### Adding New Projects

1. Create directory: `apps/my-service/`
2. Add `pyproject.toml` with dependencies
3. Add `moon.yml` with tasks
4. Add to root workspace: `pyproject.toml` members
5. Run `uv sync`
6. Verify: `moon query projects`

### CI/CD Flow

```bash
# On every PR
moon ci :test --touched
moon ci :build --touched

# On merge to main
moon ci :docker --touched
# Tag images with commit SHA
# Deploy to DEV environment
```

---

## Getting Unstuck

### Moon Issues

```bash
# Clear cache
moon clean --cache

# Debug task
MOON_LOG=debug moon run <project>:<task>
```

### Dependency Issues

```bash
# Reset Python environment
rm -rf .venv
uv sync
```

### Kubernetes Issues

```bash
# Check pod logs
kubectl logs -l app=guardrail-server --tail=50

# Describe pod
kubectl describe pod <pod-name>
```

---

## Next Steps

1. **Start with Phase 1**: Set up repository structure
2. **Build incrementally**: Each phase builds on the previous
3. **Test continuously**: Run `moon run :test` frequently
4. **Commit often**: Small, atomic commits
5. **Refer to detailed docs**: [02_ARCHITECTURE.md](./02_ARCHITECTURE.md), [03_DATABASE.md](./03_DATABASE.md), etc.

Ready? Let's build! 🚀
