# Guardrails Development Guide

Complete development and deployment documentation for the guardrails platform.

## Prerequisites

- Python 3.12+
- Docker Desktop
- kubectl (with K8s cluster access)
- uv (Python package manager)

## Local Development

### Install Dependencies
```bash
cd /Users/jainish/os/playground
uv sync
```

### Run Individual Service
```bash
# Model service
cd apps/model-prompt-guard
uv run uvicorn model.main:app --reload --port 8001

# Guardrail server
cd apps/guardrail-server
uv run uvicorn guardrail.main:app --reload --port 8000
```

### Use Docker Compose (Full Stack)
```bash
cd apps/guardrail-server
docker-compose up --build
```

## Linting & Type Checking

```bash
# From project root
uv run ruff check .       # Lint
uv run ruff format .      # Format
uv run mypy apps/         # Type check
```

## Testing

```bash
uv run pytest                        # All tests
uv run pytest apps/guardrail-server  # Specific app
```

---

## Kubernetes Deployment

### Build & Push Images

```bash
# First time: Login to OCI Container Registry
docker login bom.ocir.io
# Username: <namespace>/oracleidentitycloudservice/<email>
# Password: <auth-token from OCI Console>

# Build and push all images (tags: latest + git SHA)
./scripts/oci-push-images.sh

# With specific version tag
./scripts/oci-push-images.sh v1.0.0
```

See [RUNBOOK.md](../../infra/RUNBOOK.md#phase-6-container-registry-oci) for full registry setup.

### Deploy to Cluster
```bash
./scripts/deploy.sh
# Or step by step:
kubectl apply -f infra/k8s/guardrails/namespace.yaml
kubectl apply -f infra/k8s/guardrails/configs/
kubectl apply -f infra/k8s/guardrails/databases.yaml
kubectl apply -f infra/k8s/guardrails/models/
kubectl apply -f infra/k8s/guardrails/guardrail-server/
kubectl apply -f infra/k8s/guardrails/ingress.yaml
```

### Verify Deployment
```bash
kubectl get pods -n guardrails-platform
kubectl port-forward svc/guardrail-server 8000:8000 -n guardrails-platform
curl http://localhost:8000/v1/health
```

### Test Validation API
```bash
curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test_key" \
  -H "Content-Type: application/json" \
  -d '{"project_id": "test", "text": "Hello world", "type": "input"}'
```

---

## Architecture

| Component | Port | Description |
|-----------|------|-------------|
| guardrail-server | 8000 | FastAPI orchestrator |
| model-prompt-guard | 8000 | Prompt injection detection |
| model-pii-detect | 8000 | PII detection |
| model-hate-detect | 8000 | Hate speech detection |
| model-content-class | 8000 | Content classification |
| postgres | 5432 | Configuration storage |
| redis | 6379 | Caching & rate limiting |

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| MODEL_TIMEOUT_SECONDS | 0.08 | Per-model timeout |
| CB_FAILURE_THRESHOLD | 5 | Failures to open circuit |
| CB_RECOVERY_TIMEOUT | 30 | Seconds before recovery |
| REDIS_URL | redis://redis:6379/0 | Redis connection |
| DATABASE_URL | postgres://... | PostgreSQL connection |

---

## Monitoring

- Grafana Dashboard: `Guardrails Platform`
- Metrics: `/metrics` endpoint
- Debug: `/debug/circuit-breakers`
