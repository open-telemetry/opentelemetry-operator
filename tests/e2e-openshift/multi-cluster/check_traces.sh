#!/bin/bash

# Define an array of service names
SERVICE_NAMES=("telemetrygen-http" "telemetrygen-grpc")

# Get the Jaeger URL
JAEGER_URL=$(oc -n chainsaw-multi-cluster-receive get route jaeger-allinone -o json | jq '.spec.host' -r)

# Initialize a flag to check if traces exist for all services
all_traces_exist=false

# Keep checking until traces exist for all services
while [[ $all_traces_exist == false ]]; do
  all_traces_exist=true

  # Loop through each service name
  for SERVICE_NAME in "${SERVICE_NAMES[@]}"; do
    trace_count=$(curl -ksSL "https://$JAEGER_URL/api/traces?service=$SERVICE_NAME&limit=1" | jq -r '.data | length')
    if [[ $trace_count -gt 0 ]]; then
      echo "Traces for $SERVICE_NAME exist in Jaeger."
    else
      echo "Trace for $SERVICE_NAME does not exist in Jaeger."
      all_traces_exist=false
    fi
  done

  # If traces do not exist for all services, sleep for a while before the next check
  if [[ $all_traces_exist == false ]]; then
    sleep 10
  fi
done

echo "Traces exist for all service names."
