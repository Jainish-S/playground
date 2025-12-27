# LLM Guardrails Platform - Documentation

**Complete documentation for building a multi-tenant SaaS platform providing real-time LLM safety guardrails.**

---

## Quick Navigation

### 🚀 Getting Started

**New to the project?** Start here:
1. 📖 [Overview](./00_OVERVIEW.md) - What is this platform? (10 min read)
2. 🛠️ [Developer Journey](./01_DEVELOPER_JOURNEY.md) - Step-by-step build guide (essential!)

### 📚 Deep Dives

**Building the platform?** Reference these:
3. 🏗️ [Architecture](./02_ARCHITECTURE.md) - Detailed system design
4. 🗄️ [Database](./03_DATABASE.md) - Schema, queries, migrations
5. 🔌 [API Reference](./04_API_REFERENCE.md) - All endpoints documented
6. 🚀 [Deployment & Operations](./05_DEPLOYMENT_AND_OPERATIONS.md) - Kubernetes, monitoring, runbooks

---

## Document Overview

### [00_OVERVIEW.md](./00_OVERVIEW.md)

**What you'll learn**:
- Platform capabilities and use cases
- Service architecture at a glance
- Technology choices and rationale
- Design principles
- Performance targets
- Data retention policies

**Read this if**: You want a high-level understanding before diving into implementation.

**Time**: 10-15 minutes

---

### [01_DEVELOPER_JOURNEY.md](./01_DEVELOPER_JOURNEY.md) ⭐ **MUST READ**

**What you'll learn**:
- **Phase 1**: Repository structure setup with moon
- **Phase 2**: Guardrail Server MVP (FastAPI + single model)
- **Phase 3**: ML model services (4 models) + parallel orchestration
- **Phase 4-8**: Platform API, Analytics, Dashboard, Production hardening

**Key features**:
- ✅ Exact commands to run
- ✅ File-by-file code examples
- ✅ Checkpoints after each phase
- ✅ Troubleshooting tips
- ✅ Aligned with moon workflow

**Read this if**: You're building the platform from scratch.

**Time**: Reference document (1-2 hours to skim, use during implementation)

---

### [02_ARCHITECTURE.md](./02_ARCHITECTURE.md)

**What you'll learn**:
- Service responsibilities and boundaries
- Request flows (validation, dashboard, analytics)
- Data flows (write path, read path, aggregation)
- Failure handling (circuit breakers, graceful degradation)
- Security architecture
- Scaling strategy

**Read this if**: You need to understand how components interact.

**Time**: 20-30 minutes

---

### [03_DATABASE.md](./03_DATABASE.md)

**What you'll learn**:
- Complete PostgreSQL schema (organizations, users, projects, API keys, logs)
- RBAC queries
- Common queries (analytics, validation)
- Partitioning strategy
- Migrations with Alembic
- Backup strategy

**Read this if**: You're implementing the Platform API or Analytics.

**Time**: 15-20 minutes (reference during development)

---

### [04_API_REFERENCE.md](./04_API_REFERENCE.md)

**What you'll learn**:
- Guardrail API (`/v1/validate`)
- Platform API (auth, orgs, projects, API keys, analytics)
- ML model service interface
- Error responses
- Rate limits
- SDK examples

**Read this if**: You're implementing API endpoints or integrating with the platform.

**Time**: Reference document (browse as needed)

---

### [05_DEPLOYMENT_AND_OPERATIONS.md](./05_DEPLOYMENT_AND_OPERATIONS.md)

**What you'll learn**:
- Kubernetes deployment steps
- CI/CD pipeline setup
- Monitoring with Prometheus + Grafana
- Alerting rules
- Incident response runbooks
- Maintenance procedures (scaling, updates, backups)

**Read this if**: You're deploying to production or operating the platform.

**Time**: Reference document (critical for operations)

---

## Reading Paths

### Path 1: "I want to understand the platform"

1. [00_OVERVIEW.md](./00_OVERVIEW.md) - Big picture
2. [02_ARCHITECTURE.md](./02_ARCHITECTURE.md) - How it works
3. [04_API_REFERENCE.md](./04_API_REFERENCE.md) - What can it do?

**Time**: ~1 hour

---

### Path 2: "I want to build this"

1. [00_OVERVIEW.md](./00_OVERVIEW.md) - Context
2. [01_DEVELOPER_JOURNEY.md](./01_DEVELOPER_JOURNEY.md) - **Start here and follow step-by-step**
3. Refer to [02_ARCHITECTURE.md](./02_ARCHITECTURE.md), [03_DATABASE.md](./03_DATABASE.md), [04_API_REFERENCE.md](./04_API_REFERENCE.md) as needed during implementation

