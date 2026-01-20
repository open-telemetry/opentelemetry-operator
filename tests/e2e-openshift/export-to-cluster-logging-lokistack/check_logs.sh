#!/usr/bin/env bash

TOKEN=$(oc -n openshift-logging create token otel-collector-deployment)
LOKI_URL=$(oc -n openshift-logging get route logging-loki -o json | jq '.spec.host' -r)

while true; do
  # Fetch logs
  RAW_OUTPUT=$(logcli -o raw --tls-skip-verify \
  --bearer-token="${TOKEN}" \
  --addr "https://${LOKI_URL}/api/logs/v1/application" query '{log_type="application"}')

  # Extract the part of the output containing the common labels
  COMMON_LABELS=$(echo "$RAW_OUTPUT" | grep "Common labels:")

  # Log output for the actual log messages
  LOG_OUTPUT=$(echo "$RAW_OUTPUT" | grep -v "Common labels:")

  # Check if specific log messages exist
  if echo "$COMMON_LABELS" | grep -q 'app="server"' && \
    echo "$COMMON_LABELS" | grep -q 'k8s_container_name="telemetrygen"' && \
    echo "$COMMON_LABELS" | grep -q 'k8s_namespace_name="chainsaw-incllogs"' && \
    echo "$COMMON_LABELS" | grep -q 'kubernetes_container_name="telemetrygen"' && \
    echo "$COMMON_LABELS" | grep -q 'kubernetes_namespace_name="chainsaw-incllogs"' && \
    echo "$LOG_OUTPUT" | grep -q "the message"; then
    echo "Logs found:"
    echo "$COMMON_LABELS"
    break
  else
    echo "Logs not found. Continuing to check..."
    sleep 5
  fi
done
