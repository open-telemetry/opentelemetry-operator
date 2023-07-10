#!/bin/bash
JAEGER_URL=$(oc -n kuttl-otlp-metrics get route jaeger-allinone -o json | jq '.spec.host' -r)
SERVICE_NAME="telemetrygen"

trace_exists=$(curl -ksSL "https://$JAEGER_URL/api/traces?service=$SERVICE_NAME&limit=1" | jq -r '.data | length')

if [[ $trace_exists -gt 0 ]]; then
  echo "Traces for $SERVICE_NAME exist in Jaeger."
else
  echo "Trace for $SERVICE_NAME does not exist in Jaeger."
  exit 1  # Fail the test step if the trace doesn't exist
fi
