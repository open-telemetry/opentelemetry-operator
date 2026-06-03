# OpenTelemetry FileLog Receiver Test

This test demonstrates the OpenTelemetry FileLog receiver configuration for collecting logs from Kubernetes pods.

## 🎯 What This Test Does

The test validates that the FileLog receiver can:
- Collect logs from all pods in the cluster using a DaemonSet deployment
- Parse container logs using the container operator
- Export collected logs to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: SecurityContextConstraints (SCC) and OpenTelemetryCollector configuration
- **Key Features**:
  - DaemonSet deployment mode for cluster-wide log collection
  - FileLog receiver with path includes/excludes for filtering system namespaces
  - Container operator for parsing Kubernetes container logs
  - Debug exporter for verification
  - Security constraints for OpenShift compliance

### 2. Test Application (Log Generator)
- **File**: [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)  
- **Contains**: ConfigMap and ReplicationController for log generation
- **Key Features**:
  - Generates plaintext logs at 60 messages per second
  - Uses sidecar injection for OpenTelemetry integration
  - Includes proper security context and resource constraints

### 3. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that logs are properly collected and processed
- **Verification Criteria**:
  - `log.file.path` - File path is included in logs
  - `SVTLogger` - Application log content  
  - `Body: Str(.*SVTLogger.*app-log-plaintext-` - Structured log format
  - `k8s.container.name: Str(app-log-plaintext)` - Container name attribute
  - `k8s.namespace.name: Str(chainsaw-filelog)` - Namespace attribute

### 4. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
2. **Create Test Application** - Deploy from [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)  
3. **Verify Log Collection** - Run [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Continuously monitors OpenTelemetry collector pod logs
- Searches for specific log indicators to confirm proper processing
- Validates that container operator correctly parses Kubernetes metadata
- Ensures FileLog receiver captures application logs with proper attributes

## 🧹 Cleanup

The test runs in the `chainsaw-filelog` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses DaemonSet mode to collect logs from all nodes
- Excludes system namespaces (kube-*, openshift-*, default)
- Includes file path and uses container operator for parsing
- Runs with minimal privileges using SecurityContextConstraints
- Mounts `/var/log/pods` as read-only from the host 