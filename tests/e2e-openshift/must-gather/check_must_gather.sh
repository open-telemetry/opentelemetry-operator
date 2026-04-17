#!/usr/bin/env bash

# Create a temporary directory to store must-gather
MUST_GATHER_DIR=$(mktemp -d)

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=ghcr.io/open-telemetry/opentelemetry-operator/must-gather:latest -- /usr/bin/gather --operator-namespace $otelnamespace

# Define required files and directories
REQUIRED_ITEMS=(
  event-filter.html
  # stateful collector and its owned resources
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/opentelemetry.io/opentelemetrycollectors/stateful.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/opentelemetry.io/targetallocators/stateful.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/apps/statefulsets/stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/apps/deployments/stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/stateful-collector-headless.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/stateful-collector-monitoring.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/configmaps/stateful-collector-*.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/configmaps/stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/serviceaccounts/stateful-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/policy/poddisruptionbudgets/stateful-targetallocator.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/policy/poddisruptionbudgets/stateful-collector.yaml
  # gather collector and its owned resources
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/opentelemetry.io/opentelemetrycollectors/gather.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/apps/deployments/gather-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/gather-collector.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/gather-collector-monitoring.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/services/gather-collector-headless.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/configmaps/gather-collector-*.yaml
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/core/serviceaccounts/gather-collector.yaml
  # Instrumentation
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/opentelemetry.io/instrumentations/nodejs.yaml
  # Collector pod logs
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/pods/gather-collector-*/otc-container/otc-container/logs/current.log
  *-must-gather-sha256-*/namespaces/chainsaw-must-gather/pods/stateful-collector-*/otc-container/otc-container/logs/current.log
  # Operator deployment and logs
  *-must-gather-sha256-*/namespaces/$otelnamespace/apps/deployments/opentelemetry-operator-controller-manager.yaml
  *-must-gather-sha256-*/namespaces/$otelnamespace/pods/opentelemetry-operator-controller-manager-*/manager/manager/logs/current.log
  timestamp
)

# Define optional OLM-related items (only present when operator is deployed via OLM)
OPTIONAL_ITEMS=(
  *-must-gather-sha256-*/cluster-scoped-resources/operators.coreos.com/operators/*opentelemetry-operator*.yaml
  *-must-gather-sha256-*/namespaces/$otelnamespace/operators.coreos.com/clusterserviceversions/opentelemetry-operator-v*.yaml
  *-must-gather-sha256-*/namespaces/$otelnamespace/operators.coreos.com/installplans/install-*.yaml
  *-must-gather-sha256-*/namespaces/$otelnamespace/operators.coreos.com/subscriptions/opentelemetry-*.yaml
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
