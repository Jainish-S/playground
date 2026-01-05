#!/bin/bash
# Build and push all Go-based guardrail services to OCI Registry
# Usage: ./scripts/build-and-push-go.sh [--dry-run]

set -e

REGISTRY="bom.ocir.io/bm96q5bq36zw/guardrail"
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "latest")
DRY_RUN=false

if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN=true
    echo "=== DRY RUN MODE ==="
fi

# Services to build (Go-based)
SERVICES=(
    "guardrail-server-go:apps/guardrail-server-go"
    "model-prompt-guard-go:apps/model-prompt-guard-go"
    "model-pii-detect-go:apps/model-pii-detect-go"
    "model-hate-detect-go:apps/model-hate-detect-go"
    "model-content-class-go:apps/model-content-class-go"
)

echo "Building Go services with tag: $GIT_SHA"
echo "Registry: $REGISTRY"
echo ""

for entry in "${SERVICES[@]}"; do
    NAME="${entry%%:*}"
    PATH_DIR="${entry##*:}"
    
    echo ">>> Building $NAME from $PATH_DIR"
    
    IMAGE_LATEST="$REGISTRY/$NAME:latest"
    IMAGE_SHA="$REGISTRY/$NAME:$GIT_SHA"
    
    if [ "$DRY_RUN" = true ]; then
        echo "  [DRY RUN] Would build: $PATH_DIR/Dockerfile"
        echo "  [DRY RUN] Would tag as: $IMAGE_LATEST, $IMAGE_SHA"
    else
        # Build from repo root with build context including go.work and packages
        docker build \
            --provenance=false \
            -f "$PATH_DIR/Dockerfile" \
            -t "$IMAGE_LATEST" \
            -t "$IMAGE_SHA" \
            .
        
        echo "  Pushing $IMAGE_LATEST"
        docker push "$IMAGE_LATEST"
        
        echo "  Pushing $IMAGE_SHA"
        docker push "$IMAGE_SHA"
    fi
    
    echo ""
done

echo "=== Build complete ==="
echo "Images tagged with: latest, $GIT_SHA"
