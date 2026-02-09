#!/usr/bin/env bash
# Functional TLS compliance test using tls-scanner.
# Scans the collector's gRPC endpoint and verifies TLS compliance
# against the active cluster TLS security profile.
# tls-scanner uses nmap internally, which requires NET_RAW capability.
# Args:
#   $1 - EXPECTED_MIN_VERSION: "1.2" or "1.3" (for logging)
#   $2 - TEST_REJECTION: unused (kept for interface compatibility)
set -euo pipefail

EXPECTED_MIN_VERSION="${1:?expected min_version argument}"

SERVICE="tls-profile-test-collector"
GRPC_PORT="4317"
IMAGE="quay.io/rhn_support_ikanse/tls-scanner:latest"

fail() {
  echo "FAIL: $1"
  exit 1
}

cleanup() {
  oc delete pod tls-scanner -n "$NAMESPACE" --ignore-not-found 2>/dev/null || true
  oc delete sa tls-scanner-sa -n "$NAMESPACE" --ignore-not-found 2>/dev/null || true
}
trap cleanup EXIT

# Create a ServiceAccount with privileged SCC for nmap raw socket access
oc create sa tls-scanner-sa -n "$NAMESPACE"
oc adm policy add-scc-to-user privileged -z tls-scanner-sa -n "$NAMESPACE"

# Run tls-scanner pod with the privileged SA
oc run tls-scanner -n "$NAMESPACE" \
  --image="$IMAGE" \
  --restart=Never \
  --overrides='{
    "spec": {
      "serviceAccountName": "tls-scanner-sa",
      "containers": [{
        "name": "tls-scanner",
        "image": "'"$IMAGE"'",
        "command": ["sleep", "300"],
        "securityContext": {
          "privileged": true
        }
      }]
    }
  }'
oc wait --for=condition=Ready pod/tls-scanner -n "$NAMESPACE" --timeout=120s

# Scan the collector gRPC endpoint for TLS compliance
echo "Scanning $SERVICE:$GRPC_PORT (expected min_version=$EXPECTED_MIN_VERSION)..."
oc exec tls-scanner -n "$NAMESPACE" -- \
  tls-scanner -host "$SERVICE" -port "$GRPC_PORT" \
  || fail "tls-scanner compliance check failed for $SERVICE:$GRPC_PORT"

echo "PASS: Functional TLS verification (min_version=$EXPECTED_MIN_VERSION)"
