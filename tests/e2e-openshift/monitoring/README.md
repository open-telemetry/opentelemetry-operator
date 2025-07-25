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

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true 
    alertmanagerMain:
      enableUserAlertmanagerConfig: true
```

### OpenTelemetry Collector

Deploy a collector with observability metrics enabled:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: cluster-collector
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
          http: {}
    processors:
      memory_limiter:
        check_interval: 1
        limit_percentage: 75
        spike_limit_percentage: 15
    exporters:
      debug:
        verbosity: detailed
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [debug]
        logs:
          receivers: [otlp]
          exporters: [debug]
        metrics:
          receivers: [otlp]
          exporters: [debug]
```

### Monitoring Roles

Create ClusterRole and binding for metrics collection:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: monitoring-prometheus-exporter
rules:
- apiGroups:
  - ""
  resources:
  - services
  - endpoints
  - pods
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: monitoring-prometheus-exporter
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: monitoring-prometheus-exporter
subjects:
- kind: ServiceAccount
  name: cluster-collector-collector
  namespace: default
```

### Prometheus Exporter Collector

Deploy an additional collector with Prometheus exporter:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: prometheus-exporter
spec:
  mode: deployment
  config:
    receivers:
      otlp:
        protocols:
          grpc: {}
          http: {}
    exporters:
      prometheus:
        endpoint: "0.0.0.0:8889"
        namespace: otel
        const_labels:
          cluster: openshift
        resource_to_telemetry_conversion:
          enabled: true
    service:
      pipelines:
        metrics:
          receivers: [otlp]
          exporters: [prometheus]
```

### Telemetry Generator

Deploy a workload that generates telemetry data:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telemetry-generator
  labels:
    app: telemetry-generator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telemetry-generator
  template:
    metadata:
      labels:
        app: telemetry-generator
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: generator
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.124.1
        ports:
        - containerPort: 8080
          name: metrics
        env:
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://cluster-collector-collector:4318"
        - name: OTEL_EXPORTER_OTLP_PROTOCOL
          value: "http/protobuf"
        - name: OTEL_SERVICE_NAME
          value: "telemetry-generator"
        args:
        - metrics
        - --otlp-endpoint=http://cluster-collector-collector:4318
        - --metrics=100
        - --otlp-http
        - --otlp-insecure=true
        - --rate=10
        command: ["/telemetrygen"]
```

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

The test includes a script (`check_metrics.sh`) to verify metrics are exposed:

```bash
#!/bin/bash
set -ex

# Function to check metrics endpoint
check_metrics_endpoint() {
    local service="$1"
    local port="$2"
    local path="$3"
    
    echo "Checking metrics endpoint: $service:$port$path"
    
    # Port forward to the service
    oc port-forward "svc/$service" "$port:$port" &
    PF_PID=$!
    sleep 5
    
    # Check metrics endpoint
    if curl -f "http://localhost:$port$path" | grep -E "(otelcol_|otel_)"; then
        echo "✅ Metrics found on $service:$port$path"
        kill $PF_PID 2>/dev/null || true
        return 0
    else
        echo "❌ No metrics found on $service:$port$path"
        kill $PF_PID 2>/dev/null || true
        return 1
    fi
}

# Check main collector internal metrics (if enabled)
check_metrics_endpoint "cluster-collector-collector" "8888" "/metrics"

# Check Prometheus exporter metrics
check_metrics_endpoint "prometheus-exporter-collector" "8889" "/metrics"
```

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