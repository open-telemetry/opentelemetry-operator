#!/bin/bash
# Verifies Intermediate TLS profile injection in the generated collector ConfigMap.
# TLS defaults are applied at reconciliation time (ConfigMap generation), not stored in the CR.
# Checks S1 (injection with TLS block), S2 (no injection without TLS block),
# S3 (multi-protocol), S4 (single-endpoint), S5 (exporter TLS).
set -euo pipefail

fail() { echo "FAIL: $1"; exit 1; }

# Get the collector ConfigMap name (it includes a hash suffix)
CM_NAME=$(kubectl get configmap -n "$NAMESPACE" -l app.kubernetes.io/component=opentelemetry-collector \
  -o jsonpath='{.items[0].metadata.name}')
[ -n "$CM_NAME" ] || fail "collector ConfigMap not found"

# Extract the collector.yaml from the ConfigMap and convert YAML to JSON
CONFIG=$(kubectl get configmap "$CM_NAME" -n "$NAMESPACE" -o jsonpath='{.data.collector\.yaml}' | yq -o json)

# --- Receivers with TLS (injection expected) ---

# S1+S3: OTLP gRPC (multi-protocol receiver, grpc protocol)
GRPC_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.grpc.tls.min_version // empty')
[ "$GRPC_MIN" = "1.2" ] || fail "otlp.grpc.tls.min_version: expected '1.2', got '$GRPC_MIN'"
GRPC_CIPHERS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.grpc.tls.cipher_suites')
[ "$GRPC_CIPHERS" != "null" ] && [ "$GRPC_CIPHERS" != "[]" ] \
  || fail "otlp.grpc.tls.cipher_suites: expected non-empty, got '$GRPC_CIPHERS'"

# S4: Zipkin (single-endpoint receiver)
ZIPKIN_MIN=$(echo "$CONFIG" | jq -r '.receivers.zipkin.tls.min_version // empty')
[ "$ZIPKIN_MIN" = "1.2" ] || fail "zipkin.tls.min_version: expected '1.2', got '$ZIPKIN_MIN'"
ZIPKIN_CIPHERS=$(echo "$CONFIG" | jq '.receivers.zipkin.tls.cipher_suites')
[ "$ZIPKIN_CIPHERS" != "null" ] && [ "$ZIPKIN_CIPHERS" != "[]" ] \
  || fail "zipkin.tls.cipher_suites: expected non-empty, got '$ZIPKIN_CIPHERS'"

# S5: Prometheus exporter
PROM_MIN=$(echo "$CONFIG" | jq -r '.exporters.prometheus.tls.min_version // empty')
[ "$PROM_MIN" = "1.2" ] || fail "prometheus.tls.min_version: expected '1.2', got '$PROM_MIN'"
PROM_CIPHERS=$(echo "$CONFIG" | jq '.exporters.prometheus.tls.cipher_suites')
[ "$PROM_CIPHERS" != "null" ] && [ "$PROM_CIPHERS" != "[]" ] \
  || fail "prometheus.tls.cipher_suites: expected non-empty, got '$PROM_CIPHERS'"

# --- Components without TLS (S2: no injection expected) ---

# OTLP HTTP (no tls block configured)
HTTP_TLS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.http.tls // empty')
[ -z "$HTTP_TLS" ] || [ "$HTTP_TLS" = "" ] \
  || fail "otlp.http should not have tls block, got '$HTTP_TLS'"

# Debug exporter (no tls block configured)
DEBUG_TLS=$(echo "$CONFIG" | jq '.exporters.debug.tls // empty')
[ -z "$DEBUG_TLS" ] || [ "$DEBUG_TLS" = "" ] \
  || fail "debug exporter should not have tls block, got '$DEBUG_TLS'"

echo "PASS: Intermediate profile verified on ConfigMap fields"
