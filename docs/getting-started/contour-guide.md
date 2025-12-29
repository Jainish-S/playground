# Contour Ingress Controller Architecture

> **Purpose**: This document explains the architecture for using Contour (Envoy-based) as the ingress controller for the Guardrail LLM Platform. Use this as a learning resource for understanding why Contour was chosen and how to configure it from scratch.

---

## Why Contour Over Nginx?

| Feature | Nginx (Traditional) | Contour (This Project) |
|---------|---------------------|------------------------|
| **Config Style** | Annotations (unvalidated text) | HTTPProxy CRDs (typed, validated at apply-time) |
| **Metrics** | Basic (parse logs or pay $$$) | **Deep, native Prometheus** (P50/P95/P99 per route) |
| **Rate Limiting** | Local only (global requires $3k+/instance) | **Global & Local (free)** |
| **Dynamic Updates** | Process reload (can drop connections) | **xDS hot reload (zero downtime)** |
| **gRPC Support** | Legacy HTTP/2 adapter | **Native, optimized** |

### Key Benefits for ML/AI Workloads

1. **Observability**: Envoy exposes `envoy_cluster_upstream_rq_time` histogram per upstream cluster - you can see ML model latency breakdown in Grafana without parsing logs
2. **Rate Limiting**: Global rate limiting prevents abuse across all pods without paying for Nginx Plus
3. **Type Safety**: HTTPProxy CRDs are validated by Kubernetes - typos are caught at `kubectl apply`, not in production

---

## Architecture

```
                    ┌─────────────────────────────────────────────────┐
                    │              External Traffic                    │
                    │         api.guardrail.com:443                   │
                    └─────────────────┬───────────────────────────────┘
                                      │ TLS (cert-manager auto)
                                      ▼
                    ┌─────────────────────────────────────────────────┐
                    │           projectcontour namespace               │
                    │  ┌─────────────┐    ┌──────────────────────┐   │
                    │  │   Contour   │◀──▶│   Envoy Proxy        │   │
                    │  │ (xDS ctrl)  │    │ NodePort 30080/30443 │   │
                    │  └─────────────┘    └──────────┬───────────┘   │
                    │         ▲                      │ gRPC          │
                    │         │              ┌───────▼───────┐       │
                    │  ┌──────┴──────┐       │ Rate Limit    │       │
                    │  │ HTTPProxy   │       │ Service (RLS) │       │
                    │  │ CRDs        │       └───────┬───────┘       │
                    │  └─────────────┘               │ Redis         │
                    └────────────────────────────────┼───────────────┘
                                                     │
                                      ┌──────────────▼──────────────┐
                                      │     default namespace        │
                                      │                              │
 ┌────────────────────────────────────┴──────────────────────────────┴───┐
 │                                                                        │
 │  ┌──────────────────┐     ┌──────────────────────────────────────┐   │
 │  │  guardrail-server│────▶│  ML Models (direct K8s DNS calls)   │   │
 │  │  :8000           │     │  • model-prompt-guard:8000          │   │
 │  │  (FastAPI)       │     │  • model-pii-detect:8000            │   │
 │  └────────┬─────────┘     │  • model-hate-detect:8000           │   │
 │           │               │  • model-content-class:8000         │   │
 │           │               └──────────────────────────────────────┘   │
 │           ▼                                                          │
 │  ┌─────────────┐  ┌─────────────┐                                   │
 │  │ PostgreSQL  │  │   Redis     │                                   │
 │  │ :5432       │  │   :6379     │                                   │
 │  └─────────────┘  └─────────────┘                                   │
 └──────────────────────────────────────────────────────────────────────┘

 ┌──────────────────────────────────────────────────────────────────────┐
 │                    observability namespace                           │
 │  ┌─────────────┐  ┌─────────────┐                                   │
 │  │ Prometheus  │◀─│ Grafana     │                                   │
 │  │ :9090       │  │ :3000       │                                   │
 │  └──────┬──────┘  └─────────────┘                                   │
 │         │ scrapes:                                                   │
 │         │ • envoy:8002/stats/prometheus                             │
 │         │ • contour:8000/metrics                                     │
 │         │ • guardrail-server:8000/metrics                           │
 └─────────┴────────────────────────────────────────────────────────────┘
```

