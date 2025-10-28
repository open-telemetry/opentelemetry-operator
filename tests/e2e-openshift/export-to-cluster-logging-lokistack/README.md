# Export to Cluster Logging LokiStack Test

This test demonstrates how to export OpenTelemetry logs to OpenShift's cluster logging infrastructure using LokiStack for centralized log management and analysis.

## Test Overview

This test creates:
1. A MinIO instance for LokiStack object storage
2. A LokiStack instance for log storage and querying
3. An OpenTelemetry Collector that processes and exports logs to LokiStack
4. Log generation to test the end-to-end flow
5. Integration with OpenShift logging UI plugin

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Red Hat OpenShift Logging Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### MinIO Object Storage

Deploy MinIO for LokiStack storage backend:

**Configuration:** [`install-minio.yaml`](./install-minio.yaml)

Creates MinIO infrastructure:
- 2Gi PersistentVolumeClaim for storage
- MinIO deployment with demo credentials (tempo/supersecret)
- Service for internal cluster access
- Secret with S3-compatible access configuration

### LokiStack Instance

Deploy LokiStack for log storage:

**Configuration:** [`install-loki.yaml`](./install-loki.yaml)

Creates a LokiStack with:
- 1x.demo size for testing environments
- S3-compatible storage via MinIO
- v13 schema with openshift-logging tenant mode
- Integration with cluster logging infrastructure

### OpenTelemetry Collector Configuration

Deploy collector with LokiStack integration:

**Configuration:** [`otel-collector.yaml`](./otel-collector.yaml)

Configures a complete collector setup with:
- Service account with LokiStack write permissions
- ClusterRole for accessing pods, namespaces, and nodes
- Bearer token authentication for LokiStack gateway
- k8sattributes processor for Kubernetes metadata
- Transform processor for log level normalization
- Dual pipeline: one for LokiStack export, one for debug output

### Log Generation

Generate test logs to validate the pipeline:

**Configuration:** [`generate-logs.yaml`](./generate-logs.yaml)

Creates a job that:
- Generates 20 structured log entries in OTLP format
- Sends logs via HTTP POST to the collector
- Includes service metadata and custom attributes
- Uses proper OTLP JSON structure for compatibility

### Logging UI Plugin

Enable the logging UI plugin for log visualization:

**Configuration:** [`logging-uiplugin.yaml`](./logging-uiplugin.yaml)

Configures the OpenShift console plugin for:
- Log visualization in the OpenShift web console
- Integration with LokiStack for log querying
- Enhanced logging user interface experience

## Deployment Steps

1. **Install MinIO for object storage:**
   ```bash
   oc apply -f install-minio.yaml
   ```

2. **Deploy LokiStack instance:**
   ```bash
   oc apply -f install-loki.yaml
   ```

3. **Deploy OpenTelemetry Collector with RBAC:**
   ```bash
   oc apply -f otel-collector.yaml
   ```

4. **Enable logging UI plugin:**
   ```bash
   oc apply -f logging-uiplugin.yaml
   ```

5. **Generate test logs:**
   ```bash
   oc apply -f generate-logs.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Storage Infrastructure
- **MinIO**: Object storage backend with `tempo` bucket
- **PVC**: 2Gi persistent volume for MinIO storage
- **Secret**: `logging-loki-s3` with MinIO access credentials

### Logging Stack
- **LokiStack**: `logging-loki` instance in demo mode
- **Gateway**: HTTP gateway for log ingestion
- **Storage Schema**: v13 schema with 2023-10-15 effective date

### OpenTelemetry Integration
- **Service Account**: `otel-collector-deployment` with logging permissions
- **Collector**: `otel-collector` with LokiStack exporter
- **RBAC**: Cluster role for writing to LokiStack

### Log Generation
- **Job**: `generate-logs` creating test log entries
- **UI Plugin**: Logging view plugin for OpenShift console

## Testing the Configuration

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_logs.sh`](./check_logs.sh)

The script verifies:
- Log generation job completes successfully
- Collector exports logs to LokiStack gateway
- LokiStack gateway receives and processes log entries
- End-to-end log flow from generation to storage

## Additional Verification Commands

Verify the logging infrastructure:

```bash
# Check LokiStack status
oc get lokistack logging-loki -o yaml

# Check MinIO deployment
oc get deployment minio -o yaml

# View LokiStack gateway service
oc get svc logging-loki-gateway-http

# Check collector service account permissions
oc auth can-i create application --as=system:serviceaccount:openshift-logging:otel-collector-deployment

# Port forward to MinIO for bucket verification
oc port-forward svc/minio 9000:9000 &
curl -u tempo:supersecret http://localhost:9000/minio/health/live  # Using demo test credentials

# Check collector metrics
oc port-forward svc/otel-collector 8888:8888 &
curl http://localhost:8888/metrics | grep otelcol_exporter
```

## Verification

The test verifies:
- ✅ MinIO is deployed and accessible as object storage
- ✅ LokiStack instance is ready and configured
- ✅ OpenTelemetry Collector has proper RBAC permissions
- ✅ Collector is configured with LokiStack OTLP exporter
- ✅ Bearer token authentication is working
- ✅ Log generation job completes successfully
- ✅ Logs are processed through k8sattributes and transform processors
- ✅ Logs are successfully exported to LokiStack
- ✅ Logging UI plugin is enabled for log visualization

## Key Features

- **LokiStack Integration**: Native integration with OpenShift cluster logging
- **Object Storage**: MinIO backend for log persistence
- **Authentication**: Bearer token authentication with service accounts
- **Log Processing**: Kubernetes attributes and log transformation
- **OTLP Protocol**: Uses OTLP HTTP for log export to LokiStack
- **Multi-Pipeline**: Separate pipelines for LokiStack export and debug output
- **UI Integration**: Logging console plugin for log visualization

## Configuration Notes

- LokiStack runs in `1x.demo` size for testing environments
- MinIO uses ephemeral storage with demo credentials (tempo/supersecret) - **FOR TESTING ONLY**
- Collector uses bearer token authentication for LokiStack access
- Service CA certificate is used for TLS communication with LokiStack
- Log processors add Kubernetes metadata and normalize severity levels
- The application log type is set for proper tenant routing in LokiStack 