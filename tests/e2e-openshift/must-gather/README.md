# Must-Gather and Target Allocator Test

This test demonstrates OpenTelemetry Target Allocator functionality and diagnostic collection using OpenShift's must-gather capability for troubleshooting and monitoring OpenTelemetry components.

## Test Overview

This test creates:
1. An OpenTelemetry Collector with Target Allocator enabled
2. RBAC configuration for the Target Allocator
3. A sample application deployment for telemetry collection
4. Instrumentation configuration for automatic telemetry injection
5. Must-gather diagnostic collection verification

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Service Account and RBAC

Target Allocator requires specific permissions to discover and allocate targets:

```yaml
apiVersion: v1
automountServiceAccountToken: true
kind: ServiceAccount
metadata:
  name: ta
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: smoke-targetallocator
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - namespaces
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: (join('-', ['default-view', $namespace]))
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: smoke-targetallocator
subjects:
- kind: ServiceAccount
  name: ta
  namespace: ($namespace)
```

### OpenTelemetry Collector with Target Allocator

Deploy a StatefulSet collector with Target Allocator enabled:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: stateful
spec:
  config:
    receivers:
      jaeger:
          protocols:
            grpc: {}

      # Collect own metrics
      prometheus:
        config:
          global:
            scrape_interval: 30s
            scrape_protocols: ['PrometheusProto','OpenMetricsText1.0.0','OpenMetricsText0.0.1','PrometheusText0.0.4']
          scrape_configs:
            - job_name: 'otel-collector'
              scrape_interval: 10s
              static_configs:
                - targets: [ '0.0.0.0:8888' ]

    processors: {}

    exporters:
      debug: {}
    service:
      pipelines:
        traces:
          receivers: [jaeger]
          exporters: [debug]
  mode: statefulset
  targetAllocator:
    enabled: true
    serviceAccount: ta
```

### Sample Application

Deploy a sample application to generate telemetry:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-app
  labels:
    app: sample-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sample-app
  template:
    metadata:
      labels:
        app: sample-app
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: app
        image: nginx:latest
        ports:
        - containerPort: 80
          name: http
        - containerPort: 8080
          name: metrics
        command:
        - /bin/sh
        - -c
        - |
          # Simple HTTP server that exposes metrics
          cat > /usr/share/nginx/html/metrics << 'EOF'
          # HELP sample_requests_total Total number of requests
          # TYPE sample_requests_total counter
          sample_requests_total{method="GET",status="200"} 42
          # HELP sample_request_duration_seconds Request duration in seconds
          # TYPE sample_request_duration_seconds histogram
          sample_request_duration_seconds_bucket{le="0.1"} 10
          sample_request_duration_seconds_bucket{le="0.5"} 15
          sample_request_duration_seconds_bucket{le="1.0"} 20
          sample_request_duration_seconds_bucket{le="+Inf"} 25
          sample_request_duration_seconds_sum 30.5
          sample_request_duration_seconds_count 25
          EOF
          nginx -g 'daemon off;'
---
apiVersion: v1
kind: Service
metadata:
  name: sample-app-service
  labels:
    app: sample-app
spec:
  selector:
    app: sample-app
  ports:
  - name: http
    port: 80
    targetPort: 80
  - name: metrics
    port: 8080
    targetPort: 8080
```

### Collector Sidecar Configuration

Deploy an additional collector as a sidecar:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: sidecar
spec:
  mode: sidecar
  config:
    receivers:
      jaeger:
        protocols:
          grpc:
    exporters:
      debug:
    service:
      pipelines:
        traces:
          receivers: [jaeger]
          exporters: [debug]
```

### Instrumentation Configuration

Configure automatic instrumentation for applications:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: java-instrumentation
spec:
  exporter:
    endpoint: http://stateful-collector:4318
  propagators:
    - tracecontext
    - baggage
  sampler:
    type: parentbased_traceidratio
    argument: "0.25"
  java:
    image: ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:latest
    env:
      - name: OTEL_JAVAAGENT_DEBUG
        value: "true"
      - name: OTEL_INSTRUMENTATION_JDBC_ENABLED
        value: "true"
      - name: OTEL_INSTRUMENTATION_KAFKA_ENABLED
        value: "true"
```

## Deployment Steps

1. **Deploy sample application:**
   ```bash
   oc apply -f install-app.yaml
   ```

2. **Install collector sidecar:**
   ```bash
   oc apply -f install-collector-sidecar.yaml
   ```

3. **Configure instrumentation:**
   ```bash
   oc apply -f install-instrumentation.yaml
   ```

