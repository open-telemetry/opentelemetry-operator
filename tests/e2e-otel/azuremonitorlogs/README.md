# OpenTelemetry Azure Monitor Logs Forwarding Test

This test validates forwarding OpenShift cluster application logs to Azure Monitor via an OpenTelemetry Collector and ClusterLogForwarder (CLF). Logs flow from application pods through the Cluster Logging Operator (CLO) collectors, into an OTel Collector gateway, and are exported to Azure Monitor using OTLP over HTTP with OAuth2 authentication.

## What This Test Does

The test validates that:
- Azure Monitor resources (Log Analytics workspace, DCE, DCR) can be provisioned via ARM template
- An OTel Collector with `oauth2client` extension authenticates to Azure Monitor using a service principal
- The `otlphttp` exporter sends logs to Azure Monitor's OTLP ingestion endpoint
- A ClusterLogForwarder routes application logs to the OTel Collector via OTLP
- Logs arrive in the Azure Monitor `OTelLogs` table and can be queried via KQL

## Prerequisites

- **Azure CLI** (`az`) logged in with permissions to create resource groups, workspaces, service principals, and DCR/DCE resources
- **Azure subscription** set to the target subscription (e.g., `az account set --subscription <name>`)
- **OpenShift cluster** with:
  - OpenTelemetry Operator installed
  - Cluster Logging Operator (CLO) 6.x installed
  - `oc` CLI configured with cluster-admin access
- **Chainsaw** test framework (for automated test execution)

## Architecture

```
+------------------+     +---------------------+     +---------------------+     +-------------------+
| ocp-logtest      | --> | CLO Collector Pods   | --> | OTel Collector      | --> | Azure Monitor     |
| (log generator)  |     | (openshift-logging)  |     | (otlphttp exporter) |     | (OTelLogs table)  |
+------------------+     +---------------------+     +---------------------+     +-------------------+
  Writes to stdout         ClusterLogForwarder         oauth2client extension      DCE -> DCR ->
  ~1 log/sec               routes application          authenticates via SP        Log Analytics
  SVTLogger format         logs via OTLP               sends to Azure OTLP        Workspace
                           to port 4318                ingestion endpoint
```

## Test Resources

### 1. Azure Monitor Resource Provisioning
- **File**: [`create-azure-monitor-resources.sh`](./create-azure-monitor-resources.sh)
- **Purpose**: Creates all Azure resources needed for OTLP log ingestion
- **Key Actions**:
  - Creates resource group `otel-azure-monitor-logs` in `eastus`
  - Creates Log Analytics workspace `otel-azm-logs`
  - Deploys DCE and DCR via ARM template
  - Creates service principal with `Monitoring Metrics Publisher` role on the DCR
  - Stores credentials in a Kubernetes secret `azure-monitor-credentials`
  - Stores test metadata in a ConfigMap `azure-monitor-test-config`

### 2. ARM Template for DCE/DCR
- **File**: [`dcr-arm-template.json`](./dcr-arm-template.json)
- **Purpose**: Deploys Data Collection Endpoint (DCE) and Data Collection Rule (DCR)
- **Key Features**:
  - Uses API version `2024-03-11` with `directDataSources.otelLogs`
  - Routes `Microsoft-OTel-Logs` stream to Log Analytics workspace
  - Outputs DCR immutable ID, DCE logs ingestion endpoint, and DCR resource ID

### 3. Azure Resource Cleanup
- **File**: [`delete-azure-monitor-resources.sh`](./delete-azure-monitor-resources.sh)
- **Purpose**: Tears down all Azure resources after the test
- **Key Actions**: Deletes service principal, resource group (cascading to workspace/DCE/DCR), and Kubernetes secret/ConfigMap

### 4. OpenTelemetry Collector
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OTel Collector CR (Deployment mode) with Azure Monitor exporter
- **Key Features**:
  - `oauth2client` extension for Azure AD authentication via service principal
  - `otlp` receiver (HTTP port 4318, gRPC port 4317)
  - `transform` processor to merge resource attributes into log attributes
  - `memory_limiter` and `batch` processors
  - `otlphttp/azure` exporter using `logs_endpoint` for full OTLP URL
  - `debug` exporter for troubleshooting
  - Uses the operator's default collector image (the `image` field can be uncommented to pin a specific version)

