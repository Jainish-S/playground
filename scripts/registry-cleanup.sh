#!/bin/bash
# Registry Cleanup Script for OCI Container Registry
#
# This script manages OCI Container Registry images:
# - Deletes images with "unknown" tags (from manifest lists)
# - Keeps only the latest N images per repository
# - Configures retention policies
#
# Prerequisites:
#   - OCI CLI configured (oci setup config)
#   - Appropriate permissions on Container Registry
#
# Usage:
#   ./scripts/registry-cleanup.sh [--dry-run]
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRY_RUN=false
KEEP_LATEST=2  # Keep latest N images per repository (excluding :latest tag)

if [[ "$1" == "--dry-run" ]]; then
    DRY_RUN=true
    echo "🔍 DRY RUN MODE - No changes will be made"
    echo ""
fi

# Get compartment OCID from environment or OCI CLI config
# Note: OCI repos are typically in root tenancy compartment
COMPARTMENT_ID="${OCI_COMPARTMENT_OCID:-}"

if [ -z "$COMPARTMENT_ID" ]; then
    # Get tenancy OCID from OCI CLI config (repos are usually in root compartment)
    COMPARTMENT_ID=$(grep tenancy ~/.oci/config 2>/dev/null | head -1 | cut -d'=' -f2)
fi

if [ -z "$COMPARTMENT_ID" ]; then
    echo "❌ Error: Could not get compartment OCID"
    echo ""
    echo "Set OCI_COMPARTMENT_OCID environment variable:"
    echo "  export OCI_COMPARTMENT_OCID=ocid1.tenancy.oc1..aaaa..."
    echo ""
    echo "Or ensure OCI CLI is configured: oci setup config"
    exit 1
fi

echo "=============================================="
echo "OCI Container Registry Cleanup"
echo "=============================================="
echo ""
echo "📦 Compartment: ${COMPARTMENT_ID:0:20}..."
echo "🔢 Keep latest: $KEEP_LATEST images per repo"
echo ""

# List all repositories
echo "📋 Fetching repositories..."
REPOS=$(oci artifacts container repository list \
    --compartment-id "$COMPARTMENT_ID" \
    --query "data.items[].{name: \"display-name\", id: id}" \
    --output json 2>/dev/null || echo "[]")

REPO_COUNT=$(echo "$REPOS" | jq length)
echo "   Found $REPO_COUNT repositories"
echo ""

# Process each repository
echo "$REPOS" | jq -c '.[]' | while read -r repo; do
    REPO_NAME=$(echo "$repo" | jq -r '.name')
    REPO_ID=$(echo "$repo" | jq -r '.id')
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📁 Repository: $REPO_NAME"
    echo ""
    
    # Get all images in this repository
    IMAGES=$(oci artifacts container image list \
        --compartment-id "$COMPARTMENT_ID" \
        --repository-name "$REPO_NAME" \
        --query "data.items[].{id: id, digest: digest, version: version, timeCreated: \"time-created\"}" \
        --output json 2>/dev/null || echo "[]")
    
    IMAGE_COUNT=$(echo "$IMAGES" | jq length)
    echo "   Images: $IMAGE_COUNT total"
    
    # Count unknown/untagged images
    UNKNOWN_IMAGES=$(echo "$IMAGES" | jq '[.[] | select(.version == null or .version == "" or (.version | startswith("sha256:")))]')
    UNKNOWN_COUNT=$(echo "$UNKNOWN_IMAGES" | jq length)
    
    if [ "$UNKNOWN_COUNT" -gt 0 ]; then
        echo "   ⚠️  Unknown/untagged images: $UNKNOWN_COUNT"
        
        if [ "$DRY_RUN" = true ]; then
            echo "   [DRY RUN] Would delete $UNKNOWN_COUNT unknown images"
        else
            echo "   🗑️  Deleting unknown images..."
            echo "$UNKNOWN_IMAGES" | jq -r '.[].id' | while read -r img_id; do
                oci artifacts container image delete --image-id "$img_id" --force 2>/dev/null && \
                    echo "      Deleted: ${img_id:0:30}..." || \
                    echo "      Failed: ${img_id:0:30}..."
            done
        fi
    else
        echo "   ✅ No unknown images"
    fi
    
    # Get tagged images sorted by creation time
    TAGGED_IMAGES=$(echo "$IMAGES" | jq '[.[] | select(.version != null and .version != "" and (.version | startswith("sha256:") | not))] | sort_by(.timeCreated) | reverse')
    TAGGED_COUNT=$(echo "$TAGGED_IMAGES" | jq 'length')
    
    # Find images to delete (older than KEEP_LATEST, excluding :latest)
    DELETE_IMAGES=$(echo "$TAGGED_IMAGES" | jq "[.[$KEEP_LATEST:] | .[] | select(.version != \"latest\")]")
    DELETE_COUNT=$(echo "$DELETE_IMAGES" | jq 'length')
    
    if [ "$DELETE_COUNT" -gt 0 ]; then
        echo "   📊 Keeping newest $KEEP_LATEST tagged images"
        echo "   🗑️  Old images to delete: $DELETE_COUNT"
        
        if [ "$DRY_RUN" = true ]; then
            echo "   [DRY RUN] Would delete:"
            echo "$DELETE_IMAGES" | jq -r '.[].version' | while read -r ver; do
                echo "      - $ver"
            done
        else
            echo "$DELETE_IMAGES" | jq -r '.[].id' | while read -r img_id; do
                oci artifacts container image delete --image-id "$img_id" --force 2>/dev/null && \
                    echo "      Deleted: ${img_id:0:30}..." || \
                    echo "      Failed: ${img_id:0:30}..."
            done
        fi
    fi
    
    echo ""
done

echo "=============================================="
echo "✅ Cleanup complete!"
echo "=============================================="
echo ""
echo "💡 Tips:"
echo "   - Run with --dry-run first to preview changes"
echo "   - Configure retention policy in OCI Console for automatic cleanup"
echo "   - Use semantic version tags (v1.0.0) instead of :latest for production"
echo ""

# Node image cleanup info
echo "📌 Kubernetes Node Image Cleanup:"
echo "   Kubelet automatically garbage collects images when disk pressure occurs."
echo "   Default settings:"
echo "     - imageGCHighThresholdPercent: 85%"
echo "     - imageGCLowThresholdPercent: 80%"
echo ""
echo "   To customize, add to kubelet config:"
echo '     imageGCHighThresholdPercent: 70'
echo '     imageGCLowThresholdPercent: 50'
echo ""
