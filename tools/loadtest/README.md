# Guardrail Load Test Tool

A Go-based load testing tool designed to simulate multi-tenant traffic against the Guardrails platform validation API. Collects detailed latency metrics and per-tenant performance data.

## Features

- **Multi-tenant simulation**: Simulates multiple tenants with unique API keys and project IDs
- **Configurable load**: Control RPS (requests per second), duration, and worker concurrency
- **Detailed metrics**: P50, P90, P95, P99 latency percentiles, success/error rates
- **Per-tenant breakdown**: Track performance per simulated tenant
- **JSON/Text output**: Machine-readable JSON or human-friendly formatted output
- **Kubernetes-ready**: Runs as a Job with configurable parameters via ConfigMap

## Usage

### Local Development

#### Build

```bash
cd tools/loadtest
go build -o loadtest .
```

#### Run

```bash
# Basic test (local dev server)
./loadtest --target http://localhost:8000 --duration 60s --rps 100

# Production-like test
./loadtest \
  --target http://guardrail.local \
  --duration 300s \
  --rps 500 \
  --workers 20 \
  --tenants 10 \
  --output json
```

#### Command-line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--target` | `http://localhost:8000` | Guardrail API URL |
| `--duration` | `60s` | Test duration (e.g., 60s, 5m) |
| `--rps` | `100` | Target requests per second |
| `--workers` | `10` | Number of concurrent workers |
| `--tenants` | `5` | Number of simulated tenants |
| `--output` | `text` | Output format (json/text) |

### Kubernetes Deployment

The loadtest runs as a Kubernetes Job in the `guardrails-platform` namespace.

#### Step 1: Build and Push Image

```bash
# Build Docker image
docker build -t bom.ocir.io/bm96q5bq36zw/guardrail/loadtest:latest tools/loadtest/

# Push to OCI registry
docker push bom.ocir.io/bm96q5bq36zw/guardrail/loadtest:latest

# Optional: Tag with version
docker tag bom.ocir.io/bm96q5bq36zw/guardrail/loadtest:latest \
  bom.ocir.io/bm96q5bq36zw/guardrail/loadtest:v1.0.0
docker push bom.ocir.io/bm96q5bq36zw/guardrail/loadtest:v1.0.0
```

#### Step 2: Configure Test Parameters

Edit the ConfigMap in [infra/k8s/loadtest/loadtest-job.yaml](../../infra/k8s/loadtest/loadtest-job.yaml):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: loadtest-config
  namespace: guardrails-platform
data:
  TARGET: "http://envoy.projectcontour"
  DURATION: "120s"
  RPS: "100"
  WORKERS: "10"
  TENANTS: "5"
  HOST_HEADER: "guardrail.local"
```

**Configuration Notes:**
- `TARGET`: Internal cluster service (e.g., `http://envoy.projectcontour`) or external URL
- `DURATION`: Test duration in Go duration format (e.g., `30s`, `5m`, `1h`)
- `RPS`: Target requests per second (distribute across workers)
- `WORKERS`: Concurrent goroutines sending requests
- `TENANTS`: Number of unique tenant identities to simulate
- `HOST_HEADER`: HTTP Host header (required when routing through Envoy/Contour)

#### Step 3: Run the Load Test

```bash
# Apply the Job manifest
kubectl apply -f infra/k8s/loadtest/loadtest-job.yaml

# Watch job progress
kubectl get jobs -n guardrails-platform -w

# View results
kubectl logs -n guardrails-platform -l app=loadtest --tail=100
```

#### Step 4: Clean Up

```bash
# Delete the job (keeps logs for ttlSecondsAfterFinished duration)
kubectl delete job guardrail-loadtest -n guardrails-platform
```

## Understanding Results

### Text Output Example

```
╔═══════════════════════════════════════════════════════════════╗
║                    LOAD TEST RESULTS                          ║
╠═══════════════════════════════════════════════════════════════╣
║  Duration        : 120.0s                                     ║
║  Target RPS      : 100                                        ║
║  Achieved RPS    : 98.7                                       ║
║  Total Requests  : 11.8K                                      ║
╠═══════════════════════════════════════════════════════════════╣
║  LATENCY (ms)                                                 ║
║    P50           : 45.2                                       ║
║    P90           : 78.5                                       ║
║    P95           : 92.3                                       ║
║    P99           : 127.4                                      ║
║    Max           : 234.1                                      ║
╠═══════════════════════════════════════════════════════════════╣
║  SUCCESS/ERROR                                                ║
║    Success       : 11.8K   (100.0%)                           ║
║    Timeout       : 0       (0.0%)                             ║
║    Server Error  : 0       (0.0%)                             ║
╚═══════════════════════════════════════════════════════════════╝

Per-Tenant Breakdown:
┌────────────┬───────────┬──────────┬──────────┬──────────┐
│ Tenant     │ Requests  │ Success  │ P50 (ms) │ P99 (ms) │
├────────────┼───────────┼──────────┼──────────┼──────────┤
│ tenant-1   │      2360 │   100.0% │     44.8 │    125.2 │
│ tenant-2   │      2368 │   100.0% │     45.1 │    126.8 │
│ tenant-3   │      2372 │   100.0% │     45.6 │    129.1 │
│ tenant-4   │      2356 │   100.0% │     45.0 │    128.4 │
│ tenant-5   │      2364 │   100.0% │     45.3 │    127.0 │
└────────────┴───────────┴──────────┴──────────┴──────────┘
```

