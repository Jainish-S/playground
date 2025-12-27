# Moon: Task Runner Guide

This guide documents moon (moonrepo.dev), the task runner and monorepo orchestration tool used in this repository.

## What is Moon?

Moon is a **polyglot task runner** that sits between simple scripts (make, npm scripts) and complex build systems (Bazel, Nx). It's written in Rust for speed and reliability.

### Key Benefits for This Repo

1. **Smart Caching**: Only rebuild what changed based on file hashes
2. **Polyglot Support**: Manages Python (uv), Go, Node.js projects together
3. **Dependency Graphs**: Runs tasks in correct order automatically
4. **Integrated Toolchains**: Auto-downloads and manages language versions
5. **Parallel Execution**: Runs independent tasks simultaneously

---

## Core Concepts

### Projects

A **project** is a deployable service, library, or package within the monorepo.

**Examples:**
```
apps/guardrail-server/      → Project ID: "guardrail-server"
apps/platform-api/          → Project ID: "platform-api"
packages/api-contracts/     → Project ID: "api-contracts"
```

**Configuration**: `.moon/workspace.yml` defines all projects

```yaml
# Example workspace.yml
projects:
  globs:
    - 'apps/*'
    - 'packages/*'
```

Each project can have an optional `moon.yml` for custom settings.

### Tasks

A **task** is a command that runs in the context of a project.

**Task Types:**
- **Build**: Generates artifacts (has `outputs` defined)
- **Test**: Validates code (default type)
- **Run**: Long-running processes like dev servers (marked `local: true`)

**Examples:**
```bash
# Run a specific task
moon run guardrail-server:dev

# Run a task across all projects
moon run :test

# Run tasks only in changed projects
moon run :test --touched
```

### Dependencies

Moon tracks two types of dependencies:

1. **Explicit**: Manually defined in `moon.yml` via `dependsOn`
2. **Implicit**: Auto-discovered by scanning imports (Python, Go, Node.js)

**Example:**
```yaml
# apps/guardrail-server/moon.yml
dependsOn:
  - 'api-contracts'  # Explicit: needs protobuf definitions
```

---

## Project Structure

This repository uses the following structure:

```
.
├── .moon/
│   ├── workspace.yml       # Project discovery and global config
│   ├── toolchain.yml       # Language versions (Python, Go, Node.js)
│   └── tasks.yml           # Shared task definitions
│
├── apps/                   # Deployable applications
│   ├── guardrail-server/
│   │   ├── moon.yml        # Project-specific config (optional)
│   │   ├── src/
│   │   └── tests/
│   └── platform-api/
│       └── ...
│
└── packages/               # Shared libraries
    ├── api-contracts/
    └── ...
```

---

## Common Commands

### Discovery

```bash
# List all projects
moon query projects

# Show project graph (dependencies)
moon query graph

# Check which projects would run for a task
moon query touched :build
```

### Running Tasks

```bash
# Run specific project:task
moon run guardrail-server:dev

# Run task across all projects
moon run :test
moon run :build

# Run only for changed projects (based on git)
moon run :test --touched

# Run in CI mode (stricter, no local tasks)
moon ci --touched
```

### Building

```bash
# Build specific project
moon run guardrail-server:build

# Build all projects
moon run :build

# Build only changed projects + their dependents
moon run :build --touched
```

### Docker

```bash
# Build Docker images for changed services
moon run :docker --touched

# Build specific service image
moon run platform-api:docker
```

---

## Task Configuration

Tasks are defined in `.moon/tasks.yml` (global) or `moon.yml` (per-project).

### Example: Python FastAPI Service

```yaml
# apps/guardrail-server/moon.yml
tasks:
  # Development server
  dev:
    command: 'uv'
    args:
      - 'run'
      - 'uvicorn'
      - 'src.main:app'
      - '--reload'
      - '--host'
      - '0.0.0.0'
      - '--port'
      - '8000'
    local: true  # Don't run in CI, don't cache

  # Tests
  test:
    command: 'uv'
    args:
      - 'run'
      - 'pytest'
      - 'tests/'
      - '--cov=src'
    inputs:
      - 'src/**/*.py'
      - 'tests/**/*.py'
      - 'pyproject.toml'

  # Build
  build:
    command: 'uv'
    args: ['sync', '--frozen']
    outputs:
      - '.venv/'

  # Docker build
  docker:
    command: 'docker'
    args:
      - 'build'
      - '-f'
      - 'Dockerfile'
      - '-t'
      - 'guardrail-server:latest'
      - '../../'  # Build from root to access uv.lock
    inputs:
      - 'src/**'
      - 'Dockerfile'
      - '../../uv.lock'
      - '../../pyproject.toml'
```

