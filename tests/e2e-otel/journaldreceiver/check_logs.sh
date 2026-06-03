#!/bin/bash
# This script checks the OpenTelemetry collector pod for the presence of Logs.

# Define the label selector
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"
NAMESPACE="chainsaw-journald"

# Define the search strings
SEARCH_STRINGS=(
  "_SYSTEMD_UNIT"
  "_UID"
  "_HOSTNAME"
  "_SYSTEMD_INVOCATION_ID"
  "_SELINUX_CONTEXT"
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
