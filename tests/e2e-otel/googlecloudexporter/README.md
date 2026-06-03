# Google Cloud Exporter E2E Test

This test validates the Google Cloud Exporter functionality using Workload Identity Federation for authentication on OpenShift clusters.

## Overview

The Google Cloud Exporter sends OpenTelemetry data to Google Cloud services:
- **Traces** → Google Cloud Trace
- **Metrics** → Google Cloud Monitoring
- **Logs** → Google Cloud Logging

## Authentication

This test uses **Workload Identity Federation (WIF)** for secure authentication without requiring service account keys. The setup creates:

1. A Google Cloud Service Account with appropriate permissions
2. An OpenShift Service Account in the test namespace
3. Workload Identity bindings between them
4. A credential configuration file mounted as a ConfigMap

## Prerequisites

### Google Cloud Setup
- Valid GCP project with billing enabled
- `gcloud` CLI authenticated and configured
- The following APIs enabled:
  - Cloud Trace API
  - Cloud Monitoring API
  - Cloud Logging API
  - IAM Service Account Credentials API

### OpenShift/Kubernetes Setup
- OpenShift cluster with Workload Identity Federation configured
- OpenTelemetry Operator installed
- `oc` CLI authenticated to the cluster

### Required Permissions
- GCP: `roles/iam.serviceAccountAdmin`, `roles/iam.workloadIdentityPoolAdmin`
- OpenShift: Cluster admin or sufficient permissions to create projects and service accounts

## Test Components

### Files Structure
```
tests/e2e-otel/googlecloudexporter/
├── README.md                           # This documentation
├── chainsaw-test.yaml                  # Chainsaw test definition
├── otel-collector.yaml                 # OpenTelemetry Collector configuration
├── otel-collector-assert.yaml          # Collector validation assertions
├── telemetry-generator.yaml            # Test data generator pod
├── telemetry-generator-assert.yaml     # Generator validation assertions
├── gcp-wif-create.sh                   # WIF setup script
├── gcp-wif-delete.sh                   # WIF cleanup script
└── check_gcp_telemetry.sh              # Verification script
```

### Authentication Setup (`gcp-wif-create.sh`)
- Creates OpenShift namespace and service account
- Creates Google Cloud service account with required IAM roles:
  - `roles/cloudtrace.agent` (for traces)
  - `roles/monitoring.metricWriter` (for metrics)
  - `roles/logging.logWriter` (for logs)
- Establishes Workload Identity Federation binding
- Generates and mounts credential configuration

### OpenTelemetry Collector Configuration
The collector is configured with:
- **Google Cloud Exporter** for all three signal types
- **Resource detection** for GCP and Kubernetes metadata
- **K8s attributes processor** for pod/namespace context
- **Transform processor** for attribute compatibility
- **Batch processor** for efficient export

### Telemetry Generator
Uses the `telemetrygen` tool to generate test data for all signal types:
- **Traces**: 10 test spans with service name `gcp-exporter-test`
- **Metrics**: 10 test metrics with service name `gcp-exporter-test`
- **Logs**: 10 test log entries with service name `gcp-exporter-test`

All telemetry includes custom attributes:
- `test.type="googlecloudexporter"`
- `test.signal="<traces|metrics|logs>"`

## Running the Test

### Manual Execution
```bash
# Set your GCP project (optional - uses gcloud default)
export GCP_PROJECT_ID="your-project-id"

# Run the test
chainsaw test --test-dir tests/e2e-otel/googlecloudexporter/
```

### Environment Variables
- `GCP_PROJECT_ID`: Target GCP project (optional, defaults to gcloud config)

## Verification

The test performs comprehensive validation:

1. **Infrastructure**: Verifies collector and generator pods are running
2. **Authentication**: Checks WIF credential setup and mounting
3. **Data Export**: Confirms telemetry data appears in Google Cloud services
4. **Error Detection**: Scans logs for export failures or authentication issues

### Manual Verification Commands

```bash
# Check collector logs
kubectl logs -n chainsaw-googlecloudexporter -l app.kubernetes.io/name=opentelemetry-collector

# Check telemetrygen job status
kubectl get jobs -n chainsaw-googlecloudexporter

# Check traces in GCP
gcloud trace list-traces --project=$GCP_PROJECT_ID --filter="spanName:telemetrygen" --limit=10

# Check metrics in GCP Monitoring (via Cloud Console)
# https://console.cloud.google.com/monitoring/metrics-explorer
# Look for metrics with prefix "opentelemetry.io" and service "gcp-exporter-test"

# Check logs in GCP Logging
gcloud logging read "
  resource.type=k8s_container
  AND (
    jsonPayload.body.stringValue:(\"telemetrygen\") OR
    jsonPayload.service.name=gcp-exporter-test
  )
  AND timestamp >= \"$(date -u -d '10 minutes ago' '+%Y-%m-%dT%H:%M:%SZ')\"
" --project=$GCP_PROJECT_ID
```

## Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify Workload Identity Pool and Provider exist
   - Check service account IAM bindings
   - Ensure OIDC issuer is correctly configured

2. **No Data in Google Cloud**
   - Check collector logs for export errors
   - Verify Google Cloud APIs are enabled
   - Confirm project permissions and quotas

3. **Pod Startup Issues**
   - Check OpenTelemetry Operator status
   - Verify ConfigMap creation and mounting
   - Review namespace permissions

### Debug Commands

```bash
# Check WIF configuration
oc describe configmap gcp-wif-credentials -n chainsaw-googlecloudexporter

# Verify service account binding
gcloud iam service-accounts get-iam-policy \
  otel-googlecloudexporter-sa@$GCP_PROJECT_ID.iam.gserviceaccount.com

# Check collector configuration
kubectl get opentelemetrycollector -n chainsaw-googlecloudexporter -o yaml

# Check telemetrygen job logs
kubectl logs -n chainsaw-googlecloudexporter job/generate-traces
kubectl logs -n chainsaw-googlecloudexporter job/generate-metrics
kubectl logs -n chainsaw-googlecloudexporter job/generate-logs
```

## Cleanup

The test includes automatic cleanup in both `catch` and `finally` blocks, but manual cleanup can be performed:

```bash
# Run cleanup script
./gcp-wif-delete.sh

# Or clean up manually
oc delete project chainsaw-googlecloudexporter
gcloud iam service-accounts delete otel-googlecloudexporter-sa@$GCP_PROJECT_ID.iam.gserviceaccount.com
```

## Security Considerations

- Uses Workload Identity Federation instead of service account keys
- Minimal IAM permissions (write-only to specific services)
- Credentials are mounted as read-only volumes
- Service account tokens have limited lifetime (1 hour)
- Test namespace isolation

## References

- [Google Cloud Exporter Documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/googlecloudexporter/README.md)
- [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)
- [OpenTelemetry Operator](https://github.com/open-telemetry/opentelemetry-operator)
- [Chainsaw Testing Framework](https://kyverno.github.io/chainsaw/)