### Example: Go Service

```yaml
# apps/api-gateway/moon.yml
tasks:
  dev:
    command: 'go'
    args: ['run', './cmd/server']
    local: true

  test:
    command: 'go'
    args: ['test', './...']

  build:
    command: 'go'
    args:
      - 'build'
      - '-o'
      - 'bin/server'
      - './cmd/server'
    outputs:
      - 'bin/'

  docker:
    command: 'docker'
    args:
      - 'build'
      - '-f'
      - 'Dockerfile'
      - '-t'
      - 'api-gateway:latest'
      - '.'
```

---

## Caching

Moon caches task outputs based on:
- Task configuration hash
- Input file hashes (from `inputs` setting)
- Dependency task outputs

**Cache hits** skip execution entirely and restore outputs.

**Example workflow:**
```bash
# First run: builds everything
moon run :build

# No changes: instant (cache hit)
moon run :build

# Change one file: only rebuilds affected projects
echo "# comment" >> apps/guardrail-server/src/main.py
moon run :build  # Only rebuilds guardrail-server
```

---

## Change Detection

Moon detects changes using:
- **Git**: Compares against `main` branch (or `--base` flag)
- **File hashes**: Checks if inputs changed since last run

**Useful flags:**
```bash
# Only test changed projects
moon run :test --touched

# Compare against specific branch
moon run :test --touched --base=develop

# Show what would run (dry-run)
moon run :build --touched --dry-run
```

---

## Integration with This Repo

### Python (uv)

All Python projects share a single `uv.lock` at the root:

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

**Moon runs:** `uv sync` from project directory but uses root lockfile.

### Go Workspaces

Go modules are linked via `go.work`:

```go
// go.work
use (
    ./apps/api-gateway
    ./packages/go-common
)
```

**Moon runs:** `go build` from project directory with workspace context.

### Docker Builds

Python Dockerfiles **must** build from repository root to access `uv.lock`:

```dockerfile
# apps/guardrail-server/Dockerfile
FROM python:3.12-slim
WORKDIR /app

# Copy root files first
COPY uv.lock pyproject.toml ./
COPY apps/guardrail-server/pyproject.toml ./apps/guardrail-server/

RUN pip install uv && uv sync --frozen

COPY apps/guardrail-server/ ./apps/guardrail-server/
CMD ["uv", "run", "uvicorn", "apps.guardrail-server.src.main:app"]
```

---

## Tips & Best Practices

### 1. Use Globs for Project Discovery

```yaml
# .moon/workspace.yml
projects:
  globs:
    - 'apps/*'
    - 'packages/*'
```

New projects are auto-discovered when added to these paths.

### 2. Define Common Tasks Globally

```yaml
# .moon/tasks.yml
tasks:
  lint:
    command: 'ruff'
    args: ['check', 'src/']

  format:
    command: 'ruff'
    args: ['format', 'src/']
```

All projects inherit these. Override per-project if needed.

### 3. Mark Dev Servers as Local

```yaml
tasks:
  dev:
    command: 'uvicorn'
    args: ['src.main:app', '--reload']
    local: true  # Never caches, never runs in CI
```

### 4. Use `--touched` in CI

```yaml
# .github/workflows/ci.yml
- name: Run tests
  run: moon ci :test --touched
```

Only tests changed projects, speeding up CI dramatically.

### 5. Define Inputs Explicitly

```yaml
tasks:
  build:
    command: 'go'
    args: ['build', '-o', 'bin/server', './cmd/server']
    inputs:
      - 'cmd/**/*.go'
      - 'internal/**/*.go'
      - 'go.mod'
      - 'go.sum'
    outputs:
      - 'bin/'
```

Ensures cache invalidates when relevant files change.

---

## Troubleshooting

### Cache Issues

```bash
# Clear all caches
moon clean --cache

# Force rebuild without cache
moon run :build --force
```

### Dependency Issues

```bash
# Visualize project graph
moon query graph --dot | dot -Tpng > graph.png

# Check task dependencies
moon query tasks guardrail-server
```

### Debugging

```bash
# Run with verbose logging
MOON_LOG=debug moon run :build

# Show what inputs moon is tracking
moon query hash guardrail-server:build
```

---

## Reference

- **Official Docs**: https://moonrepo.dev/docs
- **Configuration**: https://moonrepo.dev/docs/config
- **CLI Reference**: https://moonrepo.dev/docs/commands
- **Examples**: https://github.com/moonrepo/examples

---

## Next Steps

1. Read `.moon/workspace.yml` to understand project structure
2. Check `.moon/tasks.yml` for global task definitions
3. Run `moon query projects` to see all available projects
4. Try `moon run :test --touched` to run tests efficiently
