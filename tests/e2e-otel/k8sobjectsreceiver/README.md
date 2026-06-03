# OpenTelemetry Kubernetes Objects Receiver Test

This test demonstrates the OpenTelemetry Kubernetes Objects receiver configuration for collecting Kubernetes object data.

## 🎯 What This Test Does

The test validates that the Kubernetes Objects receiver can:
- Collect Kubernetes object data using both pull and watch modes
- Access Kubernetes API to gather pod and event information
- Export collected object data to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-k8sobjectsreceiver.yaml`](./otel-k8sobjectsreceiver.yaml)
- **Contains**: ServiceAccount, ClusterRole, ClusterRoleBinding, and OpenTelemetryCollector configuration
- **Key Features**:
  - Deployment mode for collecting Kubernetes object data
  - k8sobjects receiver with pull and watch modes for different object types
  - RBAC permissions for accessing pods and events resources
  - ServiceAccount authentication for secure API access
  - Debug exporter for object data verification

### 2. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that Kubernetes object data is properly collected and processed
- **Verification Criteria**:
  - `Body: Map({"object":` - Object data structure in logs
  - `k8s.resource.name` - Kubernetes resource name attribute
  - `event.domain` - Event domain information
  - `event.name` - Event name information

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-k8sobjectsreceiver.yaml`](./otel-k8sobjectsreceiver.yaml)
2. **Verify Object Collection** - Run [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 K8s Objects Receiver Configuration

### Object Collection Modes:
1. **Pods Collection**: `pull` mode
   - Periodically collects pod information
   - Captures current state of pods in the cluster

2. **Events Collection**: `watch` mode  
   - Real-time streaming of Kubernetes events
   - Monitors event changes as they occur

### Supported Object Types:
- **Pods**: Container and workload information
- **Events**: Kubernetes cluster events and notifications

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Monitors OpenTelemetry collector pod logs for Kubernetes object data
- Searches for specific object data structures to confirm proper collection
- Validates that k8sobjects receiver captures object metadata correctly
- Ensures both pull and watch modes are functioning properly

## 🧹 Cleanup

The test runs in the `chainsaw-k8sobjectsreceiver` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses deployment mode (single instance) to collect object data
- Configures two object types with different collection modes:
  - `pods` using pull mode (periodic collection)
  - `events` using watch mode (real-time streaming)
- Requires RBAC permissions to access pods and events resources
- Supports both core API events and events.k8s.io API group events
- Outputs collected data as logs for analysis and verification
- Uses ServiceAccount authentication for secure API access