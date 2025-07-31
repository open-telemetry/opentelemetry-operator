# OpenShift Route Integration Test

This test demonstrates how to expose OpenTelemetry Collector endpoints externally using OpenShift Routes.

## Test Overview

This test creates a simple OpenTelemetry Collector deployment with OTLP receivers and exposes the endpoints via OpenShift Routes with insecure termination. The test verifies that the routes are created correctly and that telemetry data can be sent to the collector through the route.

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### OpenTelemetry Collector with Route Ingress

The test deploys an OpenTelemetry Collector with external access via Routes:

**Configuration:** [`00-install.yaml`](./00-install.yaml)

Creates a collector that:
- Exposes OTLP endpoints (HTTP and gRPC) via OpenShift Routes
- Uses insecure termination for testing purposes
- Includes custom annotations for route configuration
- Routes traces to debug exporter for verification

## Deployment Steps

1. **Apply the OpenTelemetry Collector configuration:**
   ```bash
   oc apply -f 00-install.yaml
   ```

2. **Verify the deployment is ready:**
   ```bash
   oc wait --for=condition=ready pod -l app.kubernetes.io/name=simplest-collector --timeout=300s
   ```

3. **Check that routes are created:**
   ```bash
   oc get routes
   ```

## Expected Resources

The test creates and verifies these resources:

### Deployment
- **Name**: `simplest-collector`
- **Status**: 1 ready replica

### Routes Created by Operator
Two routes are automatically created:

#### OTLP gRPC Route
```yaml
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  annotations:
    something.com: "true"
  labels:
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/name: otlp-grpc-simplest-route
  name: otlp-grpc-simplest-route
spec:
  port:
    targetPort: otlp-grpc
  to:
    kind: Service
    name: simplest-collector
  wildcardPolicy: None
```

#### OTLP HTTP Route
```yaml
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  annotations:
    something.com: "true"
  labels:
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/name: otlp-http-simplest-route
  name: otlp-http-simplest-route
spec:
  port:
    targetPort: otlp-http
  to:
    kind: Service
    name: simplest-collector
  wildcardPolicy: None
```

## Testing the Route

The test verifies the HTTP route works by sending telemetry data:

1. **Get the route hostname:**
   ```bash
   otlp_http_host=$(oc get route otlp-http-simplest-route -o jsonpath='{.spec.host}')
   ```

2. **Send test telemetry data:**
   ```bash
   curl --fail -ivX POST http://${otlp_http_host}:80/v1/traces \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

## Verification

The test verifies:
- ✅ Deployment has 1 ready replica
- ✅ Two routes are created (HTTP and gRPC)
- ✅ Route annotations are properly applied
- ✅ Routes point to the correct service
- ✅ HTTP route accepts telemetry data with 2xx response

## Key Features

- **Route Annotations**: Custom annotations (`something.com: "true"`) are applied to routes
- **Insecure Termination**: Routes use insecure termination for simplicity
- **Dual Protocol Support**: Both HTTP and gRPC OTLP endpoints are exposed
- **Automatic Route Management**: Routes are automatically created and managed by the operator

## Configuration Notes

- Routes use `insecure` termination - for secure communication, use TLS termination
- The collector uses a simple debug exporter to log received telemetry
- Both HTTP and gRPC OTLP protocols are enabled on default ports 