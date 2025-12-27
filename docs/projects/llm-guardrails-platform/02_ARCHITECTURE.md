# LLM Guardrails Platform - Architecture

**Detailed system design and component interactions.**

---

## System Context

```
┌──────────────────────────────────────────────────────────────────┐
│                      EXTERNAL ACTORS                              │
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │   Client     │    │   Platform   │    │    Admin     │       │
│  │ Applications │    │    Users     │    │  (Internal)  │       │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘       │
└─────────┼───────────────────┼───────────────────┼───────────────┘
          │                   │                   │
          ▼                   ▼                   ▼
┌──────────────────────────────────────────────────────────────────┐
│                      KUBERNETES CLUSTER                           │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │           Nginx Ingress (TLS, Rate Limiting)               │  │
│  └────────────────────────────────────────────────────────────┘  │
│         │                   │                   │                │
│  ┌──────┴────────┐   ┌──────┴────────┐   ┌─────┴──────┐        │
│  │  Guardrail    │   │  Platform API │   │ Dashboard  │        │
│  │   Server      │   │   (FastAPI)   │   │ (Next.js)  │        │
│  │  (FastAPI)    │   └───────────────┘   └────────────┘        │
│  └───────┬───────┘                                               │
│          │                                                       │
│  ┌───────┴───────────────────────────────────────┐              │
│  │          ML Model Services (4 models)         │              │
│  └───────────────────────────────────────────────┘              │
│                                                                   │
│  ┌───────────────────────────────────────────────┐              │
│  │  PostgreSQL    Redis    S3 (Object Storage)  │              │
│  └───────────────────────────────────────────────┘              │
└──────────────────────────────────────────────────────────────────┘
```

---

## Service Architecture

### 1. Guardrail Server

**Purpose**: Real-time ML inference orchestration

**Responsibilities**:
- Validate API keys
- Enforce rate limits
- Call ML models in parallel
- Aggregate results
- Handle failures gracefully (circuit breaker)
- Log requests asynchronously

**Tech Stack**:
- FastAPI (async HTTP)
- httpx (model service calls)
- Redis (caching, rate limiting)
- PostgreSQL (fallback for cache misses)

**Scaling**: 2-8 replicas (HPA based on in-flight requests)

**Key Files**:
```
apps/guardrail-server/
├── src/guardrail/
│   ├── main.py                  # FastAPI app lifecycle
│   ├── api/routes.py            # /v1/validate endpoint
│   ├── core/orchestrator.py     # Parallel model calls
│   ├── core/circuit_breaker.py  # Failure handling
│   ├── auth/api_key.py          # Key validation
│   └── storage/redis_client.py  # Caching layer
```

### 2. Platform API

**Purpose**: Tenant, user, and project management

**Responsibilities**:
- User authentication (JWT)
- Organization/project CRUD
- API key generation/revocation
- RBAC enforcement
- Analytics queries
- Transaction tagging

**Tech Stack**:
- FastAPI
- SQLAlchemy (async)
- PostgreSQL
- Redis (session store)

**Scaling**: 2-4 replicas

**Key Files**:
```
apps/platform-api/
├── src/platform/
│   ├── api/v1/
│   │   ├── orgs.py              # Organization endpoints
│   │   ├── projects.py          # Project endpoints
│   │   ├── users.py             # User management
│   │   └── api_keys.py          # Key management
│   ├── core/
│   │   ├── security.py          # JWT, password hashing
│   │   └── permissions.py       # RBAC logic
│   └── models/                  # SQLAlchemy models
```

### 3. Analytics Worker

**Purpose**: Background data processing

**Responsibilities**:
- Consume logs from Redis Streams
- Write to PostgreSQL (batch inserts)
- Archive to S3
- Hourly aggregation jobs
- Daily rollups

**Tech Stack**:
- Celery
- Redis (broker, streams)
- PostgreSQL
- boto3 (S3)

**Scaling**: 1-2 replicas (Celery workers)

**Key Files**:
```
apps/analytics-worker/
├── src/worker/
│   ├── tasks/
│   │   ├── log_processor.py     # Redis → PostgreSQL
│   │   ├── hourly_agg.py        # Aggregation job
│   │   └── s3_archiver.py       # Cold storage
│   └── celeryconfig.py          # Celery config
```

### 4. Web Dashboard

**Purpose**: User-facing management UI

**Responsibilities**:
- User authentication
- Organization/project UI
- Analytics visualizations
- Request log viewer
- Team management

**Tech Stack**:
- Next.js (App Router)
- React Server Components
- TailwindCSS
- Recharts (analytics)

**Scaling**: 2-4 replicas

**Key Files**:
```
apps/web-dashboard/
├── app/
│   ├── dashboard/
│   │   ├── analytics/page.tsx   # Charts
│   │   ├── projects/page.tsx    # Project list
│   │   └── logs/page.tsx        # Request viewer
│   └── api/                     # BFF endpoints
```

### 5. ML Model Services

**Purpose**: Run ML models for guardrail checks

