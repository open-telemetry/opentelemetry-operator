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

**Configuration:** [`install-target-allocator.yaml`](./install-target-allocator.yaml)

Configures Target Allocator permissions by:
- Creating service account `ta` with cluster access
- Defining ClusterRole `smoke-targetallocator` for pod and namespace discovery
- Binding the role to the service account for target allocation functionality

### OpenTelemetry Collector with Target Allocator

**Configuration:** [`install-target-allocator.yaml`](./install-target-allocator.yaml)

Deploys a StatefulSet collector that:
- Runs in StatefulSet mode for persistent target allocation
- Enables Target Allocator with service account `ta`
- Includes Jaeger receiver for trace collection
- Configures Prometheus receiver for metrics scraping
- Uses debug exporter for trace verification

### Sample Application

**Configuration:** [`install-app.yaml`](./install-app.yaml)

Deploys a sample application that:
- Creates a simple nginx-based application with metrics endpoint
- Exposes Prometheus-compatible metrics on `/metrics` endpoint
- Includes Prometheus scraping annotations for auto-discovery
- Provides sample counter and histogram metrics for testing

### Collector Sidecar Configuration

**Configuration:** [`install-collector-sidecar.yaml`](./install-collector-sidecar.yaml)

Deploys an additional collector that:
- Runs in sidecar mode for pod-level trace collection
- Includes Jaeger gRPC receiver for trace ingestion
- Uses debug exporter for trace verification
- Demonstrates sidecar deployment pattern

### Instrumentation Configuration

**Configuration:** [`install-instrumentation.yaml`](./install-instrumentation.yaml)

Configures automatic instrumentation that:
- Enables Java auto-instrumentation with debug logging
- Sends traces to the stateful collector via OTLP HTTP
- Uses trace context and baggage propagators
- Includes JDBC and Kafka instrumentation
- Applies 25% trace sampling rate

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

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_must_gather.sh`](./check_must_gather.sh)

The script verifies:
- Must-gather collection completes successfully
- OpenTelemetry CRDs are included in the collection
- Collector logs and configurations are captured
- Target allocator information is present
- Required diagnostic files are collected

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