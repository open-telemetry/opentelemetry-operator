# Platform Monitoring Integration Test

This test demonstrates how to integrate OpenTelemetry Collector with OpenShift's platform monitoring stack by exposing collector metrics for Prometheus scraping.

## Test Overview

This test creates:
1. A cluster monitoring configuration to enable user workload monitoring
2. An OpenTelemetry Collector with metrics observability enabled
3. A Prometheus exporter that exposes metrics for monitoring
4. A telemetry generator to create test traffic and metrics

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Cluster Monitoring Configuration

Enable user workload monitoring in the cluster:

**Configuration:** [`00-workload-monitoring.yaml`](./00-workload-monitoring.yaml)

Enables user workload monitoring by:
- Setting `enableUserWorkload: true` in cluster monitoring config
- Allowing user-defined alerting rules

### OpenTelemetry Collector

Deploy a collector with observability metrics enabled:

**Configuration:** [`01-otel-collector.yaml`](./01-otel-collector.yaml)

Deploys a main collector with:
- Observability metrics enabled for internal monitoring
- OTLP receivers for HTTP and gRPC protocols
- Memory limiter processor for resource protection
- Debug exporters for traces, logs, and metrics

### Telemetry Generator

Deploy a workload that generates telemetry data:

**Configuration:** [`02-generate-telemetry.yaml`](./02-generate-telemetry.yaml)

Creates a deployment that:
- Generates test metrics continuously
- Sends data to the main collector via OTLP HTTP
- Includes Prometheus scraping annotations

### Monitoring Roles

Create ClusterRole and binding for metrics collection:

**Configuration:** [`03-create-monitoring-roles.yaml`](./03-create-monitoring-roles.yaml)

Configures RBAC permissions for:
- Service discovery and endpoint access
- Pod and namespace monitoring
- Metrics collection across the cluster

### Prometheus Exporter Collector

Deploy an additional collector with Prometheus exporter:

**Configuration:** [`04-use-prometheus-exporter.yaml`](./04-use-prometheus-exporter.yaml)

Creates a specialized collector that:
- Receives OTLP metrics
- Exposes metrics in Prometheus format on port 8889
- Includes resource attribute conversion to metric labels

## Deployment Steps

1. **Enable user workload monitoring:**
   ```bash
   oc apply -f 00-workload-monitoring.yaml
   ```

2. **Deploy main OpenTelemetry Collector:**
   ```bash
   oc apply -f 01-otel-collector.yaml
   ```

3. **Deploy telemetry generator:**
   ```bash
   oc apply -f 02-generate-telemetry.yaml
   ```

4. **Create monitoring roles:**
   ```bash
   oc apply -f 03-create-monitoring-roles.yaml
   ```

5. **Deploy Prometheus exporter collector:**
   ```bash
   oc apply -f 04-use-prometheus-exporter.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Cluster Monitoring
- **ConfigMap**: `cluster-monitoring-config` in `openshift-monitoring` namespace
- **Status**: User workload monitoring enabled

### Main Collector
- **Name**: `cluster-collector-collector`
- **Mode**: Deployment with observability metrics enabled
- **Pipelines**: Traces, logs, and metrics to debug exporter

### Prometheus Exporter Collector
- **Name**: `prometheus-exporter-collector`
- **Mode**: Deployment with Prometheus metrics endpoint
- **Endpoint**: Port 8889 with metrics in Prometheus format

### Telemetry Generator
- **Name**: `telemetry-generator`
- **Behavior**: Generates metrics and sends to main collector
- **Annotations**: Prometheus scraping annotations

## Testing the Configuration

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_metrics.sh`](./check_metrics.sh)

The script verifies:
- Main collector internal metrics endpoint (port 8888)
- Prometheus exporter metrics endpoint (port 8889)
- Presence of OpenTelemetry collector metrics
- Metrics format compatibility with Prometheus

## Verification

The test verifies:
- ✅ User workload monitoring is enabled in cluster configuration
- ✅ Main OpenTelemetry Collector deployment is ready
- ✅ Collector observability metrics are enabled
- ✅ Telemetry generator is running and sending data
- ✅ Monitoring RBAC roles are created
- ✅ Prometheus exporter collector is deployed
- ✅ Metrics are exposed on Prometheus endpoint (port 8889)
- ✅ Collector internal metrics are accessible (port 8888)

## Key Features

- **Platform Integration**: Integrates with OpenShift's native monitoring stack
- **Dual Metrics Sources**: Both collector internal metrics and application metrics
- **Prometheus Format**: Metrics exposed in Prometheus scraping format
- **Resource Conversion**: Resource attributes converted to metric labels
- **Monitoring Annotations**: Prometheus scraping annotations for auto-discovery
- **RBAC Configuration**: Proper permissions for metrics collection

## Configuration Notes

- User workload monitoring must be enabled for custom metrics scraping
- The main collector uses `observability.metrics.enableMetrics: true` for internal metrics
- Prometheus exporter collector runs separately to expose application metrics
- Metrics are exposed on port 8889 with Prometheus format
- Resource attributes are converted to telemetry labels for better context
- Telemetry generator creates test metrics to validate the pipeline 