---

## Installation

### Step 1: Install Contour

```bash
# Add Bitnami Helm repo
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install Contour
helm install contour bitnami/contour \
  --namespace projectcontour \
  --create-namespace \
  -f infra/k8s/contour/values.yaml
```

### Step 2: Verify Installation

```bash
# Check pods
kubectl get pods -n projectcontour

# Expected output:
# NAME                       READY   STATUS    RESTARTS   AGE
# contour-xxx-xxx            1/1     Running   0          1m
# contour-envoy-xxx          1/1     Running   0          1m
```

---

## HTTPProxy Configuration

### Main API Gateway

```yaml
# infra/k8s/contour/httpproxy-guardrail.yaml
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: guardrail-api
  namespace: default
  annotations:
    # cert-manager will automatically provision TLS certificate
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  virtualhost:
    fqdn: api.guardrail.com
    tls:
      secretName: guardrail-tls  # auto-created by cert-manager
    # Global rate limiting (synchronized across all Envoy pods)
    rateLimitPolicy:
      global:
        descriptors:
          - entries:
              - genericKey:
                  value: guardrail-api-global
  routes:
    # Main validation API
    - conditions:
        - prefix: /v1/
      services:
        - name: guardrail-server
          port: 8000
      timeoutPolicy:
        response: 2s        # Hard max response time
        idle: 30s           # Connection idle timeout
      retryPolicy:
        count: 2
        perTryTimeout: 500ms
        retryOn: 5xx,reset,connect-failure
      healthCheckPolicy:
        path: /v1/health
        intervalSeconds: 5
        timeoutSeconds: 2
        unhealthyThresholdCount: 3
        healthyThresholdCount: 2
      loadBalancerPolicy:
        strategy: WeightedLeastRequest  # Routes to least-loaded pod
      # Local rate limit as backup (per Envoy pod)
      rateLimitPolicy:
        local:
          requests: 100
          unit: second

    # Metrics (internal only via Twingate)
    - conditions:
        - prefix: /metrics
      services:
        - name: guardrail-server
          port: 8000
```

### Rate Limiting Strategy

We use a **two-tier approach**:

1. **Global Rate Limiting (Primary)**: 1000 requests/minute across ALL pods
   - Uses external Rate Limit Service (RLS) with Redis backend
   - Prevents the "local limit sync problem" (where pod A has capacity but pod B rejects)

2. **Local Rate Limiting (Backup)**: 100 requests/second per Envoy pod
   - Fast, no external dependencies
   - Kicks in if RLS is slow/unavailable

```yaml
# Rate Limit Service Config
# infra/k8s/contour/ratelimit/config.yaml
domain: contour
descriptors:
  - key: generic_key
    value: guardrail-api-global
    rate_limit:
      unit: minute
      requests_per_unit: 1000
```

---

## ML Model Routing Decision

### Why Direct K8s DNS (Not Contour)?

For internal ML model calls from guardrail-server:

```python
# Current approach (direct DNS)
MODEL_PROMPT_GUARD_URL = "http://model-prompt-guard:8000"
```

**We chose NOT to route ML models through Contour because:**

1. **Sub-millisecond latency requirement**: Adding Contour hop adds ~0.5-1ms
2. **No external exposure needed**: Models are internal-only
3. **Simple scaling**: K8s HPA handles model pod scaling

**Future consideration**: When you have multiple model versions (v1, v2, canary), Contour HTTPProxy weight-based routing becomes valuable:

```yaml
# Future: Model versioning via Contour
routes:
  - conditions:
      - prefix: /predict
    services:
      - name: model-prompt-guard-v1
        port: 8000
        weight: 90  # 90% traffic
      - name: model-prompt-guard-v2
        port: 8000
        weight: 10  # 10% canary
```

---

## Observability

### Prometheus Scrape Config

