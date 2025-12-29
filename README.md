# Welcome to my Playground

A **polyglot monorepo** for learning, experimentation, and building production-grade infrastructure.

## Quick Start

See [CLAUDE.md](./CLAUDE.md) for project vision and development guidelines.

## Infrastructure

OCI Kubernetes infrastructure with zero-cost free tier optimization.

```bash
cd infra/terraform/environments/dev
terraform init && terraform apply
```

Full guide: [infra/RUNBOOK.md](./infra/RUNBOOK.md)

### Structure

```
infra/
├── terraform/           # Infrastructure as Code
│   ├── modules/         # Reusable modules (network, oke, registry, storage, twingate)
│   └── environments/    # Environment configs
└── k8s/                 # Kubernetes manifests
    ├── base/            # Resource quotas, namespaces
    ├── contour/         # Contour ingress (Envoy-based)
    ├── cert-manager/    # TLS certificates
    ├── twingate/        # Zero-trust access
    └── observability/   # Prometheus, Grafana, exporters
```
