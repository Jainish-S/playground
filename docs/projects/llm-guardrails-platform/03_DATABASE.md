# LLM Guardrails Platform - Database Schema

**PostgreSQL schema for multi-tenant platform.**

---

## Entity Relationship Overview

```
organizations ──┬── org_memberships (users)
                └── projects ──┬── dag_configs
                               ├── api_keys
                               ├── request_logs
                               └── hourly_stats
```

---

## Core Tables

### organizations

```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    settings JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(320) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    full_name VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### org_memberships

```sql
CREATE TABLE org_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, org_id)
);
```

### projects

```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active_dag_version INT NOT NULL DEFAULT 1,
    settings JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);
```

### dag_configs

```sql
CREATE TABLE dag_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version INT NOT NULL,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, version)
);
```

**Example config**:
```json
{
    "enabled_models": [
        {"id": "prompt-guard", "endpoint": "http://...", "threshold": 0.5},
        {"id": "pii-detect", "endpoint": "http://...", "threshold": 0.7}
    ],
    "aggregation_strategy": "any_flag",
    "timeout_ms": 80
}
```

### api_keys

```sql
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    project_ids UUID[] NOT NULL,
    rate_limit_qps INT NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_test BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### request_logs (Partitioned)

```sql
CREATE TABLE request_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    project_id UUID NOT NULL,
    dag_version INT NOT NULL,
    request_type VARCHAR(20) NOT NULL DEFAULT 'input',
    flagged BOOLEAN NOT NULL,
    flag_reasons TEXT[] NOT NULL DEFAULT '{}',
    model_results JSONB NOT NULL,
    partial_failure BOOLEAN NOT NULL DEFAULT false,
    failed_models TEXT[] NOT NULL DEFAULT '{}',
    latency_ms INT NOT NULL,
    s3_key VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE request_logs_2025_01 PARTITION OF request_logs
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### hourly_stats

```sql
CREATE TABLE hourly_stats (
    id BIGSERIAL PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    hour_bucket TIMESTAMPTZ NOT NULL,
    total_requests INT NOT NULL DEFAULT 0,
    flagged_requests INT NOT NULL DEFAULT 0,
    partial_failure_count INT NOT NULL DEFAULT 0,
    model_stats JSONB NOT NULL DEFAULT '{}',
    avg_latency_ms FLOAT NOT NULL DEFAULT 0,
    p99_latency_ms FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, hour_bucket)
);
```

---

## Indexes

```sql
-- Organizations
CREATE INDEX idx_orgs_slug ON organizations(slug);

-- Memberships
CREATE INDEX idx_memberships_user ON org_memberships(user_id);
CREATE INDEX idx_memberships_org ON org_memberships(org_id);

-- Projects
CREATE INDEX idx_projects_org ON projects(org_id);

-- API Keys
CREATE INDEX idx_api_keys_org ON api_keys(org_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- Request Logs
CREATE INDEX idx_logs_tenant_time ON request_logs(tenant_id, created_at DESC);
CREATE INDEX idx_logs_project_time ON request_logs(project_id, created_at DESC);
CREATE INDEX idx_logs_flagged ON request_logs(project_id, created_at DESC) WHERE flagged = true;

-- Hourly Stats
CREATE INDEX idx_hourly_project_time ON hourly_stats(project_id, hour_bucket DESC);
```

---

## RBAC Queries

### Check user permission

```sql
-- Check if user can access project
SELECT 1
FROM org_memberships om
JOIN projects p ON p.org_id = om.org_id
WHERE om.user_id = $1
  AND p.id = $2
  AND om.role IN ('owner', 'admin', 'member');
```

### Get accessible projects

```sql
-- For owner/admin: all projects
-- For member: only assigned projects
SELECT p.id, p.name
FROM projects p
JOIN org_memberships om ON om.org_id = p.org_id
WHERE om.user_id = $1
  AND om.role IN ('owner', 'admin');
```

---

## Common Queries

### Validate API key

```sql
SELECT id, org_id, project_ids, rate_limit_qps
FROM api_keys
WHERE key_hash = $1
  AND is_active = true
  AND (expires_at IS NULL OR expires_at > NOW());
```

### Get project analytics

```sql
SELECT
    hour_bucket,
    total_requests,
    flagged_requests,
    avg_latency_ms,
    p99_latency_ms
FROM hourly_stats
WHERE project_id = $1
  AND hour_bucket >= $2
  AND hour_bucket < $3
ORDER BY hour_bucket ASC;
```

### Get recent flagged requests

```sql
SELECT id, created_at, flag_reasons, model_results
FROM request_logs
WHERE project_id = $1
  AND flagged = true
  AND created_at >= NOW() - INTERVAL '7 days'
ORDER BY created_at DESC
LIMIT 100;
```

---

## Migrations (Alembic)

### Setup

```bash
cd apps/platform-api
uv add alembic asyncpg
uv run alembic init migrations
```

### Create migration

```bash
uv run alembic revision -m "create organizations table"
```

### Apply migrations

```bash
uv run alembic upgrade head
```

---

## Data Retention

| Table | Hot | Cold | Total |
|-------|-----|------|-------|
| request_logs | 30 days (PostgreSQL) | 60 days (archive) | 90 days |
| hourly_stats | Forever | - | Forever |
| api_keys | Until revoked | - | Until revoked |
| users | Forever | - | Forever |

**Lifecycle Policy**:
```sql
-- Drop old partitions
DROP TABLE request_logs_2024_10;

-- Archive to S3 before dropping
pg_dump -t request_logs_2024_10 | gzip > s3://...
```

---

## Performance Tuning

### Partitioning Strategy

Monthly partitions for `request_logs`:
```sql
-- Automated partition creation
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
BEGIN
    partition_date := DATE_TRUNC('month', NOW() + INTERVAL '1 month');
    partition_name := 'request_logs_' || TO_CHAR(partition_date, 'YYYY_MM');

    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF request_logs FOR VALUES FROM (%L) TO (%L)',
        partition_name,
        partition_date,
        partition_date + INTERVAL '1 month'
    );
END;
$$ LANGUAGE plpgsql;
```

### Connection Pooling

```python
# Use asyncpg with connection pooling
pool = await asyncpg.create_pool(
    dsn=DATABASE_URL,
    min_size=5,
    max_size=20
)
```

---

## Backup Strategy

### Daily Backups

```bash
# Full backup
pg_dump -Fc guardrails > backup_$(date +%Y%m%d).dump

# Upload to S3
aws s3 cp backup_*.dump s3://backups/postgresql/
```

### Point-in-Time Recovery

```bash
# Enable WAL archiving
archive_mode = on
archive_command = 'aws s3 cp %p s3://backups/wal/%f'

# Restore to specific time
pg_restore -d guardrails -t '2025-01-15 14:30:00' backup.dump
```

---

## Next Steps

- **API Reference**: [04_API_REFERENCE.md](./04_API_REFERENCE.md)
- **Deployment**: [05_DEPLOYMENT.md](./05_DEPLOYMENT.md)
