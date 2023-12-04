#!/bin/bash

# Name of the daemonset to check
DAEMONSET_NAME="daemonset-test-collector"

# Get the desired and ready pod counts for the daemonset
read DESIRED READY <<< $(kubectl get daemonset -n $NAMESPACE $DAEMONSET_NAME -o custom-columns=:status.desiredNumberScheduled,:status.numberReady --no-headers)

# Check if the desired count matches the ready count
if [ "$DESIRED" -eq "$READY" ]; then
  echo "Desired count ($DESIRED) matches the ready count ($READY) for $DAEMONSET_NAME."
else
  echo "Desired count ($DESIRED) does not match the ready count ($READY) for $DAEMONSET_NAME."
  exit 1
fi