### 5. OTel Collector Assertion
- **File**: [`otel-collector-assert.yaml`](./otel-collector-assert.yaml)
- **Purpose**: Asserts the collector Deployment has 1 ready replica and Service exposes ports 4317/4318

### 6. ClusterLogForwarder Setup
- **File**: [`setup-cluster-log-forwarder.sh`](./setup-cluster-log-forwarder.sh)
- **Purpose**: Creates the logcollector SA, grants permissions, and deploys ClusterLogForwarder
- **Key Actions**:
  - Creates `logcollector` service account in `openshift-logging` namespace
  - Grants `collect-application-logs` cluster role
  - Creates CLF with OTLP output pointing to the OTel Collector service
  - Requires `observability.openshift.io/tech-preview-otlp-output: "enabled"` annotation
  - Waits for CLO collector pods to become ready

### 7. ClusterLogForwarder Cleanup
- **File**: [`cleanup-cluster-log-forwarder.sh`](./cleanup-cluster-log-forwarder.sh)
- **Purpose**: Deletes the ClusterLogForwarder, cluster role bindings, and logcollector SA from `openshift-logging` namespace

### 8. Log Generator Application
- **File**: [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)
- **Contains**: ConfigMap and ReplicationController for log generation
- **Key Features**:
  - Uses `ocp-logtest` image that writes `SVTLogger` lines to stdout at ~1 log/sec
  - Runs as non-root user (UID 1000) with restricted security context

### 9. Log Generator Assertion
- **File**: [`app-plaintext-logs-assert.yaml`](./app-plaintext-logs-assert.yaml)
- **Purpose**: Asserts the ReplicationController has 1 available/ready replica

### 10. Azure Monitor Verification
- **File**: [`check_azure_logs.sh`](./check_azure_logs.sh)
- **Purpose**: Queries Azure Monitor to verify logs arrived
- **Key Features**:
  - Reads workspace customer ID from ConfigMap
  - Checks collector pod logs for export errors
  - Queries `OTelLogs` table via KQL: `OTelLogs | where Body has 'SVTLogger' | take 10`
  - Retries up to 25 times with 30-second intervals (~12 minutes)

### 11. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow with 5 steps and cleanup handlers

## Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Azure Monitor resources and credentials** - Run [`create-azure-monitor-resources.sh`](./create-azure-monitor-resources.sh) (cleanup: [`delete-azure-monitor-resources.sh`](./delete-azure-monitor-resources.sh))
2. **Deploy OTel Collector with Azure Monitor exporter** - Apply [`otel-collector.yaml`](./otel-collector.yaml), assert readiness
3. **Grant log collector permissions and deploy ClusterLogForwarder** - Run [`setup-cluster-log-forwarder.sh`](./setup-cluster-log-forwarder.sh) (cleanup: [`cleanup-cluster-log-forwarder.sh`](./cleanup-cluster-log-forwarder.sh))
4. **Deploy log generator application** - Annotate namespace for SCC, apply [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml), assert readiness
5. **Verify logs are received in Azure Monitor** - Run [`check_azure_logs.sh`](./check_azure_logs.sh)

## Verification

The [`check_azure_logs.sh`](./check_azure_logs.sh) script validates:
- The OTel Collector pod has no export errors
- The Azure Monitor `OTelLogs` table contains log entries with `SVTLogger` in the `Body` column
- Returns the first matching log entry as a sample, showing attributes like `k8s.namespace.name`, `k8s.pod.name`, and `openshift.log.type`

## Cleanup

The test automatically cleans up:
- **Azure resources**: Service principal, resource group (cascading to workspace, DCE, DCR), Kubernetes secret and ConfigMap
- **Cluster resources**: ClusterLogForwarder, logcollector SA, and cluster role bindings in `openshift-logging`; log generator pods and OTel Collector in the test namespace
- **Namespace**: Chainsaw deletes the test namespace automatically

## Key Configuration Notes

