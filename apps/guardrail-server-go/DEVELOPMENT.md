# Guardrail Server (Go) - Development Guide

## Overview

The Guardrail Server is an HTTP server that orchestrates parallel calls to ML model services for LLM safety validation. This is the Go implementation, matching the functionality of the Python/FastAPI version.

## Quick Start

```bash
# Run locally
moon run guardrail-server-go:dev

# Or directly
go run ./cmd/server
```

## Development Commands

| Command | Description |
|---------|-------------|
| `moon run guardrail-server-go:dev` | Start development server |
| `moon run guardrail-server-go:build` | Build binary to `bin/server` |
| `moon run guardrail-server-go:test` | Run tests with coverage |
| `moon run guardrail-server-go:lint` | Run golangci-lint |
| `moon run guardrail-server-go:format` | Format code with gofmt |
| `moon run guardrail-server-go:docker` | Build Docker image |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8000` | Server port |
| `DEBUG` | `false` | Enable debug mode |
| `MODEL_PROMPT_GUARD_URL` | `http://model-prompt-guard:8000` | Prompt Guard service URL |
| `MODEL_PII_DETECT_URL` | `http://model-pii-detect:8000` | PII Detect service URL |
| `MODEL_HATE_DETECT_URL` | `http://model-hate-detect:8000` | Hate Detect service URL |
| `MODEL_CONTENT_CLASS_URL` | `http://model-content-class:8000` | Content Class service URL |
| `MODEL_TIMEOUT_SECONDS` | `0.08` | Model call timeout (80ms) |
| `MODEL_CONNECT_TIMEOUT` | `0.02` | Connection timeout (20ms) |
| `CB_FAILURE_THRESHOLD` | `5` | Failures before circuit opens |
| `CB_RECOVERY_TIMEOUT` | `30` | Seconds before half-open |
| `CB_SUCCESS_THRESHOLD` | `3` | Successes to close circuit |
| `RETRY_ENABLED` | `true` | Enable retry logic |
| `RETRY_MAX_ATTEMPTS` | `2` | Maximum retry attempts |
| `RETRY_WAIT_MS` | `5` | Wait between retries (ms) |

## API Endpoints

### Main API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/validate` | Validate text against guardrail models |
| `GET` | `/v1/health` | Liveness probe |
| `GET` | `/v1/ready` | Readiness probe |
| `GET` | `/metrics` | Prometheus metrics |

### Debug API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/debug/circuit-breakers` | Get circuit breaker states |
| `POST` | `/debug/circuit-breakers/{model}/close` | Force close circuit |
| `POST` | `/debug/circuit-breakers/{model}/open` | Force open circuit |

## Testing Locally

```bash
# Start the server
moon run guardrail-server-go:dev &

# Test health
curl http://localhost:8000/v1/health

# Test validation (requires model services running)
curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"test","text":"Hello world","type":"input"}'
```

## Architecture

```
internal/
├── api/          # HTTP routes and handlers
├── circuitbreaker/  # Circuit breaker implementation
├── client/       # HTTP client pool for model calls
├── config/       # Configuration loading
└── orchestrator/ # Parallel model call orchestration
```

## Key Features

- **Parallel Model Calls**: Uses goroutines + WaitGroup for concurrent calls
- **Circuit Breaker**: CLOSED → OPEN → HALF_OPEN state machine
- **Retry Logic**: Configurable retries with backoff
- **Prometheus Metrics**: Request latency, in-flight, circuit state
- **Graceful Shutdown**: Drains in-flight requests before stopping
