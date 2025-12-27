# Documentation

**Complete documentation for the polyglot monorepo and all projects within.**

---

## Quick Start

### New Developers

1. 🚀 [Getting Started](./getting-started/README.md) - Environment setup
2. 📖 [Moon Guide](./getting-started/moon-guide.md) - Task runner reference
3. 🏗️ [Project Documentation](#projects) - Choose your project

### Returning Developers

- **Run tasks**: `moon run <project>:dev`
- **Run tests**: `moon run :test --touched`
- **Build images**: `moon run :docker --touched`
- **Check health**: `moon check`

---

## Documentation Structure

```
docs/
├── getting-started/          # Onboarding guides
│   ├── README.md            # Environment setup
│   └── moon-guide.md        # Moon task runner guide
│
└── projects/                # Project-specific documentation
    └── llm-guardrails-platform/
        ├── README.md                        # Navigation guide
        ├── 00_OVERVIEW.md                   # High-level vision
        ├── 01_DEVELOPER_JOURNEY.md          # Build guide (step-by-step)
        ├── 02_ARCHITECTURE.md               # System design
        ├── 03_DATABASE.md                   # Schema & queries
        ├── 04_API_REFERENCE.md              # API endpoints
        └── 05_DEPLOYMENT_AND_OPERATIONS.md  # K8s & operations
```

---

## Getting Started

### Prerequisites

Ensure these tools are installed:

| Tool | Purpose | Install |
|------|---------|---------|
| `uv` | Python package manager | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| `moon` | Task runner | `npm install -g @moonrepo/cli` |
| `docker` | Containers | [docker.com/get-docker](https://www.docker.com/get-docker) |
| `kubectl` | Kubernetes CLI | `brew install kubectl` |

### Quick Setup

```bash
# Clone and enter repo
git clone <repo-url>
cd playground

# Install dependencies
uv sync

# Verify moon setup
moon query projects

# Run tests
moon run :test
```

**Full setup**: [getting-started/README.md](./getting-started/README.md)

---

## Projects

### LLM Guardrails Platform

**A multi-tenant SaaS platform providing real-time LLM safety guardrails.**

📁 **Location**: `apps/` and `packages/`
📖 **Docs**: [projects/llm-guardrails-platform/](./projects/llm-guardrails-platform/)

**Services**:
- `guardrail-server` - Real-time ML inference orchestration (Python/FastAPI)
- `platform-api` - Tenant/user/project management (Python/FastAPI)
- `analytics-worker` - Background data processing (Python/Celery)
- `web-dashboard` - Management UI (Next.js)
- `model-*` - ML model services (4 models)

**Quick links**:
- [Overview](./projects/llm-guardrails-platform/00_OVERVIEW.md) - What it does
- [Developer Journey](./projects/llm-guardrails-platform/01_DEVELOPER_JOURNEY.md) - How to build it ⭐
- [Architecture](./projects/llm-guardrails-platform/02_ARCHITECTURE.md) - How it works
- [Database](./projects/llm-guardrails-platform/03_DATABASE.md) - Schema reference
- [API Reference](./projects/llm-guardrails-platform/04_API_REFERENCE.md) - Endpoints
- [Deployment](./projects/llm-guardrails-platform/05_DEPLOYMENT_AND_OPERATIONS.md) - Operations

---

## Core Guides

### [Getting Started](./getting-started/README.md)

**Topics covered**:
- Tool installation (uv, moon, docker, kubectl)
- Repository structure
- Development workflow
- Common commands
- Troubleshooting

**Read this first if**: You're new to the repository.

### [Moon Guide](./getting-started/moon-guide.md)

**Topics covered**:
- What is moon and why use it
- Core concepts (projects, tasks, dependencies)
- Common commands
- Task configuration
- Caching and change detection
- Integration with Python (uv) and Go workspaces

**Read this if**: You need to understand the task runner or add new tasks.

---

## Additional Resources

### Repository Guidelines

- **[CLAUDE.md](../CLAUDE.md)** - Project vision, commit guidelines, AI agent rules
- **[README.md](../README.md)** - Repository overview

### Infrastructure

- **[infra/RUNBOOK.md](../infra/RUNBOOK.md)** - OCI Kubernetes setup guide
- **[infra/terraform/](../infra/terraform/)** - Infrastructure as Code
- **[infra/k8s/](../infra/k8s/)** - Kubernetes manifests

---

## Common Tasks

### Development

```bash
# Start a service
moon run <project>:dev

# Run tests
moon run :test

# Run tests for changed projects only
moon run :test --touched

# Format code
moon run :format

# Lint code
moon run :lint
```

### Building

```bash
# Build all projects
moon run :build

# Build changed projects only
moon run :build --touched

# Build Docker images
moon run :docker --touched
```

### Deployment

```bash
# Deploy to Kubernetes
kubectl apply -k infra/k8s/

# Check deployment status
kubectl get pods

# View logs
kubectl logs -l app=<service-name>
```

---

## Development Workflow

### Daily Workflow

```bash
# 1. Pull latest changes
git pull --rebase origin main

# 2. Install dependencies
uv sync

# 3. Run tests
moon run :test --touched

# 4. Start development
moon run <project>:dev

# 5. Make changes and test
moon run <project>:test
```

### Before Committing

```bash
# Run all checks
moon run :format --touched
moon run :lint --touched
moon run :test --touched
moon run :build --touched

# Commit (no AI attribution)
git commit -m "feat: your descriptive message"
```

---

## Finding What You Need

### "I want to..."

**...set up my development environment**
→ [getting-started/README.md](./getting-started/README.md)

**...learn moon task runner**
→ [getting-started/moon-guide.md](./getting-started/moon-guide.md)

**...understand the LLM Guardrails Platform**
→ [projects/llm-guardrails-platform/00_OVERVIEW.md](./projects/llm-guardrails-platform/00_OVERVIEW.md)

**...build the LLM Guardrails Platform from scratch**
→ [projects/llm-guardrails-platform/01_DEVELOPER_JOURNEY.md](./projects/llm-guardrails-platform/01_DEVELOPER_JOURNEY.md) ⭐

**...understand the system architecture**
→ [projects/llm-guardrails-platform/02_ARCHITECTURE.md](./projects/llm-guardrails-platform/02_ARCHITECTURE.md)

**...see API endpoints**
→ [projects/llm-guardrails-platform/04_API_REFERENCE.md](./projects/llm-guardrails-platform/04_API_REFERENCE.md)

**...deploy to Kubernetes**
→ [projects/llm-guardrails-platform/05_DEPLOYMENT_AND_OPERATIONS.md](./projects/llm-guardrails-platform/05_DEPLOYMENT_AND_OPERATIONS.md)

**...set up infrastructure**
→ [infra/RUNBOOK.md](../infra/RUNBOOK.md)

---

## Documentation Principles

Following [CLAUDE.md](../CLAUDE.md) guidelines:

1. **Single Source of Truth** - Each concept documented in one place
2. **No Duplication** - Cross-reference rather than copy
3. **Educational** - Explain the "why", not just the "what"
4. **Actionable** - Provide exact commands and examples
5. **Maintained** - Keep docs in sync with code

---

## Adding New Projects

When adding a new project to the monorepo:

1. Create project folder: `apps/my-new-project/` or `packages/my-library/`
2. Add `moon.yml` with tasks
3. Add to workspace: `pyproject.toml` (Python) or `go.work` (Go)
4. Create documentation: `docs/projects/my-new-project/`
5. Follow the structure from `llm-guardrails-platform/` as a template

---

## Questions?

- **Setup issues**: See [getting-started/README.md](./getting-started/README.md#troubleshooting)
- **Moon questions**: See [getting-started/moon-guide.md](./getting-started/moon-guide.md#troubleshooting)
- **Infrastructure**: See [infra/RUNBOOK.md](../infra/RUNBOOK.md#troubleshooting)
- **Project-specific**: Check project's README in `docs/projects/<project-name>/`

---

## Contributing

See [CLAUDE.md](../CLAUDE.md) for:
- Commit message format (conventional commits, no AI attribution)
- Git workflow (rebase-only, trunk-based development)
- Documentation guidelines (single source of truth)
- Development principles (simplicity first, educational commits)

---

**Ready to start?** Begin with [getting-started/README.md](./getting-started/README.md)!