- **Service account ordering**: The `logcollector` SA must be created *before* applying the ClusterLogForwarder. CLO 6.x does not create it automatically.
- **Table name**: Logs land in `OTelLogs` (not `OTLPLogs`). Use this table name in KQL queries.
- **OAuth2 scope**: Use `https://monitor.azure.com/.default` (single slash, not double `//`).
- **Exporter endpoint**: Use `logs_endpoint` (full URL including `/v1/logs`) instead of `endpoint` (which appends `/v1/logs` automatically). The full URL format is: `https://<dce-domain>/dataCollectionRules/<dcr-immutable-id>/streams/Microsoft-OTLP-Logs/otlp/v1/logs`
- **OTLP output annotation**: The CLF requires the tech-preview annotation `observability.openshift.io/tech-preview-otlp-output: "enabled"` for OTLP output type.
- **SCC annotations**: The test namespace needs `openshift.io/sa.scc.uid-range=1000/1000` and `openshift.io/sa.scc.supplemental-groups=3000/1000` for the log generator to run.
- **Workspace soft-delete**: Azure retains deleted Log Analytics workspaces for 14 days. The create script purges any soft-deleted workspace to avoid stale table state.
- **Ingestion latency**: Azure Monitor typically takes 3-8 minutes to ingest OTLP data into the `OTelLogs` table.

## Running the Automated Test

```bash
export KUBECONFIG=/path/to/kubeconfig
az login
az account set --subscription "<subscription-name>"
chainsaw test --test-dir tests/e2e-otel/azuremonitorlogs --no-color
```

---

## Manual Setup Guide

This section provides step-by-step instructions to manually set up the complete log forwarding pipeline from an OpenShift cluster to Azure Monitor via an OpenTelemetry Collector.

### Step 1: Log in to Azure and Set Subscription

```bash
az login
az account set --subscription "<your-subscription-name>"
az account show --query "{name:name, id:id}" -o table
```

### Step 2: Create a Resource Group

```bash
LOCATION="eastus"
RG_NAME="otel-azure-monitor-logs"

az group create --name "$RG_NAME" --location "$LOCATION" --output none
```

### Step 3: Create a Log Analytics Workspace

```bash
WORKSPACE_NAME="otel-azm-logs"

az monitor log-analytics workspace create \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --location "$LOCATION" \
    --output none

# Capture workspace IDs for later use
WORKSPACE_RESOURCE_ID=$(az monitor log-analytics workspace show \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --query "id" -o tsv)

WORKSPACE_CUSTOMER_ID=$(az monitor log-analytics workspace show \
    --resource-group "$RG_NAME" \
    --workspace-name "$WORKSPACE_NAME" \
    --query "customerId" -o tsv)

echo "Workspace Resource ID: $WORKSPACE_RESOURCE_ID"
echo "Workspace Customer ID: $WORKSPACE_CUSTOMER_ID"
```

### Step 4: Deploy Data Collection Endpoint and Rule

Save the following ARM template as `dcr-arm-template.json`:

```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "location": { "type": "string" },
    "logAnalyticsWorkspaceResourceId": { "type": "string" },
    "dataCollectionEndpointName": { "type": "string", "defaultValue": "otel-azm-dce" },
    "dataCollectionRuleName": { "type": "string", "defaultValue": "otel-azm-dcr" }
  },
  "resources": [
    {
      "type": "Microsoft.Insights/dataCollectionEndpoints",
      "apiVersion": "2024-03-11",
      "name": "[parameters('dataCollectionEndpointName')]",
      "location": "[parameters('location')]",
      "properties": {
        "networkAcls": { "publicNetworkAccess": "Enabled" }
      }
    },
    {
      "type": "Microsoft.Insights/dataCollectionRules",
      "apiVersion": "2024-03-11",
      "name": "[parameters('dataCollectionRuleName')]",
      "location": "[parameters('location')]",
      "dependsOn": [
        "[resourceId('Microsoft.Insights/dataCollectionEndpoints', parameters('dataCollectionEndpointName'))]"
      ],
      "properties": {
        "dataCollectionEndpointId": "[resourceId('Microsoft.Insights/dataCollectionEndpoints', parameters('dataCollectionEndpointName'))]",
        "directDataSources": {
          "otelLogs": [
            {
              "streams": ["Microsoft-OTel-Logs"],
              "enrichWithResourceAttributes": ["*"],
              "name": "otelLogsDirectSource"
            }
          ]
        },
        "destinations": {
          "logAnalytics": [
            {
              "workspaceResourceId": "[parameters('logAnalyticsWorkspaceResourceId')]",
              "name": "logAnalyticsDest"
            }
          ]
        },
        "dataFlows": [
          {
            "streams": ["Microsoft-OTel-Logs"],
            "destinations": ["logAnalyticsDest"]
          }
        ]
      }
    }
  ],
  "outputs": {
    "dcrImmutableId": {
      "type": "string",
      "value": "[reference(resourceId('Microsoft.Insights/dataCollectionRules', parameters('dataCollectionRuleName')), '2024-03-11', 'full').properties.immutableId]"
    },
    "dceLogsEndpoint": {
      "type": "string",
      "value": "[reference(resourceId('Microsoft.Insights/dataCollectionEndpoints', parameters('dataCollectionEndpointName')), '2024-03-11', 'full').properties.logsIngestion.endpoint]"
    },
    "dcrResourceId": {
      "type": "string",
      "value": "[resourceId('Microsoft.Insights/dataCollectionRules', parameters('dataCollectionRuleName'))]"
    }
  }
}
```

