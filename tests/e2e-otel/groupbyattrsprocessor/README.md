# OpenTelemetry Group By Attributes Processor Test

This test demonstrates the OpenTelemetry Group By Attributes processor configuration for grouping metrics by specific attributes.

## 🎯 What This Test Does

The test validates a metric processing pipeline that:
- Collects Kubelet stats metrics using DaemonSet collectors
- Forwards metrics to a main collector with Group By Attributes processor
- Groups metrics to reduce cardinality and improve metric organization
- Exports processed metrics to Prometheus for monitoring

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. User Workload Monitoring Configuration
- **File**: [`workload-monitoring.yaml`](./workload-monitoring.yaml)
- **Contains**: ConfigMap to enable OpenShift user workload monitoring
- **Purpose**: Enables Prometheus monitoring for user workloads

### 2. Kubelet Stats Collector (DaemonSet)
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: DaemonSet OpenTelemetryCollector for kubelet stats collection
- **Key Features**:
  - Service account authentication for kubelet access
  - Node-based metric collection with 20-second intervals
  - OTLP forwarding to main collector
  - Master node tolerations for complete coverage

### 3. Main Collector with Group By Attributes Processor
- **File**: [`otel-groupbyattributes.yaml`](./otel-groupbyattributes.yaml)
- **Contains**: Primary OpenTelemetryCollector with Group By Attributes processor
- **Key Features**:
  - Receives metrics from DaemonSet collectors
  - Applies Group By Attributes processor for cardinality reduction
  - Prometheus exporter for metrics exposure
  - Debug exporter for verification

### 4. Monitoring View Role
- **File**: [`monitoring-view-role.yaml`](./monitoring-view-role.yaml)
- **Contains**: RBAC permissions for accessing metrics endpoints
- **Purpose**: Enables verification script to query metrics

### 5. Verification Script
- **File**: [`check_metrics.sh`](./check_metrics.sh)
- **Purpose**: Validates Group By Attributes processor functionality
- **Verification**: Checks grouped metrics via Prometheus endpoint

### 6. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Enable User Workload Monitoring** - Deploy from [`workload-monitoring.yaml`](./workload-monitoring.yaml)
2. **Create Kubelet Stats Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Create Main Collector with Group By Attributes** - Deploy from [`otel-groupbyattributes.yaml`](./otel-groupbyattributes.yaml)
4. **Create Monitoring Role** - Deploy from [`monitoring-view-role.yaml`](./monitoring-view-role.yaml)
5. **Check Metrics** - Execute [`check_metrics.sh`](./check_metrics.sh) validation script

## 🔍 Verification

The verification is handled by [`check_metrics.sh`](./check_metrics.sh), which:
- Queries the main collector's Prometheus endpoint
- Validates that kubelet stats metrics are properly collected
- Confirms Group By Attributes processor is functioning
- Checks that metrics are forwarded from DaemonSet collectors
- Verifies metric cardinality reduction through attribute grouping
- Ensures processed metrics are available for monitoring

## 🧹 Cleanup

The test runs in the `chainsaw-gba` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses DaemonSet mode for comprehensive node metric collection
- Integrates with OpenShift user workload monitoring
- Demonstrates metric forwarding between collectors
- Group By Attributes processor reduces metric cardinality
- Requires service account authentication for kubelet access
- Tolerates master node taints for complete coverage
- Enables Prometheus integration for metric monitoring 