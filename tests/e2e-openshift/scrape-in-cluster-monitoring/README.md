# Scrape In-Cluster Monitoring Test

This test demonstrates how to configure an OpenTelemetry Collector to scrape metrics from OpenShift's built-in Prometheus monitoring system using federation.

## Test Overview

This test creates an OpenTelemetry Collector that federates metrics from OpenShift's monitoring stack. It uses the Prometheus receiver to scrape specific metrics (`kube_namespace_labels`) from the cluster monitoring Prometheus instance and exports them via debug exporter.

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Cluster Role Binding

The test creates a cluster role binding to allow the collector to access monitoring data:

**Configuration:** [`create-clusterrolebinding.yaml`](./create-clusterrolebinding.yaml)

Grants the collector service account:
- `cluster-monitoring-view` role for accessing Prometheus federation endpoint
- Permissions to read metrics from the OpenShift monitoring stack

### OpenTelemetry Collector Configuration

The test deploys an OpenTelemetry Collector that federates cluster metrics:

**Configuration:** [`create-otel-instance.yaml`](./create-otel-instance.yaml)

Configures a collector that:
- Uses Prometheus receiver with federation endpoint
- Connects to OpenShift's built-in Prometheus via TLS
- Scrapes `kube_namespace_labels` metrics specifically
- Uses service account token for authentication
- Mounts CA bundle for secure communication with monitoring stack

## Deployment Steps

1. **Apply the RBAC configuration:**
   ```bash
   oc apply -f create-clusterrolebinding.yaml
   ```

2. **Apply the OpenTelemetry Collector configuration:**
   ```bash
   oc apply -f create-otel-instance.yaml
   ```

3. **Verify the deployment is ready:**
   ```bash
   oc wait --for=condition=ready pod -l app.kubernetes.io/name=otel-collector --timeout=300s
   ```

## Expected Resources

The test creates and verifies these resources:

### Deployment
- **Name**: `otel-collector`
- **Status**: Ready replica available

### Service
- **Name**: `otel-collector`
- **Ports**: Exposes collector service endpoints

## Testing the Configuration

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_logs.sh`](./check_logs.sh)

The script verifies:
- Collector successfully connects to OpenShift Prometheus
- `kube_namespace_labels` metrics are being scraped and processed
- Federation endpoint is accessible with proper authentication
- Metrics appear in collector debug output logs

## Verification

The test verifies:
- ✅ Cluster role binding is created
- ✅ ConfigMap with CA bundle is created and populated
- ✅ OpenTelemetry Collector deployment is ready
- ✅ Collector service is created
- ✅ Collector successfully scrapes `kube_namespace_labels` metric from Prometheus

## Key Features

- **Prometheus Federation**: Uses Prometheus receiver to federate metrics from cluster monitoring
- **TLS Authentication**: Properly configured TLS with service CA certificate
- **Bearer Token Auth**: Uses service account token for authentication
- **Selective Scraping**: Only scrapes specific metrics (`kube_namespace_labels`)
- **RBAC Integration**: Uses `cluster-monitoring-view` role for proper permissions

## Configuration Notes

- The collector targets the federate endpoint of OpenShift's Prometheus
- `honor_labels` is set to `false` due to a known bug in the Prometheus receiver
- TLS configuration uses the injected service CA bundle
- Bearer token authentication uses the mounted service account token
- Only the `kube_namespace_labels` metric is scraped for testing purposes 