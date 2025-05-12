#!/bin/bash

# Define the label selector
LABEL_SELECTOR="app.kubernetes.io/instance=chainsaw-kafka.kafka-receiver"

# Define the search strings
SEARCH_STRING1='-> service.name: Str("kafka")'
SEARCH_STRING2='-> test: Str(chainsaw-kafka)'

# Get the list of pods with the specified label
PODS=$(kubectl -n chainsaw-kafka get pods -l $LABEL_SELECTOR -o jsonpath='{.items[*].metadata.name}')

# Initialize flags to track if strings are found
FOUND1=false
FOUND2=false

# Loop through each pod and search for the strings in the logs
for POD in $PODS; do
    # Search for the first string
    if ! $FOUND1 && kubectl -n chainsaw-kafka logs $POD | grep -q -- "$SEARCH_STRING1"; then
        echo "\"$SEARCH_STRING1\" found in $POD"
        FOUND1=true
    fi
    # Search for the second string
    if ! $FOUND2 && kubectl -n chainsaw-kafka logs $POD | grep -q -- "$SEARCH_STRING2"; then
        echo "\"$SEARCH_STRING2\" found in $POD"
        FOUND2=true
    fi
done

# Check if either of the strings was not found
if ! $FOUND1 || ! $FOUND2; then
    echo "No Traces with service name Kafka and attribute test=chainsaw-kafka found."
    exit 1
else
    echo "Traces with service name Kafka and attribute test=chainsaw-kafka found."
fi
