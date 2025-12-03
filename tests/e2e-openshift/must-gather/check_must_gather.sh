#!/bin/bash

# Create a temporary directory to store must-gather
MUST_GATHER_DIR=$(mktemp -d)

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=ghcr.io/open-telemetry/opentelemetry-operator/must-gather:latest -- /usr/bin/must-gather --operator-namespace $otelnamespace

# Define required files and directories
REQUIRED_ITEMS=(
  event-filter.html
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/service-stateful-collector-headless.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/service-stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/targetallocator-stateful.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/deployment-stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/service-stateful-collector-monitoring.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/poddisruptionbudget-stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/poddisruptionbudget-stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/service-stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/configmap-stateful-collector-*.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/configmap-stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/statefulset-stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/opentelemetrycollector-stateful.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/stateful/serviceaccount-stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/service-gather-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/opentelemetrycollector-gather.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/service-gather-collector-monitoring.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/configmap-gather-collector-*.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/serviceaccount-gather-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/service-gather-collector-headless.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/gather/deployment-gather-collector.yaml
  *-must-gather-sha256-*/chainsaw-must-gather/instrumentation-nodejs.yaml
  *-must-gather-sha256-*/opentelemetry-operator-controller-manager-*
  *-must-gather-sha256-*/deployment-opentelemetry-operator-controller-manager.yaml
  timestamp
)

# Define optional OLM-related items (only present when operator is deployed via OLM)
OPTIONAL_ITEMS=(
  *-must-gather-sha256-*/olm/*opentelemetry-operator*.yaml
  *-must-gather-sha256-*/olm/clusterserviceversion-opentelemetry-operator-v*.yaml
  *-must-gather-sha256-*/olm/installplan-install-*.yaml
  *-must-gather-sha256-*/olm/subscription-opentelemetry-*.yaml
)

# Verify each required item
for item in "${REQUIRED_ITEMS[@]}"; do
  if ! find "$MUST_GATHER_DIR" -path "$MUST_GATHER_DIR/$item" -print -quit | grep -q .; then
    echo "Missing: $item"
    exit 1
  else
    echo "Found: $item"
  fi
done

# Verify optional items (don't fail if missing)
for item in "${OPTIONAL_ITEMS[@]}"; do
  if ! find "$MUST_GATHER_DIR" -path "$MUST_GATHER_DIR/$item" -print -quit | grep -q .; then
    echo "Missing optional: $item (OK - operator may not be deployed via OLM)"
  else
    echo "Found: $item"
  fi
done

# Cleanup the must-gather directory
rm -rf $MUST_GATHER_DIR
