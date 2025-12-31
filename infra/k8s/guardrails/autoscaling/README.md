# HPA Deployment Guide

This directory contains Horizontal Pod Autoscaler (HPA) configurations for the guardrails platform.

## Current Status

✅ **Deployed and Running** (December 30, 2025)

All HPA resources are active with both primary and secondary metrics working:

```
NAME                      TARGETS              STATUS
guardrail-server-hpa      0/5, cpu: 3%/70%    ✅ Active
model-prompt-guard-hpa    0/2, cpu: 4%/60%    ✅ Active
model-pii-detect-hpa      0/2, cpu: 4%/60%    ✅ Active
model-hate-detect-hpa     0/2, cpu: 4%/60%    ✅ Active
model-content-class-hpa   0/2, cpu: 4%/60%    ✅ Active
```

## Overview

- **Strategy**: Scale based on in-flight requests (primary) and CPU (secondary)
- **Capacity**: 10 pods min → 42 pods max (84% of cluster quota)
- **Expected throughput**: 70-1400+ RPS sustained with P99 < 100ms
- **Primary metric**: In-flight requests (leading indicator)
- **Secondary metric**: CPU utilization (safety net)

## Files

- `guardrail-server-hpa.yaml` - Guardrail Server HPA (2-10 replicas)
- `model-prompt-guard-hpa.yaml` - Prompt Guard Model HPA (2-8 replicas)
- `model-pii-detect-hpa.yaml` - PII Detect Model HPA (2-8 replicas)
- `model-hate-detect-hpa.yaml` - Hate Detect Model HPA (2-8 replicas)
- `model-content-class-hpa.yaml` - Content Class Model HPA (2-8 replicas)

## Prerequisites

1. Kubernetes cluster with metrics-server installed
2. Prometheus running in `observability` namespace
3. Services deployed in `guardrails-platform` namespace
4. Helm 3.x installed

## Components Deployed

### Infrastructure
1. **Prometheus Adapter** - Exposes custom metrics to HPA (installed in `observability` namespace)
2. **Metrics Server** - Provides CPU/memory metrics (in `kube-system` namespace)
3. **Prometheus Config** - Updated with 10s scraping for guardrails (faster HPA reaction)
4. **HPA Alerts** - 3 monitoring alerts for HPA health

### HPA Resources
- `guardrail-server-hpa.yaml` - 2-10 replicas, scales on 5 in-flight requests or 70% CPU
- `model-prompt-guard-hpa.yaml` - 2-8 replicas, scales on 2 in-flight requests or 60% CPU
- `model-pii-detect-hpa.yaml` - 2-8 replicas, scales on 2 in-flight requests or 60% CPU
- `model-hate-detect-hpa.yaml` - 2-8 replicas, scales on 2 in-flight requests or 60% CPU
- `model-content-class-hpa.yaml` - 2-8 replicas, scales on 2 in-flight requests or 60% CPU

## Deployment Steps (Already Completed)

### Step 1: Install Prometheus Adapter

```bash
# Add Helm repository
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus Adapter
helm install prometheus-adapter prometheus-community/prometheus-adapter \
  --namespace observability \
  --create-namespace \
  -f ../observability/prometheus-adapter/values.yaml

# Wait for deployment
kubectl wait --for=condition=available deployment/prometheus-adapter \
  -n observability --timeout=120s
```

### Step 2: Verify Custom Metrics

```bash
# Check API service registration
kubectl get apiservices | grep custom.metrics

# Test guardrail metric
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/guardrails-platform/pods/*/guardrail_in_flight_requests" | jq .

# Test model metric
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/guardrails-platform/pods/*/model_in_flight_requests" | jq .
```

### Step 3: Update Prometheus Config

```bash
# Apply updated Prometheus configuration (10s scrape interval)
kubectl apply -f ../observability/prometheus/config.yaml

# Restart Prometheus to reload config
kubectl rollout restart deployment/prometheus -n observability
kubectl wait --for=condition=available deployment/prometheus \
  -n observability --timeout=120s
```

