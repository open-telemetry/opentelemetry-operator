#!/bin/bash
# This script checks the OpenTelemetry collector pod for the presence of specific logs.

# Define the label selector and namespace
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"
NAMESPACE="chainsaw-kubeletstatsreceiver"

# Define the search strings
SEARCH_STRINGS=(
  "container.cpu.time"
  "container.cpu.usage"
  "container.filesystem.available"
  "container.filesystem.capacity"
  "container.filesystem.usage"
  "container.memory.major_page_faults"
  "container.memory.page_faults"
  "container.memory.rss"
  "container.memory.usage"
  "container.memory.working_set"
  "k8s.node.cpu.time"
  "k8s.node.cpu.usage"
  "k8s.node.filesystem.available"
  "k8s.node.filesystem.capacity"
  "k8s.node.filesystem.usage"
  "k8s.node.memory.available"
  "k8s.node.memory.major_page_faults"
  "k8s.node.memory.page_faults"
  "k8s.node.memory.rss"
  "k8s.node.memory.usage"
  "k8s.node.memory.working_set"
  "k8s.pod.cpu.time"
  "k8s.pod.cpu.usage"
  "k8s.pod.filesystem.available"
  "k8s.pod.filesystem.capacity"
  "k8s.pod.filesystem.usage"
  "k8s.pod.memory.major_page_faults"
  "k8s.pod.memory.page_faults"
  "k8s.pod.memory.rss"
  "k8s.pod.memory.usage"
  "k8s.pod.memory.working_set"
  "k8s.pod.network.errors"
  "k8s.pod.network.io"
)

# Get the list of pods with the specified label
PODS=($(kubectl -n $NAMESPACE get pods -l $LABEL_SELECTOR -o jsonpath='{.items[*].metadata.name}'))

# Check if the PODS array is not empty
if [ ${#PODS[@]} -eq 0 ]; then
    echo "No pods found with label $LABEL_SELECTOR in namespace $NAMESPACE"
    exit 1
fi

# Take the first pod from the list
POD=${PODS[0]}

# Get all logs from the first pod
LOGS=$(kubectl -n $NAMESPACE logs $POD --tail=-1)

# Loop through each search string and check in the logs
for STRING in "${SEARCH_STRINGS[@]}"; do
    if echo "$LOGS" | grep -q -- "$STRING"; then
        echo "\"$STRING\" found in $POD"
    else
        echo "\"$STRING\" not found in $POD"
        exit 1
    fi
done

echo "Log search completed for all defined strings in pod $POD."
