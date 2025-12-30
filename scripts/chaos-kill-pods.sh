#!/usr/bin/env bash
#
# Simple chaos script: randomly kills pods to test graceful shutdown
# Run this while your load test is running

set -euo pipefail

NAMESPACE="${1:-guardrails-platform}"
INTERVAL="${2:-10}"  # Kill a pod every N seconds

echo "Chaos Pod Killer"
echo "Namespace: $NAMESPACE"
echo "Kill interval: ${INTERVAL}s"
echo "Press Ctrl+C to stop"
echo ""

# Deployments to target
DEPLOYMENTS=(
    "model-prompt-guard"
    "model-pii-detect"
    "model-hate-detect"
    "model-content-class"
    "guardrail-server"
)

while true; do
    # Pick random deployment
    DEPLOYMENT="${DEPLOYMENTS[$RANDOM % ${#DEPLOYMENTS[@]}]}"

    # Get random pod from that deployment
    POD=$(kubectl get pods -n "$NAMESPACE" -l "app=$DEPLOYMENT" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

    if [ -n "$POD" ]; then
        echo "[$(date +%H:%M:%S)] Killing pod: $POD (deployment: $DEPLOYMENT)"
        kubectl delete pod -n "$NAMESPACE" "$POD" --grace-period=1 &
    else
        echo "[$(date +%H:%M:%S)] No pods found for $DEPLOYMENT"
    fi

    sleep "$INTERVAL"
done