### Step 4: Update Prometheus Alerts

```bash
# Apply HPA monitoring alerts
kubectl apply -f ../observability/prometheus/alerts.yaml
```

### Step 5: Deploy HPAs

```bash
# Deploy all HPA resources
kubectl apply -f .

# Verify HPA status
kubectl get hpa -n guardrails-platform -o wide

# Check HPA events
kubectl get events -n guardrails-platform | grep HorizontalPodAutoscaler
```

### Step 6: Fix Metrics-Server (Completed)

The following RBAC fixes were applied to enable CPU metrics:

```bash
# Fix ClusterRoleBinding namespace
kubectl patch clusterrolebinding system:metrics-server --type=json \
  -p='[{"op": "replace", "path": "/subjects/0/namespace", "value": "kube-system"}]'

# Fix APIService namespace
kubectl patch apiservice v1beta1.metrics.k8s.io --type=json \
  -p='[{"op": "replace", "path": "/spec/service/namespace", "value": "kube-system"}]'

# Add auth-delegator permission
kubectl create clusterrolebinding metrics-server-auth-reader \
  --clusterrole=system:auth-delegator \
  --serviceaccount=kube-system:metrics-server

# Restart metrics-server
kubectl rollout restart deployment/metrics-server -n kube-system
```

## Validation

### Check HPA Status

```bash
# Watch HPA metrics
watch -n 5 'kubectl get hpa -n guardrails-platform'

# Describe specific HPA
kubectl describe hpa guardrail-server-hpa -n guardrails-platform
```

### Expected Output

```
NAME                      REFERENCE                     TARGETS                MINPODS   MAXPODS   REPLICAS
guardrail-server-hpa      Deployment/guardrail-server   0/5, cpu: 3%/70%      2         10        2
model-prompt-guard-hpa    Deployment/model-prompt-guard 0/2, cpu: 4%/60%      2         8         2
model-pii-detect-hpa      Deployment/model-pii-detect   0/2, cpu: 4%/60%      2         8         2
model-hate-detect-hpa     Deployment/model-hate-detect  0/2, cpu: 4%/60%      2         8         2
model-content-class-hpa   Deployment/model-content-class 0/2, cpu: 4%/60%     2         8         2
```

**Note**: Both metrics (in-flight and CPU) should be visible. If CPU shows `<unknown>`, see troubleshooting section.

## Troubleshooting

### Issue: "unable to get metric"

**Cause**: Prometheus Adapter not ready or custom metrics not configured

**Fix**:
```bash
# Check Prometheus Adapter logs
kubectl logs -l app.kubernetes.io/name=prometheus-adapter -n observability --tail=50

# Verify metrics exist in Prometheus
kubectl port-forward -n observability svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/query?query=guardrail_in_flight_requests"
```

### Issue: "missing request for cpu"

**Cause**: Resource requests not set in deployment

**Fix**: This should not occur as all deployments have CPU requests defined.

### Issue: HPA shows "unknown" for CPU metrics

**Cause**: Metrics-server RBAC issues (namespace mismatch or missing permissions)

**Fix**:
```bash
# Verify metrics-server is working
kubectl top nodes
kubectl top pods -n guardrails-platform

# If not working, check metrics-server logs
kubectl logs -n kube-system deployment/metrics-server --tail=20

# Apply RBAC fixes (see Step 6 above)
```

### Issue: HPA shows "unknown" for custom metrics

**Cause**: Prometheus Adapter not configured or metrics not available

**Fix**:
```bash
# Check if custom metrics are available
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1" | jq -r '.resources[] | select(.name | contains("guardrail"))| .name'

# Should show: pods/guardrail_in_flight_requests

# If not, check Prometheus Adapter logs
kubectl logs -l app.kubernetes.io/name=prometheus-adapter -n observability --tail=50

# Verify metrics exist in Prometheus
POD_NAME=$(kubectl get pods -n observability -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n observability $POD_NAME -- wget -qO- "http://localhost:9090/api/v1/query?query=guardrail_in_flight_requests" | jq .
```

