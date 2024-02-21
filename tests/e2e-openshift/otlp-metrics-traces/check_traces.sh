#!/bin/bash

JAEGER_URL=$(oc -n chainsaw-otlp-metrics get route jaeger-allinone -o json | jq '.spec.host' -r)
SERVICE_NAME="telemetrygen"

while true; do
  trace_exists=$(curl -ksSL "https://$JAEGER_URL/api/traces?service=$SERVICE_NAME&limit=1" | jq -r '.data | length')

  if [[ $trace_exists -gt 0 ]]; then
    echo "Traces for $SERVICE_NAME exist in Jaeger."
    break
  else
    echo "Trace for $SERVICE_NAME does not exist in Jaeger. Fetching again..."
  fi
done