### Key Metrics

- **Achieved RPS**: Actual throughput (should be close to target RPS)
- **P50/P90/P95/P99**: Latency percentiles in milliseconds
  - P50: Median latency (half of requests are faster)
  - P99: 99% of requests complete within this time
- **Success Rate**: Percentage of 2xx responses
- **Timeout**: Requests that exceeded 5s timeout
- **Server Error**: 5xx responses from the API

### Interpreting Results

**Good Performance:**
- Achieved RPS matches target RPS (±5%)
- P99 latency < 200ms for validation API
- Success rate > 99%
- Per-tenant latencies are balanced

**Performance Issues:**
- Achieved RPS << Target RPS → Server is saturated or client-side bottleneck
- P99 latency > 500ms → Investigate slow database queries or model inference
- High timeout rate → Increase server resources or reduce RPS
- Unbalanced tenant latencies → Check for rate limiting or resource contention

## Architecture

The load tester consists of:

1. **Client** ([client.go](./client.go)): HTTP client with connection pooling, tenant simulation, request generation
2. **Runner** ([runner.go](./runner.go)): Concurrent worker pool, rate limiting, request distribution
3. **Metrics** ([metrics.go](./metrics.go)): Latency tracking, percentile calculation, per-tenant aggregation
4. **Main** ([main.go](./main.go)): CLI interface, result formatting

### How It Works

1. Generate N tenant identities (API keys, project IDs)
2. Spawn W worker goroutines
3. Each worker pulls from a rate-limited channel (RPS target)
4. Workers send `/v1/validate` POST requests with random text samples
5. Track latency, success/failure per tenant
6. Aggregate results and calculate percentiles
7. Output formatted summary

## Test Data

The tool uses realistic text samples including:
- Normal user queries
- Long-form technical questions
- Special characters and emojis
- Edge cases (SQL injection attempts, XSS payloads)

See [client.go:198-223](./client.go) for the full sample set.

## Customization

### Adding Custom Text Samples

Edit the `textSamples` array in [client.go](./client.go) to include domain-specific content.

### Adjusting Timeout

Modify the HTTP client timeout in [client.go:72](./client.go):

```go
Timeout: 5 * time.Second, // Increase for slower APIs
```

### Changing Tenant Count

Use `--tenants` flag or update ConfigMap. Each tenant gets a unique API key and project ID.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `ImagePullBackOff` in K8s | Ensure `oci-registry-secret` exists in namespace |
| `Achieved RPS << Target RPS` | Increase `--workers` or check server capacity |
| High timeout rate | Reduce `--rps` or increase server resources |
| `connection refused` | Verify `TARGET` URL is accessible from cluster |
| Job stuck in `Pending` | Check resource requests/limits and node capacity |

## Example Scenarios

### Scenario 1: Baseline Performance Test

```bash
# 2-minute test at moderate load
kubectl set env -n guardrails-platform \
  configmap/loadtest-config \
  DURATION=120s RPS=50 WORKERS=5 TENANTS=3

kubectl apply -f infra/k8s/loadtest/loadtest-job.yaml
```

### Scenario 2: Stress Test

```bash
# 5-minute high-load test
kubectl set env -n guardrails-platform \
  configmap/loadtest-config \
  DURATION=300s RPS=500 WORKERS=50 TENANTS=20

kubectl apply -f infra/k8s/loadtest/loadtest-job.yaml
```

### Scenario 3: Single Tenant Simulation

```bash
# Test rate limiting for single tenant
kubectl set env -n guardrails-platform \
  configmap/loadtest-config \
  DURATION=60s RPS=200 WORKERS=10 TENANTS=1

kubectl apply -f infra/k8s/loadtest/loadtest-job.yaml
```

## Integration with Monitoring

The load test results can be correlated with:
- **Prometheus metrics**: Check `/metrics` endpoint during load test
- **Grafana dashboards**: View real-time latency, throughput, error rates
- **Application logs**: Investigate failed requests in guardrail-server logs

```bash
# Watch Grafana during load test
kubectl port-forward -n observability svc/grafana 3000:3000

# Check Prometheus metrics
kubectl port-forward -n observability svc/prometheus 9090:9090
```