Deploy the template:

```bash
DCE_NAME="otel-azm-dce"
DCR_NAME="otel-azm-dcr"

DEPLOY_OUTPUT=$(az deployment group create \
    --resource-group "$RG_NAME" \
    --template-file dcr-arm-template.json \
    --parameters \
        location="$LOCATION" \
        logAnalyticsWorkspaceResourceId="$WORKSPACE_RESOURCE_ID" \
        dataCollectionEndpointName="$DCE_NAME" \
        dataCollectionRuleName="$DCR_NAME" \
    --output json)

DCR_IMMUTABLE_ID=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dcrImmutableId.value')
DCE_LOGS_ENDPOINT=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dceLogsEndpoint.value')
DCR_RESOURCE_ID=$(echo "$DEPLOY_OUTPUT" | jq -r '.properties.outputs.dcrResourceId.value')

echo "DCR Immutable ID:        $DCR_IMMUTABLE_ID"
echo "DCE Logs Endpoint:       $DCE_LOGS_ENDPOINT"
echo "DCR Resource ID:         $DCR_RESOURCE_ID"
```

> **Note**: If the deployment fails with `Table for output stream 'Microsoft-OTel-Logs' is not available`, the workspace may be recovering from a soft-delete. Wait 30 seconds and retry the deployment.

### Step 5: Create a Service Principal

```bash
SP_NAME="otel-azm-logs-sp"

SP_JSON=$(az ad sp create-for-rbac \
    --name "$SP_NAME" \
    --role "Monitoring Metrics Publisher" \
    --scopes "$DCR_RESOURCE_ID" \
    --output json)

SP_APP_ID=$(echo "$SP_JSON" | jq -r '.appId')
SP_PASSWORD=$(echo "$SP_JSON" | jq -r '.password')
TENANT_ID=$(echo "$SP_JSON" | jq -r '.tenant')

echo "SP App ID:   $SP_APP_ID"
echo "Tenant ID:   $TENANT_ID"
```

> **Important**: Save the `SP_PASSWORD` securely. It is only shown once at creation time.

### Step 6: Construct the OTLP Endpoint URL

```bash
OTLP_ENDPOINT="${DCE_LOGS_ENDPOINT}/dataCollectionRules/${DCR_IMMUTABLE_ID}/streams/Microsoft-OTLP-Logs/otlp/v1/logs"
echo "OTLP Endpoint: $OTLP_ENDPOINT"
```

The full URL format is:
```
https://<dce-domain>/dataCollectionRules/<dcr-immutable-id>/streams/Microsoft-OTLP-Logs/otlp/v1/logs
```

### Step 7: Create the Kubernetes Secret

Create a namespace (or use an existing one) and store the Azure credentials:

```bash
NAMESPACE="otel-azure-logs"
oc new-project "$NAMESPACE" 2>/dev/null || true

oc -n "$NAMESPACE" create secret generic azure-monitor-credentials \
    --from-literal=AZURE_CLIENT_ID="$SP_APP_ID" \
    --from-literal=AZURE_CLIENT_SECRET="$SP_PASSWORD" \
    --from-literal=AZURE_TENANT_ID="$TENANT_ID" \
    --from-literal=AZURE_OTLP_ENDPOINT="$OTLP_ENDPOINT"
```

### Step 8: Deploy the OpenTelemetry Collector

