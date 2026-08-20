#!/bin/bash
set -e

LOCATION="eastus"
RG_NAME="otel-azure-monitor-logs"
WORKSPACE_NAME="otel-azm-logs"
DCE_NAME="otel-azm-dce"
DCR_NAME="otel-azm-dcr"
SP_NAME="otel-azm-logs-sp"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "--- Setting up Azure Monitor resources for OTLP log ingestion ---"
echo "  Resource Group: $RG_NAME"
echo "  Location: $LOCATION"
echo "  Namespace: $NAMESPACE"
echo "-------------------------------------------------------------------"

# Clean up any pre-existing resource group from a previous run
if [ "$(az group exists --name "$RG_NAME")" = "true" ]; then
    echo "Deleting existing resource group $RG_NAME (waiting for completion)..."
    az group delete --name "$RG_NAME" --yes || true
fi

# Delete any pre-existing service principal
EXISTING_SP=$(az ad sp list --display-name "$SP_NAME" --query "[0].appId" -o tsv 2>/dev/null || echo "")
if [ -n "$EXISTING_SP" ]; then
    echo "Deleting existing service principal $SP_NAME ($EXISTING_SP)..."
    az ad app delete --id "$EXISTING_SP" || true
fi

# 1. Create resource group
echo "Creating resource group: $RG_NAME"
az group create --name "$RG_NAME" --location "$LOCATION" --output none || { echo "Failed to create resource group"; exit 1; }

# 2. Purge any soft-deleted workspace with the same name to get a clean workspace
SUBSCRIPTION_ID=$(az account show --query "id" -o tsv)
echo "Checking for soft-deleted workspace: $WORKSPACE_NAME"
az rest --method delete \
    --url "https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RG_NAME}/providers/Microsoft.OperationalInsights/workspaces/${WORKSPACE_NAME}?api-version=2023-09-01&force=true" \
    2>/dev/null || true

echo "Creating Log Analytics workspace: $WORKSPACE_NAME"
az monitor log-analytics workspace create \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --location "$LOCATION" \
    --output none || { echo "Failed to create Log Analytics workspace"; exit 1; }

WORKSPACE_RESOURCE_ID=$(az monitor log-analytics workspace show \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --query "id" -o tsv)

WORKSPACE_CUSTOMER_ID=$(az monitor log-analytics workspace show \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --query "customerId" -o tsv)

echo "  Workspace Resource ID: $WORKSPACE_RESOURCE_ID"
echo "  Workspace Customer ID: $WORKSPACE_CUSTOMER_ID"

# 3. Deploy DCE and DCR using ARM template (API version 2024-03-11)
echo "Deploying Data Collection Endpoint and Rule via ARM template..."
DEPLOY_OUTPUT=""
for attempt in 1 2 3; do
    DEPLOY_OUTPUT=$(az deployment group create \
        --resource-group "$RG_NAME" \
        --template-file "${SCRIPT_DIR}/dcr-arm-template.json" \
        --parameters \
            location="$LOCATION" \
            logAnalyticsWorkspaceResourceId="$WORKSPACE_RESOURCE_ID" \
            dataCollectionEndpointName="$DCE_NAME" \
            dataCollectionRuleName="$DCR_NAME" \
        --output json 2>&1) && break
    echo "  DCR deployment attempt $attempt failed, retrying in 30s..."
    echo "  Error: $DEPLOY_OUTPUT"
    sleep 30
done
if ! echo "$DEPLOY_OUTPUT" | jq -e '.properties.outputs' > /dev/null 2>&1; then
    echo "Failed to deploy DCE/DCR ARM template after 3 attempts"
    echo "$DEPLOY_OUTPUT"
    exit 1
fi

DCR_IMMUTABLE_ID=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dcrImmutableId.value')
DCE_LOGS_ENDPOINT=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dceLogsEndpoint.value')
DCR_RESOURCE_ID=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dcrResourceId.value')

echo "  DCR Immutable ID: $DCR_IMMUTABLE_ID"
echo "  DCE Logs Ingestion Endpoint: $DCE_LOGS_ENDPOINT"
echo "  DCR Resource ID: $DCR_RESOURCE_ID"

# 4. Create Service Principal and assign Monitoring Metrics Publisher role on DCR
echo "Creating Service Principal: $SP_NAME"
SP_JSON=$(az ad sp create-for-rbac \
    --name "$SP_NAME" \
    --role "Monitoring Metrics Publisher" \
    --scopes "$DCR_RESOURCE_ID" \
    --output json) || { echo "Failed to create service principal"; exit 1; }

SP_APP_ID=$(echo "$SP_JSON" | jq -r '.appId')
SP_PASSWORD=$(echo "$SP_JSON" | jq -r '.password')
TENANT_ID=$(echo "$SP_JSON" | jq -r '.tenant')

echo "  SP App ID: $SP_APP_ID"
echo "  Tenant ID: $TENANT_ID"

# 5. Construct the OTLP ingestion endpoint (full URL for logs_endpoint config)
OTLP_ENDPOINT="${DCE_LOGS_ENDPOINT}/dataCollectionRules/${DCR_IMMUTABLE_ID}/streams/Microsoft-OTLP-Logs/otlp/v1/logs"
echo "  OTLP Endpoint: $OTLP_ENDPOINT"

# 6. Create Kubernetes secret with Azure credentials and OTLP endpoint
echo "Creating Kubernetes secret: azure-monitor-credentials in namespace $NAMESPACE"
oc -n "$NAMESPACE" create secret generic azure-monitor-credentials \
    --from-literal=AZURE_CLIENT_ID="$SP_APP_ID" \
    --from-literal=AZURE_CLIENT_SECRET="$SP_PASSWORD" \
    --from-literal=AZURE_TENANT_ID="$TENANT_ID" \
    --from-literal=AZURE_OTLP_ENDPOINT="$OTLP_ENDPOINT" || { echo "Failed to create K8s secret"; exit 1; }

# 7. Store test metadata in a ConfigMap for verification and cleanup scripts
echo "Creating ConfigMap: azure-monitor-test-config in namespace $NAMESPACE"
oc -n "$NAMESPACE" create configmap azure-monitor-test-config \
    --from-literal=RESOURCE_GROUP="$RG_NAME" \
    --from-literal=WORKSPACE_CUSTOMER_ID="$WORKSPACE_CUSTOMER_ID" \
    --from-literal=SP_APP_ID="$SP_APP_ID" \
    --from-literal=SP_NAME="$SP_NAME" || { echo "Failed to create ConfigMap"; exit 1; }

echo "--- Azure Monitor resources created successfully ---"
