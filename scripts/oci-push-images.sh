#!/bin/bash
# OCI Container Registry - Push Guardrails Images
#
# Builds and pushes all guardrails Docker images to OCI Container Registry.
# Uses OCI CLI for authentication and standard naming conventions.
#
# Usage:
#   ./scripts/oci-push-images.sh [TAG]
#
# Examples:
#   ./scripts/oci-push-images.sh           # Tag with 'latest' + git SHA
#   ./scripts/oci-push-images.sh v1.0.0    # Tag with 'v1.0.0' + git SHA
#
# Prerequisites:
#   - Docker running
#   - OCI CLI installed and configured (`oci session authenticate`)
#   - Auth token for docker login (generate via OCI Console)
#
# Registry URL format:
#   <region>.ocir.io/<namespace>/guardrail/<service>:<tag>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==========================================="
echo "Guardrails Platform - Build & Push to OCI"
echo "==========================================="
echo ""

# Configuration
OCI_REGION="${OCI_REGION:-bom}"
OCI_NAMESPACE="${OCI_NAMESPACE:-bm96q5bq36zw}"
REGISTRY="${OCI_REGION}.ocir.io/${OCI_NAMESPACE}/guardrail"

# Tag configuration
VERSION_TAG="${1:-latest}"
GIT_SHA=$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "📦 Registry: ${REGISTRY}"
echo "🏷️  Tags: ${VERSION_TAG}, ${GIT_SHA}"
echo ""

# Services to build and push
SERVICES=(
  "guardrail-server:apps/guardrail-server/Dockerfile"
  "model-prompt-guard:apps/model-prompt-guard/Dockerfile"
  "model-pii-detect:apps/model-pii-detect/Dockerfile"
  "model-hate-detect:apps/model-hate-detect/Dockerfile"
  "model-content-class:apps/model-content-class/Dockerfile"
)

# Check docker login
echo "🔐 Checking docker login to OCI..."
if ! docker info 2>/dev/null | grep -q "Username"; then
  echo ""
  echo "⚠️  Not logged into docker registry. Login with:"
  echo "   docker login ${OCI_REGION}.ocir.io"
  echo "   Username: ${OCI_NAMESPACE}/oracleidentitycloudservice/<your-email>"
  echo "   Password: <your-auth-token>"
  echo ""
  read -p "Press Enter after logging in, or Ctrl+C to abort..."
fi

cd "$PROJECT_ROOT"

echo ""
echo "🔨 Building images..."
echo ""

for entry in "${SERVICES[@]}"; do
  IFS=':' read -r service dockerfile <<< "$entry"
  
  echo "Building ${service}..."
  docker build -f "$dockerfile" -t "${service}:latest" . --quiet
  
  # Tag with version and git SHA
  docker tag "${service}:latest" "${REGISTRY}/${service}:${VERSION_TAG}"
  docker tag "${service}:latest" "${REGISTRY}/${service}:${GIT_SHA}"
  
  echo "  ✅ ${service} built and tagged"
done

echo ""
echo "🚀 Pushing images to OCI..."
echo ""

for entry in "${SERVICES[@]}"; do
  IFS=':' read -r service _ <<< "$entry"
  
  echo "Pushing ${service}..."
  docker push "${REGISTRY}/${service}:${VERSION_TAG}" --quiet
  docker push "${REGISTRY}/${service}:${GIT_SHA}" --quiet
  echo "  ✅ Pushed ${REGISTRY}/${service}:{${VERSION_TAG},${GIT_SHA}}"
done

echo ""
echo "==========================================="
echo "✅ All images built and pushed!"
echo "==========================================="
echo ""
echo "Images pushed:"
for entry in "${SERVICES[@]}"; do
  IFS=':' read -r service _ <<< "$entry"
  echo "  ${REGISTRY}/${service}:${VERSION_TAG}"
done
echo ""
echo "Next steps:"
echo "  1. Update K8s manifests with new image tags"
echo "  2. Deploy: ./scripts/deploy.sh"
