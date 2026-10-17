#!/bin/bash
set -euo pipefail

# Recreating the Pod retries admission until the operator logs the missing base.
for attempt in $(seq 1 30); do
  matching_logs=$(kubectl logs -n opentelemetry-operator-system \
    deployment/opentelemetry-operator-controller-manager --since=2m 2>/dev/null \
    | grep -F "$MISSING_BASE_NAME" || true)
  if grep -Fq "$TEST_NAMESPACE" <<<"$matching_logs"; then
    echo "Operator log contains missing base $TEST_NAMESPACE/$MISSING_BASE_NAME"
    exit 0
  fi

  if [ "$attempt" -eq 30 ]; then
    echo "No Operator log mentions $TEST_NAMESPACE/$MISSING_BASE_NAME after $attempt attempts"
    exit 1
  fi

  pod=$(kubectl get pods -n "$TEST_NAMESPACE" -l app=base-ref-missing \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  if [ -n "$pod" ]; then
    kubectl delete pod "$pod" -n "$TEST_NAMESPACE" --wait=true --timeout=30s
  fi
  sleep 2
done