Add to your Prometheus config:

```yaml
# Contour Controller
- job_name: 'contour'
  kubernetes_sd_configs:
    - role: pod
      namespaces:
        names: [projectcontour]
  relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
      action: keep
      regex: contour
    - source_labels: [__meta_kubernetes_pod_container_port_number]
      action: keep
      regex: "8000"

# Envoy Proxy (the important one!)
- job_name: 'envoy'
  kubernetes_sd_configs:
    - role: pod
      namespaces:
        names: [projectcontour]
  relabel_configs:
    - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
      action: keep
      regex: envoy
    - source_labels: [__meta_kubernetes_pod_container_port_number]
      action: keep
      regex: "8002"
  metrics_path: /stats/prometheus
```

### Key Metrics

| Metric | Description | Use Case |
|--------|-------------|----------|
| `envoy_cluster_upstream_rq_time` | Histogram of request latency per cluster | P99 latency dashboards |
| `envoy_cluster_upstream_rq_xx` | Request count by status code | Error rate alerts |
| `envoy_cluster_circuit_breakers_*` | Circuit breaker state | Detect upstream failures |
| `envoy_http_local_rate_limit_rate_limited` | Rate limit rejections | Capacity planning |

### Example PromQL Queries

```promql
# P99 latency for guardrail-server
histogram_quantile(0.99, 
  sum(rate(envoy_cluster_upstream_rq_time_bucket{
    envoy_cluster_name=~"default/guardrail.*"
  }[5m])) by (le)
)

# Request rate per service
sum(rate(envoy_cluster_upstream_rq_total[5m])) by (envoy_cluster_name)

# Error rate (5xx)
sum(rate(envoy_cluster_upstream_rq_xx{
  envoy_response_code_class="5xx"
}[5m])) / sum(rate(envoy_cluster_upstream_rq_total[5m]))
```

---

## TLS with cert-manager

### How It Works

1. cert-manager watches for HTTPProxy with annotation `cert-manager.io/cluster-issuer`
2. Automatically creates Certificate resource
3. Performs HTTP-01 challenge via Contour
4. Stores certificate in Kubernetes Secret
5. Contour loads certificate for TLS termination

### Configuration

```yaml
# Already configured in infra/k8s/cert-manager/cluster-issuer.yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod-key
    solvers:
      - http01:
          ingress:
            class: contour  # Uses Contour for challenge
```

---

## Migration from Nginx

### Step 1: Deploy Contour Side-by-Side

```bash
# Use alternate ports to avoid conflict with nginx
helm install contour bitnami/contour \
  --namespace projectcontour \
  --create-namespace \
  --set envoy.service.type=NodePort \
  --set envoy.service.nodePorts.http=30081 \
  --set envoy.service.nodePorts.https=30444
```

### Step 2: Test Contour

```bash
curl -H "Host: api.guardrail.com" http://<node-ip>:30081/v1/health
```

### Step 3: Switch Traffic

Update Twingate/DNS to point to Contour ports, then cleanup nginx:

```bash
helm uninstall ingress-nginx -n ingress-nginx
```

---

## File Structure

```
infra/k8s/contour/
├── kustomization.yaml
├── values.yaml                    # Helm values
├── httpproxy-guardrail.yaml       # Main API routing
├── ratelimit/
│   ├── deployment.yaml            # RLS pods
│   ├── service.yaml
│   └── config.yaml                # Rate limit rules
└── README.md                      # Quick reference
```

---

## Troubleshooting

### HTTPProxy Not Working

```bash
# Check status - Contour validates and reports errors
kubectl get httpproxy -A
kubectl describe httpproxy guardrail-api

# Look for "Valid" status, not "Invalid"
```

### Rate Limiting Not Working

```bash
# Check RLS is running
kubectl get pods -n projectcontour -l app=ratelimit

# Check RLS logs
kubectl logs -n projectcontour -l app=ratelimit
```

### TLS Certificate Not Issued

```bash
# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Check certificate status
kubectl get certificate -A
kubectl describe certificate guardrail-tls
```
