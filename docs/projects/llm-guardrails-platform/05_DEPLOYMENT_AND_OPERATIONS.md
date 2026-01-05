# LLM Guardrails Platform - Deployment & Operations

**Kubernetes deployment guide and operational runbooks.**

---

## Table of Contents

1. [Deployment](#deployment)
2. [Monitoring](#monitoring)
3. [Alerting](#alerting)
4. [Incident Response](#incident-response)
5. [Maintenance](#maintenance)

---

## Deployment

### Prerequisites

Ensure infrastructure is ready:
- ✅ Kubernetes cluster running (see [infra/RUNBOOK.md](../../../infra/RUNBOOK.md))
- ✅ PostgreSQL available
- ✅ Redis available
- ✅ S3 bucket created
- ✅ Docker images built

### Kubernetes Namespace

```bash
kubectl create namespace guardrails-platform
kubectl config set-context --current --namespace=guardrails-platform
```

### Secrets

```bash
# PostgreSQL connection
kubectl create secret generic postgres-credentials \
  --from-literal=url='postgresql://user:pass@postgres:5432/guardrails'

# Redis connection
kubectl create secret generic redis-credentials \
  --from-literal=url='redis://redis:6379/0'

# JWT secret
kubectl create secret generic jwt-secret \
  --from-literal=key="$(openssl rand -base64 32)"

# S3 credentials (if not using IAM roles)
kubectl create secret generic s3-credentials \
  --from-literal=access-key-id='AKIA...' \
  --from-literal=secret-access-key='...'
```

### Deploy Services

**1. Deploy ML Models**:

```bash
# Build images
moon run model-prompt-guard:docker
moon run model-pii-detect:docker
moon run model-hate-detect:docker
moon run model-content-class:docker

# Deploy to K8s
kubectl apply -f infra/k8s/models/

# Verify
kubectl get pods -l type=ml-model
kubectl wait --for=condition=ready pod -l type=ml-model --timeout=300s
```

**2. Deploy Guardrail Server**:

```bash
# Build
moon run guardrail-server:docker

# Deploy
kubectl apply -f infra/k8s/guardrail-server/

# Verify
kubectl get pods -l app=guardrail-server
kubectl logs -l app=guardrail-server --tail=10
```

**3. Deploy Platform API**:

```bash
# Build
moon run platform-api:docker

# Run migrations
kubectl run -it --rm migrate \
  --image=platform-api:latest \
  --restart=Never \
  -- uv run alembic upgrade head

# Deploy
kubectl apply -f infra/k8s/platform-api/

# Verify
kubectl get pods -l app=platform-api
```

**4. Deploy Analytics Worker**:

```bash
# Build
moon run analytics-worker:docker

# Deploy
kubectl apply -f infra/k8s/analytics-worker/

# Verify
kubectl get pods -l app=analytics-worker
kubectl get pods -l app=celery-beat
```

**5. Deploy Web Dashboard**:

```bash
# Build
moon run web-dashboard:docker

# Deploy
kubectl apply -f infra/k8s/web-dashboard/

# Verify
kubectl get pods -l app=web-dashboard
```

**6. Deploy Ingress**:

```bash
# Apply ingress rules
kubectl apply -f infra/k8s/ingress.yaml

# Get external IP
kubectl get ingress guardrails-ingress
```

### Verify Deployment

```bash
# All pods running
kubectl get pods

# Services accessible
kubectl get svc

# Ingress configured
kubectl get ingress

# Test endpoints
curl https://guardrail.example.com/v1/health
curl https://api.example.com/api/v1/health
```

---

## CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ${{ secrets.REGISTRY }}
          username: ${{ secrets.REGISTRY_USERNAME }}
          password: ${{ secrets.REGISTRY_PASSWORD }}

      - name: Build and push images
        run: |
          moon run :docker --touched
          # Tag with commit SHA
          docker tag guardrail-server:latest $REGISTRY/guardrail-server:$GITHUB_SHA
          docker push $REGISTRY/guardrail-server:$GITHUB_SHA

  deploy-dev:
    needs: build
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to DEV
        run: |
          kubectl set image deployment/guardrail-server \
            guardrail=$REGISTRY/guardrail-server:$GITHUB_SHA

  deploy-prod:
    needs: build
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to PROD
        run: |
          # Tag image with version
          docker tag $REGISTRY/guardrail-server:$GITHUB_SHA \
            $REGISTRY/guardrail-server:$GITHUB_REF_NAME
          docker push $REGISTRY/guardrail-server:$GITHUB_REF_NAME

          # Update K8s
          kubectl set image deployment/guardrail-server \
            guardrail=$REGISTRY/guardrail-server:$GITHUB_REF_NAME
```

---

## Monitoring

### Prometheus Metrics

**Guardrail Server**:
```promql
# Request rate
sum(rate(guardrail_request_total[5m])) by (status)

# Latency P99
histogram_quantile(0.99, guardrail_request_latency_seconds)

# In-flight requests
sum(guardrail_in_flight_requests)

# Circuit breaker state
guardrail_circuit_breaker_state{model="prompt-guard"}
```

**ML Models**:
```promql
# Model latency
histogram_quantile(0.99, model_inference_latency_seconds{model="prompt-guard"})

# Model errors
rate(model_inference_total{status="error"}[5m])
```

**PostgreSQL**:
```promql
# Connection count
pg_stat_activity_count

# Replication lag
pg_replication_lag_seconds
```

**Redis**:
```promql
# Memory usage
redis_memory_used_bytes / redis_memory_max_bytes

# Keyspace hits
rate(redis_keyspace_hits_total[5m]) / rate(redis_keyspace_misses_total[5m])
```

### Grafana Dashboards

**Main Dashboard**:
```json
{
  "dashboard": {
    "title": "Guardrails Platform Overview",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [{"expr": "sum(rate(guardrail_request_total[5m]))"}]
      },
      {
        "title": "Latency P99",
        "targets": [{"expr": "histogram_quantile(0.99, guardrail_request_latency_seconds)"}]
      },
      {
        "title": "Circuit Breaker Status",
        "targets": [{"expr": "guardrail_circuit_breaker_state"}]
      }
    ]
  }
}
```

**Access Grafana**:
```bash
# Via Twingate (see infra/RUNBOOK.md)
http://grafana.observability.svc.cluster.local:3000

# Credentials
kubectl get secret grafana-credentials -n observability \
  -o jsonpath='{.data.admin-password}' | base64 -d
```

---

## Alerting

### Alert Rules

**`prometheus/alerts.yml`**:

```yaml
groups:
- name: critical
  rules:
  - alert: HighLatencyP99
    expr: histogram_quantile(0.99, guardrail_request_latency_seconds) > 0.1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "P99 latency exceeded 100ms for 5 minutes"

  - alert: AllModelsDown
    expr: sum(guardrail_circuit_breaker_state) == 4
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "All ML models are down"

  - alert: DatabaseDown
    expr: pg_up == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "PostgreSQL is unreachable"

- name: warning
  rules:
  - alert: SingleModelDown
    expr: guardrail_circuit_breaker_state == 1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "Circuit breaker open for {{ $labels.model_name }}"

  - alert: HighMemoryUsage
    expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.8
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Redis memory usage > 80%"
```

### Alert Channels

```bash
# Configure Alertmanager
kubectl create secret generic alertmanager-config \
  --from-file=alertmanager.yml=config/alertmanager.yml

# Example: Slack webhook
receivers:
- name: 'slack'
  slack_configs:
  - api_url: 'https://hooks.slack.com/services/...'
    channel: '#alerts'
```

---

## Incident Response

### Runbook: High Latency

**Symptoms**:
- Alert: `HighLatencyP99`
- Dashboard shows P99 > 100ms

**Investigation**:

1. **Check model latency**:
```bash
kubectl top pods -l type=ml-model
```

2. **Check circuit breaker state**:
```bash
curl http://guardrail-server:8000/debug/circuit-breakers
```

3. **Check database performance**:
```bash
kubectl exec -it postgres-0 -- psql -c "SELECT * FROM pg_stat_activity;"
```

**Mitigation**:

1. **Scale up models**:
```bash
kubectl scale deployment model-prompt-guard --replicas=6
```

2. **Clear Redis cache** (if stale):
```bash
kubectl exec -it redis-0 -- redis-cli FLUSHDB
```

3. **Force circuit breaker closed** (if stuck):
```bash
curl -X POST http://guardrail-server:8000/debug/circuit-breakers/prompt-guard/close
```

### Runbook: All Models Down

**Symptoms**:
- Alert: `AllModelsDown`
- Validation requests returning 503

**Investigation**:

1. **Check pod status**:
```bash
kubectl get pods -l type=ml-model
```

2. **Check logs**:
```bash
kubectl logs -l type=ml-model --tail=50
```

3. **Check node resources**:
```bash
kubectl top nodes
```

**Mitigation**:

1. **Restart models**:
```bash
kubectl rollout restart deployment model-prompt-guard
kubectl rollout restart deployment model-pii-detect
kubectl rollout restart deployment model-hate-detect
kubectl rollout restart deployment model-content-class
```

2. **Scale up nodes** (if resource constrained):
```bash
# OCI: Increase node pool size
terraform apply -var="node_pool_size=5"
```

### Runbook: Database Down

**Symptoms**:
- Alert: `DatabaseDown`
- Platform API returning 503
- Guardrail server falling back to cache

**Investigation**:

1. **Check PostgreSQL pod**:
```bash
kubectl get pods -l app=postgresql
kubectl logs postgresql-0 --tail=100
```

2. **Check disk space**:
```bash
kubectl exec -it postgresql-0 -- df -h
```

3. **Check replication**:
```bash
kubectl exec -it postgresql-0 -- psql -c "SELECT * FROM pg_stat_replication;"
```

**Mitigation**:

1. **Restart PostgreSQL**:
```bash
kubectl rollout restart statefulset postgresql
```

2. **Promote standby** (if primary dead):
```bash
kubectl exec -it postgresql-1 -- pg_ctlcluster 15 main promote
```

3. **Restore from backup**:
```bash
# Download latest backup
aws s3 cp s3://backups/postgresql/latest.dump /tmp/

# Restore
kubectl exec -i postgresql-0 -- pg_restore -d guardrails < /tmp/latest.dump
```

---

## Maintenance

### Scaling

#### Capacity Planning Formula

To determine the number of model pods needed for your target RPS (Requests Per Second), use this formula:

```
RPS = (1000ms / avg_end_to_end_latency_ms) × model_pods
```

**Example calculations:**

For a model with 70ms average end-to-end latency:

- **14 RPS** → 1 pod per model (baseline)
  - Cost: ~$30/month (4 models × 1 pod × $0.0208/hr)

- **50 RPS** → 4 pods per model
  - Cost: ~$120/month (4 models × 4 pods)

- **100 RPS** → 7 pods per model
  - Cost: ~$210/month (4 models × 7 pods)

- **1000 RPS** → 70 pods per model
  - Cost: ~$2,097/month (4 models × 70 pods)

**Traffic spike buffer:** For 2x traffic spikes, double your pod count. For example, if you need to handle 50 RPS steady-state but want to handle 100 RPS spikes, provision for 100 RPS (7 pods per model).

**Key variables:**
- `avg_end_to_end_latency_ms`: Measured P50 or P90 latency from your SLO
- `model_pods`: Number of replicas for each ML model
- Cost assumes OCI compute at ~$0.0208/hr per pod

Use load testing to validate these numbers for your specific workload.

**Manual scaling**:
```bash
# Scale guardrail server
kubectl scale deployment guardrail-server --replicas=6

# Scale specific model
kubectl scale deployment model-prompt-guard --replicas=4
```

**HPA (automatic)**:
```bash
# Already configured in deployment.yaml
kubectl get hpa

# Adjust targets
kubectl patch hpa guardrail-server-hpa -p '{"spec":{"minReplicas":4}}'
```

### Updating Services

**Zero-downtime deployment**:

```bash
# Build new image
moon run guardrail-server:docker

# Tag with version
docker tag guardrail-server:latest guardrail-server:v1.2.0

# Update deployment
kubectl set image deployment/guardrail-server \
  guardrail=guardrail-server:v1.2.0

# Monitor rollout
kubectl rollout status deployment/guardrail-server

# Rollback if issues
kubectl rollout undo deployment/guardrail-server
```

### Database Migrations

```bash
# Run migration
kubectl run -it --rm migrate \
  --image=platform-api:latest \
  --restart=Never \
  -- uv run alembic upgrade head

# Check version
kubectl run -it --rm alembic-current \
  --image=platform-api:latest \
  --restart=Never \
  -- uv run alembic current
```

### Backup & Restore

**PostgreSQL backup**:
```bash
# Automated daily backup (cronjob)
kubectl create -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -Fc > /tmp/backup.dump
              aws s3 cp /tmp/backup.dump s3://backups/postgresql/$(date +%Y%m%d).dump
EOF
```

**Redis backup**:
```bash
# Trigger RDB save
kubectl exec redis-0 -- redis-cli BGSAVE

# Copy RDB file
kubectl cp redis-0:/data/dump.rdb ./redis-backup.rdb

# Upload to S3
aws s3 cp redis-backup.rdb s3://backups/redis/
```

### Log Rotation

**Request logs cleanup**:
```sql
-- Drop old partitions (automated)
DROP TABLE request_logs_2024_10;

-- Or archive first
COPY (SELECT * FROM request_logs_2024_10) TO PROGRAM 'gzip > /tmp/logs_2024_10.csv.gz';
```

**Application logs** (K8s):
```bash
# Logs automatically rotated by kubelet
# Retention: 7 days (default)

# Archive old logs to S3 (optional)
kubectl logs guardrail-server-abc123 --since=7d > logs.txt
aws s3 cp logs.txt s3://logs/guardrail-server/2025-01-15.txt
```

---

## Performance Tuning

### Database Optimization

```sql
-- Analyze query performance
EXPLAIN ANALYZE
SELECT * FROM request_logs
WHERE project_id = 'proj_abc123' AND created_at > NOW() - INTERVAL '7 days';

-- Add missing index
CREATE INDEX idx_logs_project_recent ON request_logs(project_id, created_at DESC)
WHERE created_at > NOW() - INTERVAL '30 days';

-- Vacuum (maintenance)
VACUUM ANALYZE request_logs;
```

### Redis Tuning

```bash
# Increase max memory
kubectl exec redis-0 -- redis-cli CONFIG SET maxmemory 2gb

# Set eviction policy
kubectl exec redis-0 -- redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

### Model Optimization

```python
# Use quantization for faster inference
from transformers import AutoModelForSequenceClassification, AutoTokenizer

model = AutoModelForSequenceClassification.from_pretrained(
    "meta-llama/Prompt-Guard-86M",
    torch_dtype="float16"  # Half precision
)
```

---

## Security

### TLS Certificates

```bash
# Using cert-manager (automated)
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: guardrails-tls
spec:
  secretName: guardrails-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - guardrail.example.com
  - api.example.com
  - app.example.com
EOF
```

### Rotating Secrets

```bash
# Rotate JWT secret
kubectl create secret generic jwt-secret \
  --from-literal=key="$(openssl rand -base64 32)" \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart services
kubectl rollout restart deployment platform-api
kubectl rollout restart deployment web-dashboard
```

### Network Policies

```bash
# Apply network policies
kubectl apply -f infra/k8s/network-policies/

# Verify
kubectl get networkpolicies
```

---

## Troubleshooting

### Debug Pod

```bash
# Create debug pod
kubectl run -it --rm debug \
  --image=alpine \
  --restart=Never \
  -- sh

# Inside pod
apk add curl
curl http://guardrail-server:8000/health
```

### Common Issues

**Issue**: Pods in `CrashLoopBackOff`

```bash
# Check logs
kubectl logs <pod-name> --previous

# Describe pod
kubectl describe pod <pod-name>
```

**Issue**: Service unavailable

```bash
# Check endpoints
kubectl get endpoints

# Test service internally
kubectl run -it --rm test \
  --image=curlimages/curl \
  --restart=Never \
  -- curl http://guardrail-server:8000/health
```

**Issue**: High memory usage

```bash
# Check pod resources
kubectl top pods

# Check node resources
kubectl top nodes

# Describe pod for OOM
kubectl describe pod <pod-name> | grep -i oom
```

---

## Documentation

- **Architecture**: [02_ARCHITECTURE.md](./02_ARCHITECTURE.md)
- **Database**: [03_DATABASE.md](./03_DATABASE.md)
- **API**: [04_API_REFERENCE.md](./04_API_REFERENCE.md)
- **Infrastructure**: [../../../infra/RUNBOOK.md](../../../infra/RUNBOOK.md)
