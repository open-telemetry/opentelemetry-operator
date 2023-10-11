#!/bin/bash

# Define an array of service names
SERVICE_NAMES=("telemetrygen-http" "telemetrygen-grpc")

# Get the Jaeger URL
JAEGER_URL=$(oc -n kuttl-multi-cluster-receive get route jaeger-allinone -o json | jq '.spec.host' -r)

# Initialize a flag to check if any trace exists
trace_exists=false

# Loop through each service name
for SERVICE_NAME in "${SERVICE_NAMES[@]}"; do
  trace_count=$(curl -ksSL "https://$JAEGER_URL/api/traces?service=$SERVICE_NAME&limit=1" | jq -r '.data | length')
  if [[ $trace_count -gt 0 ]]; then
    echo "Traces for $SERVICE_NAME exist in Jaeger."
    trace_exists=true
  else
    echo "Trace for $SERVICE_NAME does not exist in Jaeger."
  fi
done

# Fail the test step if no traces exist for any service name
if ! $trace_exists; then
  exit 1
fi
