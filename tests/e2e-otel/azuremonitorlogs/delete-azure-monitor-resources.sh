#!/bin/bash
set -e

RG_NAME="otel-azure-monitor-logs"

echo "--- Cleaning up Azure Monitor resources ---"

# 1. Delete the service principal
SP_APP_ID=$(oc -n "$NAMESPACE" get configmap azure-monitor-test-config -o jsonpath='{.data.SP_APP_ID}' 2>/dev/null || echo "")
if [ -n "$SP_APP_ID" ]; then
    echo "Deleting service principal: $SP_APP_ID"
    az ad app delete --id "$SP_APP_ID" || true
else
    SP_NAME="otel-azm-logs-sp"
    FALLBACK_SP=$(az ad sp list --display-name "$SP_NAME" --query "[0].appId" -o tsv 2>/dev/null || echo "")
    if [ -n "$FALLBACK_SP" ]; then
        echo "Deleting service principal by name: $SP_NAME ($FALLBACK_SP)"
        az ad app delete --id "$FALLBACK_SP" || true
    fi
fi

# 2. Delete the resource group (cascades to workspace, DCE, DCR)
if [ "$(az group exists --name "$RG_NAME")" = "true" ]; then
    echo "Deleting resource group: $RG_NAME"
    az group delete --name "$RG_NAME" --yes || true
fi

# 3. Delete Kubernetes resources
oc -n "$NAMESPACE" delete secret azure-monitor-credentials --ignore-not-found=true 2>/dev/null || true
oc -n "$NAMESPACE" delete configmap azure-monitor-test-config --ignore-not-found=true 2>/dev/null || true

echo "--- Azure Monitor cleanup completed ---"
