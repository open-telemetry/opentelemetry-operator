# OpenTelemetry Kubelet Stats Receiver Test

This test demonstrates the OpenTelemetry Kubelet Stats receiver configuration for collecting container and pod metrics from Kubernetes nodes.

## 🎯 What This Test Does

The test validates that the Kubelet Stats receiver can:
- Collect container, pod, and node metrics from kubelet stats API
- Access kubelet metrics endpoint securely using service account authentication
- Export collected metrics to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-kubeletstatsreceiver.yaml`](./otel-kubeletstatsreceiver.yaml)
- **Contains**: ServiceAccount, ClusterRole, ClusterRoleBinding, and OpenTelemetryCollector configuration
- **Key Features**:
  - DaemonSet mode for collecting metrics from all nodes
  - kubeletstats receiver with service account authentication
  - RBAC permissions for accessing nodes/stats and nodes/proxy
  - Dynamic endpoint configuration using node name environment variable
  - Debug exporter for metrics verification
  - Tolerations for master node scheduling

### 2. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that kubelet metrics are properly collected and processed
- **Verification Criteria**:
  - **Container Metrics**: `container.cpu.time`, `container.cpu.usage`, `container.filesystem.*`, `container.memory.*`
  - **Node Metrics**: `k8s.node.cpu.time`, `k8s.node.cpu.usage`, `k8s.node.filesystem.*`, `k8s.node.memory.*`
  - **Pod Metrics**: `k8s.pod.cpu.time`, `k8s.pod.cpu.usage`, `k8s.pod.filesystem.*`, `k8s.pod.memory.*`, `k8s.pod.network.*`

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-kubeletstatsreceiver.yaml`](./otel-kubeletstatsreceiver.yaml)
2. **Verify Metrics Collection** - Run [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Kubelet Stats Receiver Configuration

### Collection Settings:
- **Collection Interval**: 20 seconds
- **Authentication**: Service Account
- **Endpoint**: `https://${K8S_NODE_NAME}:10250`
- **TLS Verification**: Disabled for testing
- **Extra Metadata**: Container ID labels

### Collected Metrics Categories:

**Container Metrics:**
- CPU: time, usage
- Filesystem: available, capacity, usage
- Memory: major_page_faults, page_faults, rss, usage, working_set

**Node Metrics:**
- CPU: time, usage
- Filesystem: available, capacity, usage  
- Memory: available, major_page_faults, page_faults, rss, usage, working_set

**Pod Metrics:**
- CPU: time, usage
- Filesystem: available, capacity, usage
- Memory: major_page_faults, page_faults, rss, usage, working_set
- Network: errors, I/O

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Monitors OpenTelemetry collector pod logs for kubelet metrics
- Searches for specific metric names to confirm proper collection
- Validates that kubeletstats receiver captures comprehensive node, pod, and container metrics
- Ensures DaemonSet deployment collects from all nodes

## 🧹 Cleanup

The test runs in the `chainsaw-kubeletstatsreceiver` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses DaemonSet mode to collect metrics from all nodes
- Connects to kubelet stats API on port 10250 using HTTPS
- Uses service account authentication for secure access
- Skips TLS verification for testing purposes
- Collects metrics every 20 seconds
- Includes container.id as extra metadata label
- Tolerates master node taints for comprehensive coverage
- Dynamically configures endpoint using K8S_NODE_NAME environment variable