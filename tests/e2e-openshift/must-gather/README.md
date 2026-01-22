# Must-Gather and Target Allocator Test

This test demonstrates OpenTelemetry Target Allocator functionality and diagnostic collection using OpenShift's must-gather capability for troubleshooting and monitoring OpenTelemetry components.

## Test Overview

This test creates:
1. An OpenTelemetry Collector with Target Allocator enabled
2. RBAC configuration for the Target Allocator
3. A sample application deployment for telemetry collection
4. A gather collector for trace collection in deployment mode
5. Instrumentation configuration for automatic telemetry injection
6. Must-gather diagnostic collection verification

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

### Gather Collector Configuration

**Configuration:** [`install-collector-gather.yaml`](./install-collector-gather.yaml)

Deploys an additional collector that:
- Runs in deployment mode for trace collection
- Includes OTLP gRPC and HTTP receivers for trace ingestion
- Uses debug exporter for trace verification
- Provides a central endpoint for telemetry gathering

### Instrumentation Configuration

**Configuration:** [`install-instrumentation.yaml`](./install-instrumentation.yaml)

Configures automatic instrumentation that:
- Enables Node.js auto-instrumentation with debug logging
- Sends traces to the gather collector via OTLP gRPC endpoint
- Uses Jaeger and B3 propagators
- Exports metrics to Prometheus
- Applies 25% trace sampling rate

## Deployment Steps

1. **Deploy Target Allocator with RBAC:**
   ```bash
   oc apply -f install-target-allocator.yaml
   ```

2. **Install gather collector:**
   ```bash
   oc apply -f install-collector-gather.yaml
   ```

3. **Configure instrumentation:**
   ```bash
   oc apply -f install-instrumentation.yaml
   ```

4. **Deploy sample application:**
   ```bash
   oc apply -f install-app.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Target Allocator
- **ServiceAccount**: `ta` with cluster permissions
- **ClusterRole**: `smoke-targetallocator` for pod and namespace access
- **Collector**: `stateful-collector` in StatefulSet mode
- **Target Allocator**: Enabled with Prometheus scraping

### Sample Application
- **Deployment**: `my-nodejs` with Node.js application
- **Service**: `my-nodejs-service` exposing HTTP port
- **Instrumentation**: Automatic Node.js telemetry injection

### Gather Collector
- **Deployment**: `gather-collector` for centralized trace collection
- **Service**: `gather-collector` exposing OTLP gRPC and HTTP endpoints
- **Instrumentation**: `nodejs-instrumentation` for automatic code injection

## Testing the Configuration

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_must_gather.sh`](./check_must_gather.sh)

The script verifies:
- Must-gather collection completes successfully
- OpenTelemetry CRDs are included in the collection
- Collector logs and configurations are captured
- Target allocator information is present
- Gather collector deployment and services are collected
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
- ✅ Sample Node.js application is deployed with auto-instrumentation
- ✅ Gather collector is deployed successfully in deployment mode
- ✅ Instrumentation configuration is applied for Node.js
- ✅ Must-gather collects OpenTelemetry diagnostic information
- ✅ Must-gather includes collector logs and configurations
- ✅ Target allocation is working for Prometheus scraping

## Key Features

- **Target Allocator**: Automatically distributes Prometheus scraping targets across collector instances
- **StatefulSet Mode**: Collector runs as StatefulSet for persistent target allocation
- **RBAC Integration**: Proper permissions for target discovery and allocation
- **Must-Gather Support**: Diagnostic collection for troubleshooting
- **Deployment Mode**: Central gather collector for trace collection
- **Auto-Instrumentation**: Automatic Node.js telemetry injection for applications
- **OTLP Protocol**: Modern telemetry data transmission using OTLP gRPC/HTTP 