#!/bin/bash
# Update K8s manifests with OCI registry URL
#
# Updates all image references in guardrails K8s manifests
# to use the correct OCI registry path.
#
# Usage:
#   ./scripts/update-manifests.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="/Users/jainish/os/k8s-infra/infra"
K8S_DIR="$PROJECT_ROOT/infra/k8s/guardrails"

echo "==========================================="
echo "Update K8s Manifests with OCI Registry"
echo "==========================================="
echo ""

# Get registry info from Terraform
cd "$TF_DIR"
REGISTRY_REGION=$(terraform output -raw guardrail_registry_region 2>/dev/null || echo "")
REGISTRY_NAMESPACE=$(terraform output -raw guardrail_registry_namespace 2>/dev/null || echo "")

if [ -z "$REGISTRY_REGION" ] || [ -z "$REGISTRY_NAMESPACE" ]; then
    echo "❌ Error: Registry outputs not found"
    exit 1
fi

REGISTRY="${REGISTRY_REGION}.ocir.io/${REGISTRY_NAMESPACE}"
echo "📦 Using registry: $REGISTRY"
echo ""

# Backup and update guardrail-server deployment
echo "Updating guardrail-server deployment..."
sed -i.bak "s|image: guardrail-server:latest|image: ${REGISTRY}/guardrail/guardrail-server:latest|g" \
    "$K8S_DIR/guardrail-server/deployment.yaml"

# Update model deployments
echo "Updating model deployments..."
for model in prompt-guard pii-detect hate-detect content-class; do
    sed -i.bak "s|image: model-${model}:latest|image: ${REGISTRY}/guardrail/model-${model}:latest|g" \
        "$K8S_DIR/models/deployments.yaml"
done

# Clean up backup files
find "$K8S_DIR" -name "*.bak" -delete

echo ""
echo "✅ Manifests updated!"
echo ""
echo "Updated files:"
echo "  - $K8S_DIR/guardrail-server/deployment.yaml"
echo "  - $K8S_DIR/models/deployments.yaml"
echo ""
echo "Next: Deploy to Kubernetes"
echo "Run: ./scripts/deploy.sh"
