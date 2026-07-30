#!/usr/bin/env bash

# Create a temporary directory to store must-gather
MUST_GATHER_DIR=$(mktemp -d)

# Use MUSTGATHER_IMG if set, otherwise derive from the operator version in versions.txt
if [ -z "$MUSTGATHER_IMG" ]; then
  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  OPERATOR_VERSION=$(awk -F= '/^operator=/ {print $2}' "${SCRIPT_DIR}/../../../versions.txt")
  MUSTGATHER_IMG="ghcr.io/open-telemetry/opentelemetry-operator/must-gather:${OPERATOR_VERSION}"
fi

# Run the must-gather script
oc adm must-gather --dest-dir=$MUST_GATHER_DIR --image=$MUSTGATHER_IMG -- /usr/bin/gather --operator-namespace $otelnamespace

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

# Validate output with omc — download the binary matching the current host OS/arch
OMC=/tmp/omc
_OS=$(uname -s)
_ARCH=$(uname -m)
case "${_OS}/${_ARCH}" in
  Darwin/arm64)  _OMC_BIN="omc_Darwin_arm64" ;;
  Darwin/x86_64) _OMC_BIN="omc_Darwin_x86_64" ;;
  Linux/aarch64) _OMC_BIN="omc_Linux_arm64" ;;
  *)             _OMC_BIN="omc_Linux_x86_64" ;;
esac
curl -sL -o "$OMC" "https://github.com/gmeghnag/omc/releases/latest/download/${_OMC_BIN}"
chmod +x "$OMC"

OMC_ROOT=$(find "$MUST_GATHER_DIR" -maxdepth 1 -type d -name '*-must-gather-sha256-*' | head -1)
if [ -z "$OMC_ROOT" ]; then
  echo "omc: could not find must-gather subdirectory"
  exit 1
fi
$OMC use "$OMC_ROOT"

echo "--- omc validation ---"

$OMC get opentelemetrycollectors -A | grep -q "stateful" || { echo "omc: opentelemetrycollectors not found"; exit 1; }
echo "omc: opentelemetrycollectors OK"

$OMC get deployments -A | grep -q "gather-collector" || { echo "omc: deployments not found"; exit 1; }
echo "omc: deployments OK"

$OMC get pods -A | grep -q "gather-collector" || { echo "omc: pods not found"; exit 1; }
echo "omc: pods OK"

$OMC project chainsaw-must-gather >/dev/null

GATHER_POD=$(find "$OMC_ROOT" -path "*/pods/gather-collector-*" -type d -maxdepth 5 | head -1 | xargs -I{} basename {})
if [ -n "$GATHER_POD" ]; then
  $OMC logs "$GATHER_POD" -c otc-container 2>/dev/null | head -1 | grep -q . || { echo "omc: logs empty for $GATHER_POD"; exit 1; }
  echo "omc: logs OK"
else
  echo "omc: no gather-collector pod directory found, skipping log check"
fi

# Cleanup the must-gather directory
rm -rf $MUST_GATHER_DIR