Four independent services:
1. **Prompt Guard** - Detect prompt injection (Meta Llama Guard)
2. **PII Detector** - Find sensitive data (Presidio NER)
3. **Hate Detector** - Flag toxic content (BERT fine-tuned)
4. **Content Classifier** - Policy categorization (zero-shot)

**Interface** (all models):
```python
POST /predict
{
  "text": "...",
  "request_id": "..."
}

→ {
  "flagged": true/false,
  "score": 0.0-1.0,
  "details": ["reason1", ...],
  "latency_ms": 45
}
```

**Scaling**: 2-6 replicas each (HPA based on request latency)

---

## Request Flows

### Guardrail Validation Flow

```
Client App
  │
  │ POST /v1/validate
  │ X-API-Key: sk_live_xxx
  │ {"project_id": "...", "text": "..."}
  ▼
Nginx Ingress
  │ • TLS termination
  │ • Global rate limit (100 req/s)
  ▼
Guardrail Server
  │
  ├─[1]─► Redis: Get API key metadata (cache hit)
  │        └─ fallback: PostgreSQL query
  │
  ├─[2]─► Redis: INCR rate limit counter
  │        └─ if exceeded: return 429
  │
  ├─[3]─► Redis: Get project config
  │        └─ enabled models, thresholds, strategy
  │
  ├─[4]─► Model Services (parallel, asyncio.gather)
  │        ├─ POST prompt-guard/predict
  │        ├─ POST pii-detect/predict
  │        ├─ POST hate-detect/predict
  │        └─ POST content-class/predict
  │        (timeout: 80ms, circuit breaker protection)
  │
  ├─[5]─► Aggregate results
  │        └─ any_flag / all_flag / threshold strategy
  │
  └─[6]─► Redis: XADD guardrail_logs (async, fire-and-forget)
          └─ Analytics Worker consumes later

  ▼
Response to Client
  {
    "flagged": true,
    "flag_reasons": ["prompt_injection"],
    "model_results": {...},
    "latency_ms": 67
  }
```

**Latency Budget**:
- API key lookup: 2-5ms (Redis cache)
- Rate limit check: 1-2ms (Redis INCR)
- Config lookup: 1-2ms (Redis cache)
- Model inference: 30-60ms (parallel, slowest wins)
- Aggregation: 1-2ms
- Logging: <1ms (async)
- **Total**: 65-85ms (within 100ms P99 target)

### Dashboard User Flow

```
Browser
  │
  │ GET /dashboard/projects/abc123/analytics
  │ Cookie: session=...
  ▼
Nginx Ingress
  ▼
Web Dashboard (Next.js)
  │
  ├─[1]─► Verify JWT from cookie
  │        └─ Redis: Check session not revoked
  │
  ├─[2]─► Check RBAC: Can user access project?
  │        └─ Query PostgreSQL
  │
  ├─[3]─► Fetch analytics data (SSR)
  │        └─ Platform API: GET /api/v1/projects/{id}/analytics
  │             └─ PostgreSQL: Query hourly_stats table
  │
  └─[4]─► Render HTML with charts
          └─ Send to browser

  ▼
Rendered HTML + hydration data
```

---

## Data Flow

### Write Path (Request Logging)

```
Guardrail Server
  │ After validation
  │
  └─► Redis Streams: XADD guardrail_logs
       │ {request_id, flagged, results, text, ...}
       │
       ▼
Analytics Worker (Celery)
  │ XREADGROUP (batch of 100, every 5s)
  │
  ├─► PostgreSQL: INSERT request_logs (batch)
  │    • project_id, flagged, model_results (JSONB)
  │    • latency_ms, created_at
  │    • s3_key (reference to full payload)
  │
  ├─► S3: Upload full payload (gzipped JSON)
  │    • raw-requests/tenant=xxx/date=2025-01-15/hour=10/{request_id}.json.gz
  │
  └─► Redis: XACK (mark processed)
```

### Read Path (Analytics)

```
Dashboard
  │
  │ GET /dashboard/analytics
  ▼
Platform API
  │
  │ Query: Last 24h stats for project
  │
  └─► PostgreSQL
       │
       ├─► hourly_stats table
       │    • Pre-aggregated counts, averages
       │    • GROUP BY hour_bucket
       │
       └─► request_logs table (if drilling down)
            • WHERE project_id = ? AND flagged = true
            • LIMIT 100
            • JOIN s3_key for full payload
```

### Aggregation (Scheduled Jobs)

```
Celery Beat (scheduler)
  │
  │ Every hour at :05
  ▼
Hourly Aggregation Task
  │
  │ SELECT
  │   project_id,
  │   COUNT(*) as total_requests,
  │   COUNT(*) FILTER (WHERE flagged) as flagged_requests,
  │   AVG(latency_ms) as avg_latency,
  │   PERCENTILE_CONT(0.99) as p99
  │ FROM request_logs
  │ WHERE created_at >= hour_start AND created_at < hour_end
  │ GROUP BY project_id
  │
  └─► UPSERT INTO hourly_stats
```

---

## Failure Handling

### Circuit Breaker (Per-Model)