Apply the following OTel Collector CR. It uses the `oauth2client` extension to authenticate with Azure AD and the `otlphttp` exporter to send logs to Azure Monitor:

```bash
cat <<'EOF' | oc apply -n "$NAMESPACE" -f -
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel-logs-gateway
spec:
  mode: deployment
  env:
    - name: AZURE_CLIENT_ID
      valueFrom:
        secretKeyRef:
          name: azure-monitor-credentials
          key: AZURE_CLIENT_ID
    - name: AZURE_CLIENT_SECRET
      valueFrom:
        secretKeyRef:
          name: azure-monitor-credentials
          key: AZURE_CLIENT_SECRET
    - name: AZURE_TENANT_ID
      valueFrom:
        secretKeyRef:
          name: azure-monitor-credentials
          key: AZURE_TENANT_ID
    - name: AZURE_OTLP_ENDPOINT
      valueFrom:
        secretKeyRef:
          name: azure-monitor-credentials
          key: AZURE_OTLP_ENDPOINT
  config:
    extensions:
      oauth2client:
        client_id: ${env:AZURE_CLIENT_ID}
        client_secret: ${env:AZURE_CLIENT_SECRET}
        token_url: "https://login.microsoftonline.com/${env:AZURE_TENANT_ID}/oauth2/v2.0/token"
        scopes: ["https://monitor.azure.com/.default"]

    receivers:
      otlp:
        protocols:
          http:
            endpoint: 0.0.0.0:4318
          grpc:
            endpoint: 0.0.0.0:4317

    processors:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 80
        spike_limit_percentage: 20
      transform:
        error_mode: ignore
        log_statements:
          - context: log
            statements:
              - merge_maps(attributes, resource.attributes, "insert")
      batch:
        send_batch_max_size: 10240
        send_batch_size: 8192
        timeout: 5s

    exporters:
      debug:
        verbosity: detailed
      otlphttp/azure:
        auth:
          authenticator: oauth2client
        logs_endpoint: "${env:AZURE_OTLP_ENDPOINT}"

    service:
      extensions: [oauth2client]
      pipelines:
        logs:
          receivers: [otlp]
          processors: [memory_limiter, transform, batch]
          exporters: [otlphttp/azure, debug]
EOF
```

Wait for the collector to be ready:

```bash
oc -n "$NAMESPACE" wait deployment/otel-logs-gateway-collector \
    --for=condition=Available --timeout=120s
```

### Step 9: Create the logcollector Service Account and Grant Permissions

The `logcollector` service account must exist before the ClusterLogForwarder is created:

```bash
oc -n openshift-logging create serviceaccount logcollector

oc adm policy add-cluster-role-to-user collect-application-logs \
    system:serviceaccount:openshift-logging:logcollector
```

### Step 10: Create the ClusterLogForwarder

Replace `<NAMESPACE>` with your actual namespace (e.g., `otel-azure-logs`):

```bash
COLLECTOR_SERVICE="otel-logs-gateway-collector.${NAMESPACE}.svc.cluster.local"

cat <<EOF | oc apply -f -
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: instance
  namespace: openshift-logging
  annotations:
    observability.openshift.io/tech-preview-otlp-output: "enabled"
spec:
  managementState: Managed
  serviceAccount:
    name: logcollector
  filters:
    - name: multiline-exceptions
      type: detectMultilineException
  outputs:
    - name: otel-collector
      type: otlp
      otlp:
        url: http://${COLLECTOR_SERVICE}:4318/v1/logs
  pipelines:
    - name: app-logs-to-azure
      inputRefs:
        - application
      outputRefs:
        - otel-collector
      filterRefs:
        - multiline-exceptions
EOF
```

Wait for the CLO collector pods to start:

```bash
oc -n openshift-logging wait pod \
    -l app.kubernetes.io/component=collector \
    --for=condition=Ready --timeout=120s
```

### Step 11: Deploy a Log Generator (Optional)

To generate test logs, annotate the namespace and deploy the log generator:

