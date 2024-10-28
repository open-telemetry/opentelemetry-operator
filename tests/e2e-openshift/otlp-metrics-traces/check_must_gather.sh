#!/bin/bash

# Create the directory to store must-gather
MUST_GATHER_DIR=/tmp/otlp-metrics-traces
mkdir -p $MUST_GATHER_DIR

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=ghcr.io/open-telemetry/opentelemetry-operator/must-gather:latest -- /usr/bin/must-gather --operator-namespace $otelnamespace

# Define required files and directories
REQUIRED_ITEMS=(
  event-filter.html
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/olm/clusterserviceversion-opentelemetry-operator-*.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/olm/*opentelemetry-operator*.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/olm/installplan-install-*.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/olm/subscription-opentelemetry-operator-v*-sub.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/service-cluster-collector-collector-headless.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/deployment-cluster-collector-collector.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/service-cluster-collector-collector-monitoring.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/opentelemetrycollector-cluster-collector.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/configmap-cluster-collector-collector-57b76c99.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/serviceaccount-cluster-collector-collector.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/service-cluster-collector-collector.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/namespaces/chainsaw-otlp-metrics/cluster-collector/poddisruptionbudget-cluster-collector-collector.yaml
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/opentelemetry-operator-controller-manager-*
  ghcr-io-open-telemetry-opentelemetry-operator-must-gather-sha256-*/deployment-opentelemetry-operator-controller-manager.yaml
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
