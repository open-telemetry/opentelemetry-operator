#!/bin/bash

# Define the required service names
required_service_names=("telemetrygen-http-blue" "telemetrygen-http-red" "telemetrygen-http-green")

# Get the list of pods with the specified label
pods=$(oc get pods -n chainsaw-lb -l app.kubernetes.io/name=chainsaw-lb-backends-collector -o jsonpath="{.items[*].metadata.name}")

# Initialize an empty string to hold all service names from all pods
all_service_names=""

for pod in $pods; do
  echo "Checking pod: $pod"

  # Get the logs of the pod
  logs=$(oc -n chainsaw-lb logs $pod)

  # Extract the unique service.name values from the logs
  # Look for the debug exporter format: "     -> service.name: Str(telemetrygen-http-red)"
  service_names=$(echo "$logs" | grep -o "service\.name: Str([^)]*)" | sed 's/service\.name: Str(\([^)]*\))/\1/' | sort | uniq)
  
  # If no service names found with the primary method, try alternative parsing
  if [ -z "$service_names" ]; then
    # Try JSON format: "service.name": "value"
    service_names=$(echo "$logs" | grep -o '"service\.name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"service\.name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' | sort | uniq)
  fi
  
  # Filter to only include telemetrygen service names (exclude otelcol and other internal services)
  filtered_service_names=""
  for service_name in $service_names; do
    if [[ "$service_name" == telemetrygen-http-* ]]; then
      filtered_service_names+="$service_name "
    fi
  done

  # If no trace service names found in this pod, that's fine (some pods may not receive traces)
  if [ -z "$filtered_service_names" ]; then
    echo "No trace service names found in pod $pod (this is normal - load balancing may not send traces to all pods)"
  else
    echo "Found trace service names in pod $pod: $filtered_service_names"
    
    # Check if any service.name found in this pod was already found in another pod
    for service_name in $filtered_service_names; do
      if echo "$all_service_names" | grep -q "$service_name"; then
        echo "Error: Service name $service_name found in more than one pod"
        echo "This indicates load balancing is not working correctly"
        exit 1
      else
        all_service_names+="$service_name "
        echo "Service.name $service_name found in pod $pod"
      fi
    done
  fi
done

echo "Summary: All service names found across all pods: $all_service_names"

# Check if all required service names are present across all pods
missing_services=""
for required_service_name in "${required_service_names[@]}"; do
  if ! echo "$all_service_names" | grep -q "$required_service_name"; then
    missing_services+="$required_service_name "
  fi
done

if [ -n "$missing_services" ]; then
  echo "Error: Required service names are missing from all pods: $missing_services"
  echo "This indicates traces were not generated or not processed correctly"
  exit 1
fi

echo "Success: All required service names are present and each appears in only one pod"
echo "Load balancing is working correctly!"