```bash
oc annotate namespace "$NAMESPACE" \
    openshift.io/sa.scc.uid-range=1000/1000 --overwrite
oc annotate namespace "$NAMESPACE" \
    openshift.io/sa.scc.supplemental-groups=3000/1000 --overwrite

cat <<'EOF' | oc apply -n "$NAMESPACE" -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-log-plaintext-config
data:
  ocp_logtest.cfg: --rate 60.0
---
apiVersion: v1
kind: ReplicationController
metadata:
  labels:
    run: otel-logtest-plaintext
    test: otel-logtest-plaintext
  name: app-log-plaintext-rc
spec:
  replicas: 1
  template:
    metadata:
      generateName: otel-logtest-
      labels:
        run: otel-logtest-plaintext
        test: otel-logtest-plaintext
    spec:
      containers:
      - image: quay.io/openshifttest/ocp-logtest@sha256:6e2973d7d454ce412ad90e99ce584bf221866953da42858c4629873e53778606
        imagePullPolicy: IfNotPresent
        name: app-log-plaintext
        resources: {}
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
          privileged: false
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
        terminationMessagePath: /dev/termination-log
        volumeMounts:
        - mountPath: /var/lib/svt
          name: config
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      volumes:
      - configMap:
          name: app-log-plaintext-config
        name: config
EOF
```

### Step 12: Verify Logs in Azure Monitor

Wait 5-10 minutes for Azure Monitor ingestion, then query the `OTelLogs` table:

```bash
az monitor log-analytics query \
    --workspace "$WORKSPACE_CUSTOMER_ID" \
    --analytics-query "OTelLogs | where Body has 'SVTLogger' | take 10" \
    --output table
```

You can also query for all recent logs:

```bash
az monitor log-analytics query \
    --workspace "$WORKSPACE_CUSTOMER_ID" \
    --analytics-query "OTelLogs | order by TimeGenerated desc | take 20" \
    --output table
```

To check the OTel Collector logs for errors:

```bash
oc -n "$NAMESPACE" logs deployment/otel-logs-gateway-collector | \
    grep -iE "(error|fail|reject)"
```

### Step 13: Cleanup

Remove all resources when done:

```bash
# 1. Delete ClusterLogForwarder
oc delete clusterlogforwarder instance -n openshift-logging --ignore-not-found=true

# 2. Remove logcollector role binding and SA
oc adm policy remove-cluster-role-from-user collect-application-logs \
    system:serviceaccount:openshift-logging:logcollector
oc -n openshift-logging delete serviceaccount logcollector --ignore-not-found=true

# 3. Delete the test namespace (removes OTel Collector, log generator, secrets)
oc delete project "$NAMESPACE"

# 4. Delete Azure service principal
az ad app delete --id "$SP_APP_ID"

# 5. Delete Azure resource group (cascades to workspace, DCE, DCR)
az group delete --name "$RG_NAME" --yes
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| CLF status: `ServiceAccount "logcollector" not found` | SA not created before CLF | Create the SA first: `oc -n openshift-logging create serviceaccount logcollector` |
| CLF validation: `requires a valid tech-preview annotation` | Missing OTLP annotation | Add `observability.openshift.io/tech-preview-otlp-output: "enabled"` annotation to CLF |
| CLF output validation: `Additional type specific spec is required` | Wrong OTLP URL field | Use `otlp.url` (nested under `otlp`), not top-level `url` |
| DCR deployment: `Table for output stream ... is not available` | Workspace recovering from soft-delete | Wait 30s and retry, or purge the soft-deleted workspace first |
| No logs in Azure Monitor after 15+ minutes | Wrong table name in query | Query `OTelLogs` (not `OTLPLogs`) |
| Collector export errors with 401/403 | Wrong OAuth2 scope or SP permissions | Verify scope is `https://monitor.azure.com/.default` and SP has `Monitoring Metrics Publisher` role on the DCR |
| Log generator pods not starting | SCC restrictions | Annotate namespace: `openshift.io/sa.scc.uid-range=1000/1000` |

## Additional Information

- [Azure Monitor OTLP Ingestion with OTel Collector](https://learn.microsoft.com/en-us/azure/azure-monitor/containers/opentelemetry-protocol-ingestion)
- [Azure Monitor OTelLogs Table Reference](https://learn.microsoft.com/en-us/azure/azure-monitor/reference/tables/otellogs)
- [OTel Collector Azure Authentication Extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/azureauthextension)
- [ClusterLogForwarder API (observability.openshift.io/v1)](https://docs.openshift.com/container-platform/latest/observability/logging/log_collection_forwarding/configuring-log-forwarding.html)
