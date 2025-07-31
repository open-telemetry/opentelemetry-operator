# OTLP Metrics and Traces Test

This test demonstrates how to deploy an OpenTelemetry Collector that receives OTLP metrics and traces, exports traces to Tempo, and exposes metrics via Prometheus endpoint.

## Test Overview

This test creates:
1. A Tempo instance for trace storage
2. A workload monitoring configuration to enable user workload monitoring
3. An OpenTelemetry Collector that receives OTLP data and splits it between Tempo (traces) and Prometheus (metrics)
4. A trace generator application to create test telemetry data

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Tempo Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Tempo Instance

**Configuration:** [`00-install-tempo.yaml`](./00-install-tempo.yaml)

Creates a simple Tempo monolithic instance that:
- Provides trace storage backend for the collector
- Runs in the test namespace for trace persistence
- Uses default configuration for simplicity

### Workload Monitoring Configuration

**Configuration:** [`01-workload-monitoring.yaml`](./01-workload-monitoring.yaml)

Enables user workload monitoring by:
- Configuring Prometheus with debug logging
- Setting 15-day retention period for metrics
- Allowing scraping of user application metrics

### OpenTelemetry Collector

**Configuration:** [`02-otel-metrics-collector.yaml`](./02-otel-metrics-collector.yaml)

Deploys a collector that:
- Receives OTLP data via HTTP and gRPC protocols
- Routes traces to Tempo via OTLP exporter
- Exposes metrics via Prometheus endpoint on port 8889
- Enables resource attribute to telemetry conversion
- Includes observability metrics for monitoring

### Trace Generator Application

**Configuration:** [`03-metrics-traces-gen.yaml`](./03-metrics-traces-gen.yaml)

Deploys a trace generator that:
- Continuously generates test traces and metrics
- Sends telemetry data to the collector via OTLP HTTP
- Includes Prometheus scraping annotations
- Configures single worker for controlled test traffic

## Deployment Steps

1. **Install Tempo instance:**
   ```bash
   oc apply -f 00-install-tempo.yaml
   ```

2. **Configure workload monitoring:**
   ```bash
   oc apply -f 01-workload-monitoring.yaml
   ```

3. **Deploy OpenTelemetry Collector:**
   ```bash
   oc apply -f 02-otel-metrics-collector.yaml
   ```

4. **Deploy trace generator:**
   ```bash
   oc apply -f 03-metrics-traces-gen.yaml
   ```

5. **Verify trace generation:**
   ```bash
   oc apply -f verify-traces.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Tempo Instance
- **Name**: `tempo-otlpmetrics`
- **Status**: Ready and accepting traces

### OpenTelemetry Collector
- **Name**: `cluster-collector-collector`
- **Status**: Deployment ready with 1 replica
- **Service**: Exposes OTLP HTTP (4318) and gRPC (4317) endpoints
- **Metrics**: Prometheus endpoint on port 8889

### Trace Generator
- **Name**: `traces-generator`
- **Status**: Deployment ready with 1 replica
- **Behavior**: Continuously generates and sends traces to the collector

## Testing the Configuration

The test includes verification scripts to ensure proper operation:

**Verification Script:** [`check_metrics.sh`](./check_metrics.sh)

The script verifies:
- Collector metrics endpoint is accessible on port 8889
- Metrics include both application and collector internal metrics
- Prometheus format compatibility

### Trace Verification Job

**Configuration:** [`verify-traces.yaml`](./verify-traces.yaml)

Deploys a verification job that:
- Waits for traces to be ingested into Tempo
- Queries Tempo's search API for stored traces
- Validates that traces are successfully persisted
- Provides confirmation of end-to-end trace flow

## Verification

The test verifies:
- ✅ Tempo instance is ready and accessible
- ✅ User workload monitoring is configured
- ✅ OpenTelemetry Collector deployment is ready
- ✅ Collector service endpoints are accessible
- ✅ Trace generator is running and sending data
- ✅ Metrics are exposed on Prometheus endpoint
- ✅ Traces are stored in Tempo
- ✅ Collector internal metrics are available

## Key Features

- **Dual Pipeline Architecture**: Separate pipelines for traces (to Tempo) and metrics (to Prometheus)
- **OTLP Protocol Support**: Supports both HTTP and gRPC OTLP protocols
- **Resource Attribute Conversion**: Converts resource attributes to metric labels
- **Observability Enabled**: Collector exposes its own metrics for monitoring
- **Insecure TLS**: Uses insecure TLS for internal communication (test environment)

## Configuration Notes

- The collector runs in deployment mode (stateless)
- Traces are exported to Tempo via OTLP gRPC on port 4317
- Metrics are exposed via Prometheus exporter on port 8889
- The trace generator continuously creates test telemetry data
- User workload monitoring must be enabled to scrape application metrics 