# Getting Started

Welcome to the polyglot monorepo! This guide will help you set up your development environment and understand the repository structure.

## Prerequisites

### Required Tools

| Tool | Purpose | Install | Verify |
|------|---------|---------|--------|
| **uv** | Python package manager | `curl -LsSf https://astral.sh/uv/install.sh \| sh` | `uv --version` |
| **Go** | Go language (1.21+) | `brew install go` | `go version` |
| **Node.js** | JavaScript runtime (20+) | `brew install node` | `node --version` |
| **moon** | Task runner | `npm install -g @moonrepo/cli` | `moon --version` |
| **Docker** | Containerization | [docker.com/get-docker](https://www.docker.com/get-docker) | `docker --version` |
| **kubectl** | Kubernetes CLI | `brew install kubectl` | `kubectl version --client` |

### Optional but Recommended

| Tool | Purpose | Install |
|------|---------|---------|
| **git** | Version control | `brew install git` |
| **make** | Build automation | Built-in on macOS/Linux |
| **jq** | JSON processor | `brew install jq` |
| **ripgrep** | Fast search | `brew install ripgrep` |

---

## Quick Setup

### 1. Clone the Repository

```bash
git clone <repo-url>
cd playground
```

### 2. Install Dependencies

```bash
# Python: Install all workspace dependencies
uv sync

# Go: Download module dependencies
go mod download

# Node.js: Install packages (if using Next.js dashboard)
npm install
```

### 3. Verify Moon Setup

```bash
# List all projects
moon query projects

# Check health
moon check
```

### 4. Run Your First Task

```bash
# Run all tests
moon run :test

# Start a development server (example)
moon run guardrail-server:dev
```

---

## Repository Structure

```
.
├── .moon/                  # Moon configuration
│   ├── workspace.yml       # Project discovery
│   ├── toolchain.yml       # Language versions
│   └── tasks.yml           # Shared task definitions
│
├── apps/                   # Deployable applications
│   ├── guardrail-server/   # Python FastAPI service
│   ├── platform-api/       # Python FastAPI API
│   ├── analytics-worker/   # Python Celery worker
│   ├── api-gateway/        # Go HTTP gateway
│   └── web-dashboard/      # Next.js frontend
│
├── packages/               # Shared libraries
│   ├── api-contracts/      # Protobuf definitions
│   ├── go-common/          # Shared Go utilities
│   └── py-common/          # Shared Python utilities
│
├── infra/                  # Infrastructure as Code
│   ├── terraform/          # OCI resources
│   └── k8s/                # Kubernetes manifests
│
├── docs/                   # Documentation
│   ├── getting-started/    # Setup guides (this file!)
│   └── projects/           # Project-specific docs
│
├── pyproject.toml          # Root Python workspace
├── uv.lock                 # Python lockfile
├── go.work                 # Go workspace
└── CLAUDE.md               # AI assistant guidelines
```

---

## Development Workflow

### Daily Workflow

```bash
# 1. Sync latest changes
git pull --rebase origin main

# 2. Install new dependencies
uv sync

# 3. Run tests before starting work
moon run :test --touched

# 4. Start your development server
moon run <project>:dev

# 5. Run tests after changes
moon run <project>:test

# 6. Format and lint
moon run <project>:format
moon run <project>:lint
```

### Before Committing

```bash
# Run all checks on changed projects
moon run :test --touched
moon run :lint --touched
moon run :build --touched

# Or use the pre-commit hook (if configured)
git commit -m "feat: your message"
```

---

## Working with Moon

Moon is the task runner that orchestrates all builds, tests, and deployments.

### Core Commands

```bash
# List all projects
moon query projects

# Run a task for specific project
moon run <project>:<task>
# Example: moon run guardrail-server:test

# Run task across all projects
moon run :<task>
# Example: moon run :test

# Run only on changed projects
moon run :test --touched

# CI mode (no local tasks, stricter)
moon ci --touched
```

### Common Tasks

Most projects support these standard tasks:

| Task | Purpose | Example |
|------|---------|---------|
| `dev` | Start development server | `moon run platform-api:dev` |
| `test` | Run unit/integration tests | `moon run :test` |
| `build` | Build artifacts | `moon run :build` |
| `lint` | Check code style | `moon run :lint` |
| `format` | Auto-format code | `moon run :format` |
| `docker` | Build Docker image | `moon run guardrail-server:docker` |

**Learn more:** [Moon Guide](./moon-guide.md)

---

## Python Development

### Workspace Structure

All Python projects share a single virtual environment at the root:

```toml
# Root pyproject.toml
[tool.uv.workspace]
members = [
    "apps/guardrail-server",
    "apps/platform-api",
    "apps/analytics-worker",
    "packages/py-common"
]
```

### Adding Dependencies

```bash
# Add to specific project
cd apps/guardrail-server
uv add fastapi uvicorn

# Add dev dependency
uv add --dev pytest pytest-cov

# Sync all projects
cd ../..
uv sync
```

### Running Python Services

```bash
# Via moon (preferred)
moon run guardrail-server:dev

# Direct (for debugging)
cd apps/guardrail-server
uv run uvicorn src.main:app --reload
```

---

## Go Development

### Workspace Structure

Go modules are linked via `go.work`:

```go
// go.work
use (
    ./apps/api-gateway
    ./packages/go-common
)
```

### Adding Dependencies

```bash
cd apps/api-gateway
go get github.com/gin-gonic/gin
```

### Running Go Services

```bash
# Via moon
moon run api-gateway:dev

# Direct
cd apps/api-gateway
go run ./cmd/server
```

---

## Docker & Kubernetes

### Building Images

```bash
# Build all images for changed services
moon run :docker --touched

# Build specific service
moon run platform-api:docker
```

### Local Development

```bash
# Start services via docker-compose (if configured)
docker-compose up -d

# Or use moon dev tasks
moon run :dev
```

### Deploying to Kubernetes

```bash
# Deploy infrastructure first
cd infra/terraform/environments/dev
terraform apply

# Deploy applications
kubectl apply -k infra/k8s/
```

**See:** [Infrastructure Runbook](../../infra/RUNBOOK.md)

---

## Useful Resources

### Documentation

- [Moon Guide](./moon-guide.md) - Task runner deep dive
- [CLAUDE.md](../../CLAUDE.md) - Repository philosophy and AI guidelines
- [Infrastructure Runbook](../../infra/RUNBOOK.md) - Kubernetes setup

### Project Documentation

- [LLM Guardrails Platform](../projects/llm-guardrails-platform/00_OVERVIEW.md) - Multi-tenant SaaS platform

### External Links

- [Moon Documentation](https://moonrepo.dev/docs)
- [uv Documentation](https://docs.astral.sh/uv/)
- [Go Documentation](https://go.dev/doc/)

---

## Troubleshooting

### Moon Cache Issues

```bash
# Clear caches
moon clean --cache

# Force rebuild
moon run :build --force
```

### Python Environment Issues

```bash
# Remove and recreate venv
rm -rf .venv
uv sync
```

### Go Module Issues

```bash
# Tidy modules
cd apps/api-gateway
go mod tidy
```

### Docker Build Failures

```bash
# Clean build (no cache)
docker system prune -af
moon run :docker --force
```

---

## Getting Help

### Moon Commands

```bash
# Show help
moon --help
moon run --help

# Check project configuration
moon query projects <project-name>

# Show task details
moon query tasks <project>
```

### Community

- **GitHub Issues**: File bugs or feature requests
- **Discussions**: Ask questions in GitHub Discussions
- **Documentation**: Check `docs/` folder

---

## Next Steps

1. ✅ **Complete this setup guide**
2. 📖 **Read the [Moon Guide](./moon-guide.md)** to understand task management
3. 🏗️ **Explore a project**: Start with [LLM Guardrails Platform](../projects/llm-guardrails-platform/00_OVERVIEW.md)
4. 💻 **Run your first task**: `moon run :test`
5. 🚀 **Start developing**: Pick a project and run its `dev` task

Happy coding! 🎉
