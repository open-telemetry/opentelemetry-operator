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

The test creates a simple Tempo monolithic instance:

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: otlpmetrics
  namespace: chainsaw-otlp-metrics
spec: {}
```

### Workload Monitoring Configuration

Enable user workload monitoring to scrape application metrics:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-workload-monitoring-config
  namespace: openshift-user-workload-monitoring
data:
  config.yaml: |
    prometheus:
      logLevel: debug
      retention: 15d
```

### OpenTelemetry Collector

The collector receives OTLP data and routes traces to Tempo and metrics to Prometheus:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: cluster-collector
  namespace: chainsaw-otlp-metrics
spec:
  mode: deployment
  observability:
    metrics:
      enableMetrics: true
  config:
    receivers:
      otlp:
        protocols:
          grpc: {}
          http:  {}
    processors: {}
    exporters:
      otlp:
        endpoint: tempo-otlpmetrics.chainsaw-otlp-metrics.svc:4317
        tls:
          insecure: true
      prometheus:
        endpoint: 0.0.0.0:8889
        resource_to_telemetry_conversion:
          enabled: true # by default resource attributes are dropped
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [otlp]
        metrics:
          receivers: [otlp]
          exporters: [prometheus]
```

### Trace Generator Application

The test deploys a trace generator to create test telemetry data:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: traces-generator
  namespace: chainsaw-otlp-metrics
  labels:
    app: traces-generator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: traces-generator
  template:
    metadata:
      labels:
        app: traces-generator
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        - name: traces-generator
          image: ghcr.io/pavolloffay/traces-generator:latest
          ports:
            - containerPort: 8080
              name: metrics
          env:
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "http://cluster-collector-collector:4318"
            - name: OTEL_EXPORTER_OTLP_PROTOCOL
              value: "http/protobuf"
            - name: OTEL_SERVICE_NAME
              value: "traces-generator"
          args:
            - "--workers=1"
            - "--traces=1"
            - "--push-gateway-url=http://cluster-collector-collector:4318/v1/traces"
```

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

### Check Metrics Script (`check_metrics.sh`)

```bash
#!/bin/bash
set -ex

# Check if metrics are being exposed
oc port-forward -n chainsaw-otlp-metrics svc/cluster-collector-collector 8889:8889 &
PORT_FORWARD_PID=$!
sleep 5

# Test metrics endpoint
curl -f http://localhost:8889/metrics | grep -E "(traces_generator|otelcol_)"

kill $PORT_FORWARD_PID
```

### Trace Verification Job

The test deploys a verification job that queries Tempo for traces:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: verify-traces
  namespace: chainsaw-otlp-metrics
spec:
  template:
    spec:
      containers:
      - name: verify
        image: curlimages/curl:latest
        command:
        - /bin/sh
        - -c
        - |
          set -ex
          # Wait for traces to be ingested
          sleep 30
          
          # Query Tempo for traces
          TEMPO_ENDPOINT="http://tempo-otlpmetrics:3200"
          
          # Search for traces
          curl -s "${TEMPO_ENDPOINT}/api/search" | grep -q "traces"
          echo "Traces found in Tempo!"
      restartPolicy: Never
```

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