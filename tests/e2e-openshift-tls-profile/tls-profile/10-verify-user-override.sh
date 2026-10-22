#!/bin/bash
# Verifies that user-specified TLS values are preserved during ConfigMap generation.
# TLS defaults are applied at reconciliation time but should NOT overwrite user-specified values.
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# Get the collector ConfigMap name (it includes a hash suffix)
CM_NAME=$(kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/component=opentelemetry-collector \
  -o jsonpath='{.items[0].metadata.name}')
[ -n "$CM_NAME" ] || fail "collector ConfigMap not found"

# Extract the collector.yaml from the ConfigMap and convert YAML to JSON
CONFIG=$(kubectl get configmap "$CM_NAME" -n "$NAMESPACE" -o jsonpath='{.data.collector\.yaml}' | yq -o json)

# --- gRPC: user set min_version="1.3", did NOT set cipher_suites ---

GRPC_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.grpc.tls.min_version // empty')
[ "$GRPC_MIN" = "1.3" ] || fail "otlp.grpc.tls.min_version: expected '1.3' (user value), got '$GRPC_MIN'"

# Cipher suites: user didn't set them, so cluster value should be injected.
GRPC_CIPHERS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.grpc.tls.cipher_suites')
[ "$GRPC_CIPHERS" != "null" ] && [ "$GRPC_CIPHERS" != "[]" ] \
  || fail "otlp.grpc.tls.cipher_suites: expected cluster ciphers injected, got '$GRPC_CIPHERS'"

# Curve preferences: user set them to ["P384"], should be preserved.
GRPC_CURVES=$(echo "$CONFIG" | jq -c '.receivers.otlp.protocols.grpc.tls.curve_preferences')
[ "$GRPC_CURVES" = '["P384"]' ] \
  || fail "otlp.grpc.tls.curve_preferences: expected '[\"P384\"]' (user value), got '$GRPC_CURVES'"

# --- HTTP: user set cipher_suites, did NOT set min_version ---

HTTP_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.http.tls.min_version // empty')
[ "$HTTP_MIN" = "1.2" ] || fail "otlp.http.tls.min_version: expected '1.2' (cluster value), got '$HTTP_MIN'"

# Cipher suites: user set them to ["TLS_AES_128_GCM_SHA256"], should be preserved
HTTP_CIPHERS=$(echo "$CONFIG" | jq -c '.receivers.otlp.protocols.http.tls.cipher_suites')
[ "$HTTP_CIPHERS" = '["TLS_AES_128_GCM_SHA256"]' ] \
  || fail "otlp.http.tls.cipher_suites: expected '[\"TLS_AES_128_GCM_SHA256\"]' (user value), got '$HTTP_CIPHERS'"

# Curve preferences: user didn't set them, so cluster groups should be injected.
HTTP_CURVES=$(echo "$CONFIG" | jq -c '.receivers.otlp.protocols.http.tls.curve_preferences')
[ "$HTTP_CURVES" = '["X25519MLKEM768","X25519","P256","P384"]' ] \
  || fail "otlp.http.tls.curve_preferences: expected cluster groups injected, got '$HTTP_CURVES'"

echo "PASS: User TLS override verification"
