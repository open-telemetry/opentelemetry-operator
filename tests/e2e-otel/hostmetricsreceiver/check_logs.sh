#!/bin/bash
# This script checks the OpenTelemetry collector pod for the presence of specific logs.

# Define the label selector and namespace
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"
NAMESPACE="chainsaw-hostmetrics"

# Define the search strings
SEARCH_STRINGS=(
  "process.pid"
  "process.parent_pid"
  "process.executable.name"
  "process.executable.path"
  "process.command"
  "process.cpu.time"
  "process.disk.io"
  "process.memory.usage"
  "process.memory.virtual"
  "system.cpu.load_average.15m"
  "system.cpu.load_average.1m"
  "system.cpu.load_average.5m"
  "system.cpu.time"
  "system.disk.io"
  "system.disk.io_time"
  "system.disk.merged"
  "system.disk.operation_time"
  "system.disk.operations"
  "system.disk.pending_operations"
  "system.disk.weighted_io_time"
  # TODO: Uncomment when issue is fixed
  # Filesystem scraper not working in hostmetrics receiver.
  # See: https://issues.redhat.com/browse/TRACING-5963
  # "system.filesystem.inodes.usage"
  # "system.filesystem.usage"
  "system.memory.usage"
  "system.network.connections"
  "system.network.dropped"
  "system.network.errors"
  "system.network.io"
  "system.network.packets"
  "system.paging.faults"
  "system.paging.operations"
  "system.processes.count"
  "system.processes.created"
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
