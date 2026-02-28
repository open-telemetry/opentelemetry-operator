#!/usr/bin/env bash
# Verifies TLS profile injection on the stored collector CR.
# Args:
#   $1 - EXPECTED_MIN_VERSION: expected min_version value (e.g. "1.2" or "1.3")
#   $2 - EXPECT_CIPHERS: "true" if cipher_suites should be present, "false" if not
set -euo pipefail

EXPECTED_MIN_VERSION="${1:?expected min_version argument}"
EXPECT_CIPHERS="${2:?expected cipher expectation argument (true/false)}"

CONFIG=$(oc get otelcol tls-profile-test -n "$NAMESPACE" -o json | jq '.spec.config')

fail() {
  echo "FAIL: $1"
  echo "Full config:"
  echo "$CONFIG" | jq .
  exit 1
}

# --- Receivers with TLS ---

# OTLP gRPC (multi-protocol receiver, grpc protocol)
GRPC_MIN=$(echo "$CONFIG" | jq -r '.receivers.otlp.protocols.grpc.tls.min_version // empty')
[ "$GRPC_MIN" = "$EXPECTED_MIN_VERSION" ] || fail "otlp.grpc.tls.min_version: expected '$EXPECTED_MIN_VERSION', got '$GRPC_MIN'"

GRPC_CIPHERS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.grpc.tls.cipher_suites')
if [ "$EXPECT_CIPHERS" = "true" ]; then
  [ "$GRPC_CIPHERS" != "null" ] && [ "$GRPC_CIPHERS" != "[]" ] || fail "otlp.grpc.tls.cipher_suites: expected non-empty, got '$GRPC_CIPHERS'"
else
  [ "$GRPC_CIPHERS" = "null" ] || fail "otlp.grpc.tls.cipher_suites: expected null (TLS 1.3), got '$GRPC_CIPHERS'"
fi

# Zipkin (single-endpoint receiver)
ZIPKIN_MIN=$(echo "$CONFIG" | jq -r '.receivers.zipkin.tls.min_version // empty')
[ "$ZIPKIN_MIN" = "$EXPECTED_MIN_VERSION" ] || fail "zipkin.tls.min_version: expected '$EXPECTED_MIN_VERSION', got '$ZIPKIN_MIN'"

ZIPKIN_CIPHERS=$(echo "$CONFIG" | jq '.receivers.zipkin.tls.cipher_suites')
if [ "$EXPECT_CIPHERS" = "true" ]; then
  [ "$ZIPKIN_CIPHERS" != "null" ] && [ "$ZIPKIN_CIPHERS" != "[]" ] || fail "zipkin.tls.cipher_suites: expected non-empty, got '$ZIPKIN_CIPHERS'"
else
  [ "$ZIPKIN_CIPHERS" = "null" ] || fail "zipkin.tls.cipher_suites: expected null (TLS 1.3), got '$ZIPKIN_CIPHERS'"
fi

# --- Exporter with TLS ---

# Prometheus exporter
PROM_MIN=$(echo "$CONFIG" | jq -r '.exporters.prometheus.tls.min_version // empty')
[ "$PROM_MIN" = "$EXPECTED_MIN_VERSION" ] || fail "prometheus.tls.min_version: expected '$EXPECTED_MIN_VERSION', got '$PROM_MIN'"

PROM_CIPHERS=$(echo "$CONFIG" | jq '.exporters.prometheus.tls.cipher_suites')
if [ "$EXPECT_CIPHERS" = "true" ]; then
  [ "$PROM_CIPHERS" != "null" ] && [ "$PROM_CIPHERS" != "[]" ] || fail "prometheus.tls.cipher_suites: expected non-empty, got '$PROM_CIPHERS'"
else
  [ "$PROM_CIPHERS" = "null" ] || fail "prometheus.tls.cipher_suites: expected null (TLS 1.3), got '$PROM_CIPHERS'"
fi

# --- Components without TLS (no injection expected) ---

# OTLP HTTP (no tls block configured)
HTTP_TLS=$(echo "$CONFIG" | jq '.receivers.otlp.protocols.http.tls // empty')
[ -z "$HTTP_TLS" ] || [ "$HTTP_TLS" = "" ] || fail "otlp.http should not have tls block, got '$HTTP_TLS'"

# Debug exporter (no tls block configured)
DEBUG_TLS=$(echo "$CONFIG" | jq '.exporters.debug.tls // empty')
[ -z "$DEBUG_TLS" ] || [ "$DEBUG_TLS" = "" ] || fail "debug exporter should not have tls block, got '$DEBUG_TLS'"

echo "PASS: TLS profile verification (min_version=$EXPECTED_MIN_VERSION, ciphers=$EXPECT_CIPHERS)"
