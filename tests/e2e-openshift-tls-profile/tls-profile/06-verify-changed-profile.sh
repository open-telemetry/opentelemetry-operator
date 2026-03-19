#!/bin/bash
# Verifies changed TLS profile injection in the generated collector ConfigMap.
# TLS defaults are applied at reconciliation time (ConfigMap generation), not stored in the CR.
# Reads expected values from the tls-profile-expected ConfigMap.
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

EXPECTED_MIN=$(oc get configmap tls-profile-expected -n "$NAMESPACE" -o jsonpath='{.data.min_version}')
EXPECTED_CIPHERS=$(oc get configmap tls-profile-expected -n "$NAMESPACE" -o jsonpath='{.data.expect_ciphers}')
echo "Verifying profile: min_version=$EXPECTED_MIN, expect_ciphers=$EXPECTED_CIPHERS"

# Get the collector ConfigMap name (it includes a hash suffix)
CM_NAME=$(kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/component=opentelemetry-collector \
  -o jsonpath='{.items[0].metadata.name}')
[ -n "$CM_NAME" ] || fail "collector ConfigMap not found"

# Extract the collector.yaml from the ConfigMap and convert YAML to JSON
CONFIG=$(kubectl get configmap "$CM_NAME" -n "$NAMESPACE" -o jsonpath='{.data.collector\.yaml}' | yq -o json)

# --- Receivers with TLS ---

# OTLP gRPC
GRPC_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.grpc.tls.min_version // empty')
[ "$GRPC_MIN" = "$EXPECTED_MIN" ] || fail "otlp.grpc.tls.min_version: expected '$EXPECTED_MIN', got '$GRPC_MIN'"

GRPC_CIPHERS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.grpc.tls.cipher_suites')
if [ "$EXPECTED_CIPHERS" = "true" ]; then
  [ "$GRPC_CIPHERS" != "null" ] && [ "$GRPC_CIPHERS" != "[]" ] \
    || fail "otlp.grpc.tls.cipher_suites: expected non-empty, got '$GRPC_CIPHERS'"
else
  [ "$GRPC_CIPHERS" = "null" ] || fail "otlp.grpc.tls.cipher_suites: expected null (TLS 1.3), got '$GRPC_CIPHERS'"
fi

# Zipkin
ZIPKIN_MIN=$(echo "$CONFIG" | jq -r '.receivers.zipkin.tls.min_version // empty')
[ "$ZIPKIN_MIN" = "$EXPECTED_MIN" ] || fail "zipkin.tls.min_version: expected '$EXPECTED_MIN', got '$ZIPKIN_MIN'"

# Prometheus exporter
PROM_MIN=$(echo "$CONFIG" | jq -r '.exporters.prometheus.tls.min_version // empty')
[ "$PROM_MIN" = "$EXPECTED_MIN" ] || fail "prometheus.tls.min_version: expected '$EXPECTED_MIN', got '$PROM_MIN'"

# --- Components without TLS (no injection expected) ---
HTTP_TLS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.http.tls // empty')
[ -z "$HTTP_TLS" ] || [ "$HTTP_TLS" = "" ] \
  || fail "otlp.http should not have tls block, got '$HTTP_TLS'"

DEBUG_TLS=$(echo "$CONFIG" | jq '.exporters.debug.tls // empty')
[ -z "$DEBUG_TLS" ] || [ "$DEBUG_TLS" = "" ] \
  || fail "debug exporter should not have tls block, got '$DEBUG_TLS'"

echo "PASS: Changed profile verified on ConfigMap fields (min_version=$EXPECTED_MIN)"
