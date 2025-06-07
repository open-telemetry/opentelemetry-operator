#!/bin/bash

# Create a temporary directory to store must-gather
MUST_GATHER_DIR=$(mktemp -d)

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=ghcr.io/open-telemetry/opentelemetry-operator/must-gather:latest -- /usr/bin/must-gather --operator-namespace $otelnamespace

# Define required files and directories
REQUIRED_ITEMS=(
  event-filter.html
  *-must-gather-sha256-*/olm/*opentelemetry-operato*.yaml
  *-must-gather-sha256-*/olm/clusterserviceversion-opentelemetry-operator-v*.yaml
  *-must-gather-sha256-*/olm/installplan-install-*.yaml
  *-must-gather-sha256-*/olm/subscription-opentelemetry-operator-v*-sub.yaml
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
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/service-sidecar-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/opentelemetrycollector-sidecar.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/service-sidecar-collector-monitoring.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/configmap-sidecar-collector-*.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/serviceaccount-sidecar-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/sidecar/service-sidecar-collector-headless.yaml
  *-must-gather-sha256-*/chainsaw-must-gather/instrumentation-nodejs.yaml
  *-must-gather-sha256-*/opentelemetry-operator-controller-manager-*
  *-must-gather-sha256-*/deployment-opentelemetry-operator-controller-manager.yaml
  timestamp
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

# Cleanup the must-gather directory
rm -rf $MUST_GATHER_DIR