**Time**: 12 weeks of implementation

---

### Path 3: "I need to deploy/operate this"

1. [00_OVERVIEW.md](./00_OVERVIEW.md) - What you're deploying
2. [05_DEPLOYMENT_AND_OPERATIONS.md](./05_DEPLOYMENT_AND_OPERATIONS.md) - Deployment steps
3. Set up monitoring and alerts
4. Bookmark runbooks for incidents

**Time**: 1-2 days for initial deployment + ongoing operations

---

## Prerequisites

Before starting, ensure you have:

✅ **Environment setup**: [docs/getting-started/README.md](../../getting-started/README.md)
✅ **Moon knowledge**: [docs/getting-started/moon-guide.md](../../getting-started/moon-guide.md)
✅ **Infrastructure ready**: [infra/RUNBOOK.md](../../../infra/RUNBOOK.md)

---

## Technology Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Python 3.12, FastAPI, asyncio |
| **Frontend** | Next.js 14, React Server Components |
| **Database** | PostgreSQL 15 (partitioned tables) |
| **Cache/Queue** | Redis 7 (Streams, caching) |
| **ML Models** | Hugging Face Transformers, PyTorch |
| **Orchestration** | Kubernetes (OKE) |
| **Monitoring** | Prometheus, Grafana, Loki |
| **Task Runner** | Moon (moonrepo.dev) |
| **Package Manager** | uv (Python), npm (Node.js) |

---

## Architecture at a Glance

```
┌────────────────────────────────────────────────┐
│           CLIENT APPLICATIONS                  │
└────────────────────────────────────────────────┘
                    ▼
┌────────────────────────────────────────────────┐
│              Nginx Ingress (K8s)               │
└────────────────────────────────────────────────┘
          │               │               │
    ┌─────┴──────┐  ┌────┴────┐  ┌───────┴────────┐
    │ Guardrail  │  │Platform │  │   Dashboard    │
    │  Server    │  │   API   │  │   (Next.js)    │
    └─────┬──────┘  └─────────┘  └────────────────┘
          │
    ┌─────┴──────────────────┐
    │  ML Models (4 services)│
    └────────────────────────┘
          │
    ┌─────┴──────────────────┐
    │ PostgreSQL, Redis, S3  │
    └────────────────────────┘
```

---

## Performance Targets

| Metric | Target | Strategy |
|--------|--------|----------|
| Latency P50 | < 60ms | Parallel model calls, caching |
| Latency P99 | < 100ms | Circuit breakers, timeouts |
| Throughput | 200 QPS sustained | Horizontal scaling (HPA) |
| Peak QPS | 1000 (5x burst) | Auto-scaling + rate limiting |
| Uptime | 99.9% | Redundancy, health checks |

---

## Development Workflow

### Daily Commands

```bash
# Start development server
moon run guardrail-server:dev

# Run tests (only changed)
moon run :test --touched

# Build Docker images
moon run :docker --touched

# Deploy to K8s
kubectl apply -k infra/k8s/
```

### Before Committing

```bash
# Run all checks
moon run :lint --touched
moon run :format --touched
moon run :test --touched
moon run :build --touched
```

---

## Project Status

**Current Phase**: ✅ Phase 1 - Infrastructure Complete

**Next Steps**: Follow [01_DEVELOPER_JOURNEY.md](./01_DEVELOPER_JOURNEY.md) starting from Phase 2

---

## Contributing

See [CLAUDE.md](../../../CLAUDE.md) for:
- Commit message guidelines (no AI attribution)
- Git workflow (rebase-only)
- Documentation principles (no duplication)

---

## Questions?

- **Setup issues**: See [docs/getting-started/README.md](../../getting-started/README.md)
- **Moon questions**: See [docs/getting-started/moon-guide.md](../../getting-started/moon-guide.md)
- **Infrastructure**: See [infra/RUNBOOK.md](../../../infra/RUNBOOK.md)
- **Platform architecture**: See [02_ARCHITECTURE.md](./02_ARCHITECTURE.md)

---

## Document Maintenance

These documents are **living references** and should be updated as the platform evolves:

- Keep [01_DEVELOPER_JOURNEY.md](./01_DEVELOPER_JOURNEY.md) in sync with actual implementation
- Update [04_API_REFERENCE.md](./04_API_REFERENCE.md) when endpoints change
- Add new runbooks to [05_DEPLOYMENT_AND_OPERATIONS.md](./05_DEPLOYMENT_AND_OPERATIONS.md)

---

**Ready to build?** Start with [00_OVERVIEW.md](./00_OVERVIEW.md) then dive into [01_DEVELOPER_JOURNEY.md](./01_DEVELOPER_JOURNEY.md)!
