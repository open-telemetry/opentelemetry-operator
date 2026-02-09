#!/usr/bin/env bash
# Verifies that user-specified TLS values are preserved by the webhook.
# The webhook should NOT overwrite values the user explicitly set.
set -euo pipefail

CONFIG=$(oc get otelcol tls-profile-test -n "$NAMESPACE" -o json | jq '.spec.config')

fail() {
  echo "FAIL: $1"
  echo "Full config:"
  echo "$CONFIG" | jq .
  exit 1
}

# --- gRPC: user set min_version="1.3", did NOT set cipher_suites ---

GRPC_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.grpc.tls.min_version // empty')
[ "$GRPC_MIN" = "1.3" ] || fail "otlp.grpc.tls.min_version: expected '1.3' (user value), got '$GRPC_MIN'"

# Cipher suites: user didn't set them, so cluster value should be injected.
# With min_version=1.3 (user-set), the cluster profile is Intermediate (TLS 1.2 + ciphers).
# The webhook injects cipher_suites since the user didn't set them and the cluster profile has them.
GRPC_CIPHERS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.grpc.tls.cipher_suites')
[ "$GRPC_CIPHERS" != "null" ] && [ "$GRPC_CIPHERS" != "[]" ] || fail "otlp.grpc.tls.cipher_suites: expected cluster ciphers injected, got '$GRPC_CIPHERS'"

# --- HTTP: user set cipher_suites, did NOT set min_version ---

HTTP_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.http.tls.min_version // empty')
[ "$HTTP_MIN" = "1.2" ] || fail "otlp.http.tls.min_version: expected '1.2' (cluster value), got '$HTTP_MIN'"

# Cipher suites: user set them to ["TLS_AES_128_GCM_SHA256"], should be preserved
HTTP_CIPHERS=$(echo "$CONFIG" | jq -c '.receivers.otlp.protocols.http.tls.cipher_suites')
[ "$HTTP_CIPHERS" = '["TLS_AES_128_GCM_SHA256"]' ] || fail "otlp.http.tls.cipher_suites: expected '[\"TLS_AES_128_GCM_SHA256\"]' (user value), got '$HTTP_CIPHERS'"

echo "PASS: User TLS override verification"
