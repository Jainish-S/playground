#!/bin/bash
# Deploy Guardrails Platform to Kubernetes
#
# Deploys all guardrails components in the correct order:
# 1. Namespace and resource quotas
# 2. ConfigMaps and Secrets
# 3. PostgreSQL and Redis databases
# 4. Model services
# 5. Guardrail server
# 6. Ingress and monitoring
#
# Usage:
#   ./scripts/deploy.sh
#
# Options:
#   --dry-run    Show what would be deployed without applying

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
K8S_DIR="$PROJECT_ROOT/infra/k8s/guardrails"

DRY_RUN=""
if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN="--dry-run=client"
    echo "🔍 DRY RUN MODE - No changes will be applied"
    echo ""
fi

echo "==========================================="
echo "Guardrails Platform - Kubernetes Deployment"
echo "==========================================="
echo ""

# Step 1: Namespace
echo "📁 Step 1/6: Creating namespace..."
kubectl apply -f "$K8S_DIR/namespace.yaml" $DRY_RUN
echo "  ✅ Namespace created"
echo ""

# Step 2: ConfigMaps and Secrets
echo "🔐 Step 2/6: Applying configs and secrets..."
kubectl apply -f "$K8S_DIR/configs/" $DRY_RUN
echo "  ✅ Configs applied"
echo ""

# Step 3: Databases
echo "💾 Step 3/6: Deploying databases..."
kubectl apply -f "$K8S_DIR/databases.yaml" $DRY_RUN
if [[ -z "$DRY_RUN" ]]; then
    echo "  ⏳ Waiting for PostgreSQL..."
    kubectl wait --for=condition=ready pod -l app=postgres -n guardrails-platform --timeout=120s || true
    echo "  ⏳ Waiting for Redis..."
    kubectl wait --for=condition=ready pod -l app=redis -n guardrails-platform --timeout=120s || true
fi
echo "  ✅ Databases deployed"
echo ""

# Step 4: Model services
echo "🤖 Step 4/6: Deploying model services..."
kubectl apply -f "$K8S_DIR/models/" $DRY_RUN
if [[ -z "$DRY_RUN" ]]; then
    echo "  ⏳ Waiting for models..."
    kubectl wait --for=condition=ready pod -l type=ml-model -n guardrails-platform --timeout=180s || true
fi
echo "  ✅ Models deployed"
echo ""

# Step 5: Guardrail server
echo "🛡️ Step 5/6: Deploying guardrail server..."
kubectl apply -f "$K8S_DIR/guardrail-server/" $DRY_RUN
if [[ -z "$DRY_RUN" ]]; then
    echo "  ⏳ Waiting for guardrail server..."
    kubectl wait --for=condition=ready pod -l app=guardrail-server -n guardrails-platform --timeout=180s || true
fi
echo "  ✅ Guardrail server deployed"
echo ""

# Step 6: Ingress and monitoring
echo "🌐 Step 6/6: Deploying ingress and monitoring..."
kubectl apply -f "$K8S_DIR/ingress.yaml" $DRY_RUN
kubectl apply -f "$K8S_DIR/monitoring.yaml" $DRY_RUN 2>/dev/null || echo "  ⚠️ ServiceMonitor CRDs not found, skipping"
echo "  ✅ Ingress and monitoring deployed"
echo ""

echo "==========================================="
echo "✅ Deployment complete!"
echo "==========================================="
echo ""

if [[ -z "$DRY_RUN" ]]; then
    echo "📊 Status:"
    kubectl get pods -n guardrails-platform
    echo ""
    echo "🔗 Access:"
    echo "  - Health: kubectl port-forward svc/guardrail-server 8000:8000 -n guardrails-platform"
    echo "  - Then: curl http://localhost:8000/v1/health"
    echo ""
    echo "📈 Monitoring:"
    echo "  - Grafana dashboard: 'Guardrails Platform'"
fi
