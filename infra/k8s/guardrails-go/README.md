# Guardrails Go Platform - Deployment Guide

## Overview

This directory contains Kubernetes manifests for the Go-based LLM Guardrails Platform.
It is a parallel deployment to the Python-based `guardrails-platform` namespace.

## Namespace

```
guardrails-go
```

## Services

| Service | Description | Port |
|---------|-------------|------|
| `guardrail-server` | Main orchestration service (Go) | 8000 |
| `model-prompt-guard` | Prompt injection detection (Go) | 8000 |
| `model-pii-detect` | PII detection (Go) | 8000 |
| `model-hate-detect` | Hate speech detection (Go) | 8000 |
| `model-content-class` | Content classification (Go) | 8000 |

## Deployment Order

```bash
# 1. Create namespace
kubectl apply -f namespace.yaml

# 2. Create registry secret (copy from Python namespace or create new)
kubectl create secret docker-registry oci-registry-secret \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson \
  -n guardrails-go

# 3. Apply configs
kubectl apply -f configs/

# 4. Deploy models first
kubectl apply -f models/

# 5. Deploy guardrail server
kubectl apply -f guardrail-server/

# 6. Deploy ingress
kubectl apply -f ingress.yaml

# 7. (Optional) Deploy autoscaling
kubectl apply -f autoscaling/
```

## Verification

```bash
# Check all pods
kubectl get pods -n guardrails-go

# Test health endpoint
kubectl port-forward svc/guardrail-server 8000:8000 -n guardrails-go &
curl http://localhost:8000/v1/health

# Test validation
curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"test","text":"Hello world","type":"input"}'
```

## Key Differences from Python Version

1. **Lower Memory**: Go binaries use ~64-128MB vs Python's 256MB
2. **Faster Startup**: Go services start in <1s vs Python's 2-3s
3. **Same Port**: Both use port 8000
4. **Same API**: 100% compatible API contract
5. **Same Metrics**: Identical Prometheus metric names

## HTTPProxy

The Go services use a separate host: `api-go.guardrail.com`
This allows parallel testing with the Python version.

## Switching Traffic

To switch all traffic to Go:
1. Update the Python HTTPProxy to point to Go services, OR
2. Update DNS for `api.guardrail.com` to point to the Go HTTPProxy