**State Machine**:
```
CLOSED (normal) ──[5 failures]──> OPEN (reject requests)
     ▲                                │
     │                                │ [30s timeout]
     │                                ▼
     └──────[3 successes]────── HALF_OPEN (probe)
                                     │
                                     │ [any failure]
                                     └────> OPEN
```

**Implementation**: `apps/guardrail-server/src/guardrail/core/circuit_breaker.py`

**Metrics**: `guardrail_circuit_breaker_state{model}` (0=closed, 1=open, 2=half-open)

### Graceful Degradation

| Failure | Response | User Impact |
|---------|----------|-------------|
| **Single model down** | Skip model, set `partial_failure: true` | Validation with 3/4 models |
| **All models down** | Return 503 Service Unavailable | Request fails |
| **Redis down** | Fallback to PostgreSQL for auth/config | Slight latency increase |
| **PostgreSQL down** | Serve from Redis cache if available | Dashboard unavailable |
| **S3 down** | Skip archival, log error | No cold storage (temporary) |

### Rate Limiting

**Levels**:
1. **Ingress**: 100 req/s global (Nginx)
2. **Per-tenant**: Configurable (default 100 req/s)
3. **Per-project**: Configurable

**Implementation**: Redis INCR with expiry

```python
# Sliding window (1 second)
key = f"rate_limit:{tenant_id}:{current_second}"
count = await redis.incr(key)
if count == 1:
    await redis.expire(key, 2)  # Expire after 2s

if count > limit:
    return 429  # Too Many Requests
```

---

## Security Architecture

### Authentication

**API Keys** (for client applications):
- SHA-256 hashed storage
- Cached in Redis (5min TTL)
- Format: `sk_live_...` or `sk_test_...`

**JWT** (for dashboard users):
- Short-lived access tokens (15min)
- Long-lived refresh tokens (30 days)
- HTTP-only, Secure, SameSite cookies

### Authorization

**RBAC Model**:
```
Organization
  ├─ owner   → Full access (can delete org)
  ├─ admin   → Manage users, projects, keys
  ├─ member  → Access assigned projects
  └─ viewer  → Read-only
```

**Enforcement**:
- Every API endpoint checks permissions
- Project-level access for non-admins
- Row-level security (RLS) in PostgreSQL

### Network Security

**Kubernetes Network Policies**:
- Guardrail Server → ML models only
- Platform API → PostgreSQL, Redis only
- Dashboard → Platform API only
- No pod-to-pod by default (deny-all)

---

## Scaling Strategy

### Horizontal Pod Autoscaler (HPA)

**Guardrail Server**:
```yaml
minReplicas: 2
maxReplicas: 8
metrics:
  - type: Pods
    pods:
      metric:
        name: guardrail_in_flight_requests
      target:
        averageValue: "5"  # Scale up if >5 concurrent per pod
```

**ML Models**:
```yaml
minReplicas: 2
maxReplicas: 6
metrics:
  - type: Pods
    pods:
      metric:
        name: model_latency_ms_p99
      target:
        averageValue: "70"  # Scale up if P99 > 70ms
```

### Database Scaling

**PostgreSQL**:
- Primary-standby replication
- Read replicas for analytics queries
- Partitioning for `request_logs` (monthly)

**Redis**:
- Redis Sentinel for HA
- 3 replicas (1 primary, 2 replicas)
- AOF + RDB persistence

---

## Observability

### Metrics (Prometheus)

**Guardrail Server**:
```
guardrail_request_total{status, flagged}
guardrail_request_latency_seconds{quantile}
guardrail_in_flight_requests
guardrail_circuit_breaker_state{model}
```

**ML Models**:
```
model_inference_latency_seconds{model, quantile}
model_inference_total{model, status}
model_in_flight{model}
```

### Logs (Loki)

- Structured JSON logs
- Correlation IDs (request_id)
- Log levels: DEBUG, INFO, WARN, ERROR

### Alerts

**Critical**:
- Latency P99 > 100ms for 5min
- All models down
- PostgreSQL down

**Warning**:
- Single model down
- Latency P99 > 80ms
- Redis memory > 80%

---

## Configuration Management

### Environment Variables

**Guardrail Server**:
```bash
REDIS_URL=redis://redis:6379/0
DATABASE_URL=postgresql://...
MODEL_PROMPT_GUARD_URL=http://model-prompt-guard:8000
MODEL_TIMEOUT_SECONDS=0.08
```

### ConfigMaps

```yaml
# Kubernetes ConfigMap
MODEL_TIMEOUT_SECONDS: "0.08"
CB_FAILURE_THRESHOLD: "5"
DEFAULT_RATE_LIMIT_QPS: "100"
```

### Secrets

```yaml
# Kubernetes Secret
redis-url: "..."
database-url: "..."
jwt-secret-key: "..."
```

---

## Next Steps

- **Database**: [03_DATABASE.md](./03_DATABASE.md) - Schema and models
- **API**: [04_API_REFERENCE.md](./04_API_REFERENCE.md) - Endpoint specs
- **Deployment**: [05_DEPLOYMENT.md](./05_DEPLOYMENT.md) - Kubernetes setup
- **Operations**: [06_OPERATIONS.md](./06_OPERATIONS.md) - Monitoring and runbooks
