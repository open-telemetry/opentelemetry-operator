# OpenTelemetry Kubernetes Cluster Receiver Test

This test demonstrates the OpenTelemetry Kubernetes Cluster receiver configuration for collecting cluster-level metrics from Kubernetes.

## 🎯 What This Test Does

The test validates that the Kubernetes Cluster receiver can:
- Collect cluster-level metrics about nodes, pods, and other Kubernetes resources
- Access Kubernetes API to gather resource information
- Export collected metrics to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-k8sclusterreceiver.yaml`](./otel-k8sclusterreceiver.yaml)
- **Contains**: OpenTelemetryCollector with Kubernetes Cluster receiver and RBAC
- **Key Features**:
  - Single collector instance for cluster-wide metrics
  - Comprehensive RBAC permissions for Kubernetes API access
  - Cluster receiver with 10-second collection intervals
  - Access to nodes, pods, deployments, services, and other resources
  - Debug exporter for verification

### 2. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates Kubernetes Cluster receiver functionality
- **Verification**: Checks for cluster-level metrics in collector output

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-k8sclusterreceiver.yaml`](./otel-k8sclusterreceiver.yaml)
2. **Wait for Metrics Collection** - Allow time for cluster metrics to be collected
3. **Verify Metrics Collection** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which validates cluster metrics collection by checking for:
- `k8s.node.allocatable_cpu` - Node allocatable CPU resources
- `k8s.node.allocatable_memory` - Node allocatable memory resources
- `k8s.node.condition_memory_pressure` - Node memory pressure condition
- `k8s.node.condition_ready` - Node ready condition

## 🧹 Cleanup

The test runs in the `chainsaw-k8sclusterreceiver` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses a single collector instance (not DaemonSet) to gather cluster-wide metrics
- Requires RBAC permissions to access various Kubernetes API resources
- Collects metrics every 10 seconds from the Kubernetes API
- Monitors nodes, pods, deployments, services, and other cluster resources
- Runs in the `default` namespace with cluster-wide access 