4. **Deploy Target Allocator with RBAC:**
   ```bash
   oc apply -f install-target-allocator.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Target Allocator
- **ServiceAccount**: `ta` with cluster permissions
- **ClusterRole**: `smoke-targetallocator` for pod and namespace access
- **Collector**: `stateful-collector` in StatefulSet mode
- **Target Allocator**: Enabled with Prometheus scraping

### Sample Application
- **Deployment**: `sample-app` with metrics endpoint
- **Service**: `sample-app-service` exposing HTTP and metrics ports
- **Metrics**: Prometheus-compatible metrics on `/metrics` endpoint

### Instrumentation
- **Sidecar Collector**: `sidecar-collector` for trace collection
- **Instrumentation**: `java-instrumentation` for automatic code injection

## Testing the Configuration

The test includes a script (`check_must_gather.sh`) to verify must-gather functionality:

```bash
#!/bin/bash
set -ex

NAMESPACE=${NAMESPACE:-default}
MUST_GATHER_DIR="/tmp/must-gather-$(date +%s)"

echo "Starting must-gather collection for OpenTelemetry components..."

# Run must-gather with OpenTelemetry image
oc adm must-gather \
  --image=ghcr.io/openshift/opentelemetry-must-gather:latest \
  --dest-dir="$MUST_GATHER_DIR" \
  -- /usr/bin/gather

echo "Must-gather collection completed. Files saved to: $MUST_GATHER_DIR"

# Verify must-gather contents
verify_must_gather_contents() {
    local gather_dir="$1"
    local success=true
    
    echo "Verifying must-gather contents..."
    
    # Check for OpenTelemetry CRDs
    if find "$gather_dir" -name "*opentelemetrycollector*" -o -name "*instrumentation*" | grep -q .; then
        echo "✅ Found OpenTelemetry CRDs in must-gather"
    else
        echo "❌ OpenTelemetry CRDs not found in must-gather"
        success=false
    fi
    
    # Check for collector logs
    if find "$gather_dir" -name "*collector*log*" | grep -q .; then
        echo "✅ Found collector logs in must-gather"
    else
        echo "❌ Collector logs not found in must-gather"
        success=false
    fi
    
    # Check for target allocator logs
    if find "$gather_dir" -name "*target*allocator*" | grep -q .; then
        echo "✅ Found target allocator information in must-gather"
    else
        echo "❌ Target allocator information not found in must-gather"
        success=false
    fi
    
    # Check for instrumentation data
    if find "$gather_dir" -path "*opentelemetry*" -name "*yaml" | grep -q .; then
        echo "✅ Found OpenTelemetry YAML configurations in must-gather"
    else
        echo "❌ OpenTelemetry configurations not found in must-gather"
        success=false
    fi
    
    if [ "$success" = true ]; then
        echo "✅ Must-gather verification successful!"
        return 0
    else
        echo "❌ Must-gather verification failed!"
        return 1
    fi
}

# Verify the collected data
verify_must_gather_contents "$MUST_GATHER_DIR"

# Show structure for debugging
echo "Must-gather directory structure:"
find "$MUST_GATHER_DIR" -type f | head -20

# Cleanup
echo "Cleaning up must-gather directory..."
rm -rf "$MUST_GATHER_DIR"

echo "Must-gather test completed successfully!"
```

## Additional Verification Commands

Check Target Allocator status and functionality:

```bash
# Check Target Allocator pod status
oc get pods -l app.kubernetes.io/component=opentelemetry-targetallocator

# Check Target Allocator service
oc get svc -l app.kubernetes.io/component=opentelemetry-targetallocator

# View Target Allocator logs
oc logs -l app.kubernetes.io/component=opentelemetry-targetallocator

# Check target allocation via API
oc port-forward svc/stateful-targetallocator 8080:80 &
curl http://localhost:8080/targets
curl http://localhost:8080/jobs

# Check Prometheus configuration from Target Allocator
curl http://localhost:8080/config
```

## Verification

The test verifies:
- ✅ Service account and RBAC are configured for Target Allocator
- ✅ OpenTelemetry Collector with Target Allocator is deployed in StatefulSet mode
- ✅ Target Allocator pod is running and accessible
- ✅ Sample application is deployed with metrics endpoint
- ✅ Collector sidecar is deployed successfully
- ✅ Instrumentation configuration is applied
- ✅ Must-gather collects OpenTelemetry diagnostic information
- ✅ Must-gather includes collector logs and configurations
- ✅ Target allocation is working for Prometheus scraping

## Key Features

- **Target Allocator**: Automatically distributes Prometheus scraping targets across collector instances
- **StatefulSet Mode**: Collector runs as StatefulSet for persistent target allocation
- **RBAC Integration**: Proper permissions for target discovery and allocation
- **Must-Gather Support**: Diagnostic collection for troubleshooting
- **Sidecar Mode**: Additional collector deployment pattern
- **Auto-Instrumentation**: Automatic telemetry injection for applications
- **Prometheus Scraping**: Integration with Prometheus metrics collection 