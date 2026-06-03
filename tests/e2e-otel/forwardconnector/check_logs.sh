#!/bin/bash
# This script checks the OpenTelemetry collector pod for the presence of Logs.

# Define the label selector
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"
NAMESPACE=chainsaw-forwardconnector

# Define the search strings
SEARCH_STRING1='otel_pipeline_tag: Str(blue)'
SEARCH_STRING2='otel_pipeline_tag: Str(green)'
SEARCH_STRING3='Name           : lets-go'
SEARCH_STRING4='Name           : okey-dokey'

# Get the list of pods with the specified label
PODS=($(kubectl -n $NAMESPACE get pods -l $LABEL_SELECTOR -o jsonpath='{.items[*].metadata.name}'))

# Initialize flags to track if strings are found
FOUND1=false
FOUND2=false
FOUND3=false
FOUND4=false

# Loop through each pod and search for the strings in the logs
for POD in "${PODS[@]}"; do
    # Search for the first string
    if ! $FOUND1 && kubectl -n $NAMESPACE --tail=400 logs $POD | grep -q -- "$SEARCH_STRING1"; then
        echo "\"$SEARCH_STRING1\" found in $POD"
        FOUND1=true
    fi
    # Search for the second string
    if ! $FOUND2 && kubectl -n $NAMESPACE --tail=400 logs $POD | grep -q -- "$SEARCH_STRING2"; then
        echo "\"$SEARCH_STRING2\" found in $POD"
        FOUND2=true
    fi
    # Search for the third string
    if ! $FOUND3 && kubectl -n $NAMESPACE --tail=400 logs $POD | grep -q -- "$SEARCH_STRING3"; then
        echo "\"$SEARCH_STRING3\" found in $POD"
        FOUND3=true
    fi
    # Search for the fourth string
    if ! $FOUND4 && kubectl -n $NAMESPACE --tail=400 logs $POD | grep -q -- "$SEARCH_STRING4"; then
        echo "\"$SEARCH_STRING4\" found in $POD"
        FOUND4=true
    fi
done

# Check if any of the strings was not found
if ! $FOUND1 || ! $FOUND2 || ! $FOUND3 || ! $FOUND4; then
    echo "No Traces found in OpenTelemetry collector"
    exit 1
else
    echo "Found all the Traces in OpenTelemetry collector."
fi
