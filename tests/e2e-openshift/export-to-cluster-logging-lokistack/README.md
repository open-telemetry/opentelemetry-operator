# Export to Cluster Logging LokiStack Test

This test demonstrates how to export OpenTelemetry logs to OpenShift's cluster logging infrastructure using LokiStack for centralized log management and analysis.

## Test Overview

This test creates:
1. The `openshift-logging` namespace and a SeaweedFS instance for LokiStack object storage
2. A LokiStack instance for log storage and querying
3. An OpenTelemetry Collector that processes and exports logs to LokiStack
4. Log generation to test the end-to-end flow
5. Integration with OpenShift logging UI plugin

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Loki Operator installed
- Cluster Observability Operator installed (the logging UI plugin step looks up its namespace; the plugin itself is only for debugging)
- `logcli` installed (used to query the logs)
- `oc` CLI tool configured
- Appropriate cluster permissions

The Red Hat OpenShift Logging Operator is not required.

## Configuration Resources

### SeaweedFS Object Storage

Deploy SeaweedFS for LokiStack storage backend. The image is a multi-arch (amd64, arm64, ppc64le, s390x) build of the upstream SeaweedFS release, pinned by digest:

**Configuration:** [`install-seaweedfs.yaml`](./install-seaweedfs.yaml)

Creates SeaweedFS infrastructure:
- 2Gi PersistentVolumeClaim for storage
- SeaweedFS deployment (`weed mini`) with demo credentials (loki/supersecret) and a pre-created `loki` bucket
- Service for internal cluster access
- Secret with S3-compatible access configuration

The `openshift-logging` namespace is created by the test ([`namespace.yaml`](./namespace.yaml)) so that the Red Hat OpenShift Logging Operator is not needed.

### LokiStack Instance

Deploy LokiStack for log storage:

**Configuration:** [`install-loki.yaml`](./install-loki.yaml)

Creates a LokiStack with:
- 1x.demo size for testing environments
- S3-compatible storage via SeaweedFS
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

1. **Create the namespace and install SeaweedFS for object storage:**
   ```bash
   oc apply -f namespace.yaml
   oc apply -f install-seaweedfs.yaml
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
- **SeaweedFS**: Object storage backend with `loki` bucket
- **PVC**: 2Gi persistent volume for SeaweedFS storage
- **Secret**: `logging-loki-s3` with SeaweedFS access credentials

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

# Check SeaweedFS deployment
oc get deployment seaweedfs -o yaml

# View LokiStack gateway service
oc get svc logging-loki-gateway-http

# Check collector service account permissions
oc auth can-i create application --as=system:serviceaccount:openshift-logging:otel-collector-deployment

# Port forward to SeaweedFS and list the bucket (Loki writes chunks after they are flushed)
oc port-forward svc/seaweedfs 8333:8333 &
curl --aws-sigv4 "aws:amz:us-east-1:s3" --user loki:supersecret "http://localhost:8333/loki?list-type=2"  # Using demo test credentials

# Check collector metrics
oc port-forward svc/otel-collector 8888:8888 &
curl http://localhost:8888/metrics | grep otelcol_exporter
```

## Verification

The test verifies:
- ✅ SeaweedFS is deployed and accessible as object storage
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
- **Object Storage**: SeaweedFS backend for log persistence
- **Authentication**: Bearer token authentication with service accounts
- **Log Processing**: Kubernetes attributes and log transformation
- **OTLP Protocol**: Uses OTLP HTTP for log export to LokiStack
- **Multi-Pipeline**: Separate pipelines for LokiStack export and debug output
- **UI Integration**: Logging console plugin for log visualization

## Configuration Notes

- LokiStack runs in `1x.demo` size for testing environments
- SeaweedFS stores its data on a 2Gi PersistentVolumeClaim that is removed with the test and uses demo credentials (loki/supersecret) - **FOR TESTING ONLY**
- Collector uses bearer token authentication for LokiStack access
- Service CA certificate is used for TLS communication with LokiStack
- Log processors add Kubernetes metadata and normalize severity levels
- The application log type is set for proper tenant routing in LokiStack 