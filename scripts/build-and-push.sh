#!/bin/bash
# Build and Push Guardrails Images to OCI Registry
#
# This script builds all guardrails Docker images and pushes them to OCI.
# It reads the registry info from k8s-infra Terraform outputs.
#
# Usage:
#   ./scripts/build-and-push.sh
#
# Prerequisites:
#   - Docker running
#   - Terraform applied in k8s-infra/infra

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TF_DIR="/Users/jainish/os/k8s-infra/infra"

echo "==========================================="
echo "Guardrails Platform - Build & Push to OCI"
echo "==========================================="
echo ""

# Check if Terraform outputs exist
cd "$TF_DIR"
REGISTRY_REGION=$(terraform output -raw guardrail_registry_region 2>/dev/null || echo "")
REGISTRY_NAMESPACE=$(terraform output -raw guardrail_registry_namespace 2>/dev/null || echo "")

if [ -z "$REGISTRY_REGION" ] || [ -z "$REGISTRY_NAMESPACE" ]; then
    echo "❌ Error: Registry outputs not found in Terraform"
    echo "   Run: cd $TF_DIR && terraform apply"
    echo ""
    echo "   Or set manually:"
    echo "   export OCI_REGION=bom"
    echo "   export OCI_NAMESPACE=your-namespace"
    exit 1
fi

# Allow override from environment
OCI_REGION="${OCI_REGION:-$REGISTRY_REGION}"
# Convert to lowercase to avoid docker login mismatch
OCI_REGION=$(echo "$OCI_REGION" | tr '[:upper:]' '[:lower:]')
OCI_NAMESPACE="${OCI_NAMESPACE:-$REGISTRY_NAMESPACE}"
REGISTRY="${OCI_REGION}.ocir.io/${OCI_NAMESPACE}"

echo "📦 Registry: $REGISTRY"
echo ""

cd "$PROJECT_ROOT"

# Build images
# Format: "image-name:dockerfile:context"
# If context is omitted, defaults to project root (.)
IMAGES=(
    "guardrail-server:apps/guardrail-server/Dockerfile:."
    "model-prompt-guard:apps/model-prompt-guard/Dockerfile:."
    "model-pii-detect:apps/model-pii-detect/Dockerfile:."
    "model-hate-detect:apps/model-hate-detect/Dockerfile:."
    "model-content-class:apps/model-content-class/Dockerfile:."
    # "loadtest:tools/loadtest/Dockerfile:tools/loadtest"
)

echo "🔨 Building images..."
echo ""

# Get current git commit SHA for versioning
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "local")
echo "📌 Git SHA: $GIT_SHA"
echo ""

for img_entry in "${IMAGES[@]}"; do
    IFS=':' read -r img_name dockerfile context <<< "$img_entry"
    # Default to project root if context not specified
    context="${context:-.}"
    echo "Building $img_name..."
    echo "  Context: $context"
    echo "  Dockerfile: $dockerfile"
    # Use --provenance=false to avoid creating manifest lists with "unknown" tags
    # Also build with explicit platform to avoid multi-arch issues
    docker build --provenance=false --no-cache -f "$dockerfile" -t "$img_name:latest" "$context"
    docker tag "$img_name:latest" "${REGISTRY}/guardrail/${img_name}:latest"
    docker tag "$img_name:latest" "${REGISTRY}/guardrail/${img_name}:${GIT_SHA}"
    echo "  ✅ $img_name built and tagged (latest + ${GIT_SHA})"
done

echo ""
echo "🚀 Pushing images to OCI..."
echo ""

for img_entry in "${IMAGES[@]}"; do
    IFS=':' read -r img_name _ <<< "$img_entry"
    echo "Pushing ${REGISTRY}/guardrail/${img_name}..."
    docker push "${REGISTRY}/guardrail/${img_name}:latest"
    docker push "${REGISTRY}/guardrail/${img_name}:${GIT_SHA}"
    echo "  ✅ Pushed (latest + ${GIT_SHA})"
done

echo ""
echo "==========================================="
echo "✅ All images built and pushed!"
echo "==========================================="
echo ""
echo "Images available at:"
for img_entry in "${IMAGES[@]}"; do
    IFS=':' read -r img_name _ <<< "$img_entry"
    echo "  ${REGISTRY}/guardrail/${img_name}:latest"
done
echo ""
echo "Next: Update K8s manifests with registry URL"
echo "Run: ./scripts/update-manifests.sh"
