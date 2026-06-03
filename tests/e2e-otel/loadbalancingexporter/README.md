# OpenTelemetry Load Balancing Exporter Test

This test demonstrates the OpenTelemetry Load Balancing exporter configuration for distributing traces across multiple backend collectors.

## 🎯 What This Test Does

The test validates that the Load Balancing exporter can:
- Use Kubernetes service discovery to find backend collector endpoints
- Route traces based on service name attributes for consistent routing
- Distribute traces across 5 backend collector replicas
- Ensure each service's traces go to only one backend pod (routing consistency)

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Backend Collector Configuration
- **File**: [`otel-loadbalancingexporter-backends.yaml`](./otel-loadbalancingexporter-backends.yaml)
- **Contains**: OpenTelemetryCollector deployment with 5 replicas
- **Key Features**:
  - Multiple replica backend collectors for load distribution
  - OTLP receiver for incoming traces
  - Debug exporter for trace verification
  - Headless service for service discovery

### 2. Load Balancing Collector Configuration
- **File**: [`otel-loadbalancingexporter-lb.yaml`](./otel-loadbalancingexporter-lb.yaml)
- **Contains**: ServiceAccount, Role, RoleBinding, and Load Balancing OpenTelemetryCollector
- **Key Features**:
  - Load balancing exporter with Kubernetes service discovery
  - RBAC permissions for endpoint discovery
  - Service-based routing for consistent load distribution
  - OTLP receiver for trace ingestion

### 3. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating test traces with different service names
- **Key Features**:
  - Generates traces for three different services: blue, red, green
  - Uses telemetrygen tool for trace generation
  - Targets the load balancing collector endpoint

### 4. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that load balancing works correctly
- **Verification Criteria**:
  - All three service names appear across backend pods
  - Each service name appears in only one backend pod
  - Routing consistency is maintained per service

### 5. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Backend Collectors** - Deploy from [`otel-loadbalancingexporter-backends.yaml`](./otel-loadbalancingexporter-backends.yaml)
2. **Create Load Balancing Collector** - Deploy from [`otel-loadbalancingexporter-lb.yaml`](./otel-loadbalancingexporter-lb.yaml)
3. **Generate Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
4. **Verify Load Balancing** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Load Balancing Configuration

### Resolver Configuration:
- **Type**: Kubernetes service discovery
- **Service**: `chainsaw-lb-backends-collector-headless.chainsaw-lb`
- **Discovery**: Uses headless service to find all backend pod IPs

### Routing Configuration:
- **Routing Key**: `"service"` - Routes based on service name attribute
- **Protocol**: OTLP with insecure TLS
- **Consistency**: Same service always routes to the same backend

### Backend Discovery:
- Automatically discovers backend endpoints using Kubernetes service resolution
- Monitors endpoint changes for dynamic backend scaling
- Requires RBAC permissions to list and watch endpoints

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Checks that all three required service names are present across backend pods
- Validates each service name appears in only one backend pod (routing consistency)
- Ensures no service appears in multiple pods (proper load balancing)
- Confirms load balancing distribution across the 5 backend replicas

**Expected Services:**
- `telemetrygen-http-blue`
- `telemetrygen-http-red`
- `telemetrygen-http-green`

## 🧹 Cleanup

The test runs in the `chainsaw-lb` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses Kubernetes service discovery for dynamic backend endpoint resolution
- Routes based on service name attribute for consistent routing per service
- Requires specific RBAC permissions for endpoint discovery
- Demonstrates horizontal scaling of trace processing across multiple collectors
- Validates routing consistency ensuring traces from the same service go to the same backend
- Uses headless service to enable direct pod-to-pod communication