## Load Testing

Run load tests to validate scaling:

```bash
# Baseline test (30 RPS)
kubectl apply -f ../loadtest/loadtest-job.yaml

# Monitor scaling
watch -n 5 'kubectl get hpa -n guardrails-platform'
```

## Rollback

If issues occur, rollback to static replicas:

```bash
# Delete all HPAs
kubectl delete hpa --all -n guardrails-platform

# Set static replica counts
kubectl scale deployment guardrail-server --replicas=4 -n guardrails-platform
kubectl scale deployment model-prompt-guard --replicas=4 -n guardrails-platform
kubectl scale deployment model-pii-detect --replicas=4 -n guardrails-platform
kubectl scale deployment model-hate-detect --replicas=4 -n guardrails-platform
kubectl scale deployment model-content-class --replicas=4 -n guardrails-platform
```

## Monitoring

Key metrics to watch in Grafana:

- `kube_horizontalpodautoscaler_status_current_replicas` - Current pod count
- `kube_horizontalpodautoscaler_status_desired_replicas` - Desired pod count
- `guardrail_in_flight_requests` - Primary scaling metric
- `guardrail_request_latency_seconds` - Latency SLA compliance

## Alerts

HPA-related alerts configured in `../observability/prometheus/alerts.yaml`:

- **HPAConstrainedByQuota** (critical) - HPA cannot scale due to resource quota limits
- **LatencySLABreachAtMaxScale** (critical) - P99 latency > 100ms despite 8+ guardrail pods
- **HPANotScaling** (warning) - HPA disabled or misconfigured

## Scaling Behavior

### Scale-Up
- **Trigger**: When in-flight requests or CPU exceed target
- **Window**: 30s (guardrail), 20s (models) - filters noise while enabling fast response
- **Policy**: Double pods (100%) OR add 2 pods, whichever is more aggressive
- **Example**: At 200 RPS, guardrail scales 2→4 pods in ~40s

### Scale-Down
- **Trigger**: When metrics drop below target
- **Window**: 300s (guardrail), 180s (models) - prevents flapping during bursty traffic
- **Policy**: Remove 1 pod at a time every 60s
- **Example**: After traffic stops, takes 5min before starting to scale down

### Traffic Capacity

| Traffic (RPS) | Guardrail Pods | Model Pods (each) | Total Pods |
|--------------|----------------|-------------------|------------|
| 0-70         | 2 (min)        | 2 (min)           | 10         |
| 200          | 3-4            | 4-5               | ~23        |
| 500          | 6-7            | 7-8               | ~38        |
| 1000 (peak)  | 10 (max)       | 8 (max)           | 42         |

## Files Modified/Created

### Created
1. `guardrail-server-hpa.yaml` - Guardrail HPA manifest
2. `model-prompt-guard-hpa.yaml` - Prompt Guard HPA manifest
3. `model-pii-detect-hpa.yaml` - PII Detect HPA manifest
4. `model-hate-detect-hpa.yaml` - Hate Detect HPA manifest
5. `model-content-class-hpa.yaml` - Content Class HPA manifest
6. `../observability/prometheus-adapter/values.yaml` - Prometheus Adapter configuration

### Modified
1. `../observability/prometheus/config.yaml` (lines 86-121) - Added 10s scraping for guardrails
2. `../observability/prometheus/alerts.yaml` (lines 72-111) - Added HPA monitoring alerts

## Next Steps

1. **Monitor HPA behavior** for 24-48 hours to observe real scaling patterns
2. **Run load tests** to validate scaling under different traffic levels
3. **Tune thresholds** if needed based on actual traffic patterns
4. **Review Grafana dashboards** for HPA replica counts and in-flight metrics
