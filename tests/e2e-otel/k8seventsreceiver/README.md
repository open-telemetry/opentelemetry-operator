# OpenTelemetry Kubernetes Events Receiver Test

This test demonstrates the OpenTelemetry Kubernetes Events receiver configuration for collecting Kubernetes events.

## 🎯 What This Test Does

The test validates that the Kubernetes Events receiver can:
- Collect Kubernetes events from a specific namespace
- Access Kubernetes API to gather event information
- Export collected events to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-k8seventsreceiver.yaml`](./otel-k8seventsreceiver.yaml)
- **Contains**: ServiceAccount, ClusterRole, ClusterRoleBinding, and OpenTelemetryCollector configuration
- **Key Features**:
  - Deployment mode for collecting Kubernetes events
  - k8s_events receiver configured for specific namespace monitoring
  - RBAC permissions for accessing events, pods, nodes, and other Kubernetes resources
  - ServiceAccount authentication for secure API access
  - Debug exporter for event verification

### 2. Test Application (Event Generator)
- **File**: [`install-app.yaml`](./install-app.yaml)
- **Contains**: Deployment for generating Kubernetes events
- **Key Features**:
  - Creates pod deployment to trigger Kubernetes events
  - Provides events for the receiver to collect and process

### 3. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that Kubernetes events are properly collected and processed
- **Verification Criteria**:
  - `k8s.event.reason` - Event reason attribute
  - `k8s.event.action` - Event action attribute
  - `k8s.event.start_time` - Event start time attribute
  - `k8s.event.name` - Event name attribute
  - `k8s.event.uid` - Event UID attribute
  - `k8s.namespace.name: Str(chainsaw-k8seventsreceiver)` - Namespace attribute
  - `k8s.event.count` - Event count attribute

### 4. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-k8seventsreceiver.yaml`](./otel-k8seventsreceiver.yaml)
2. **Deploy Sample Application** - Deploy from [`install-app.yaml`](./install-app.yaml)
3. **Verify Event Collection** - Run [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Monitors OpenTelemetry collector pod logs for Kubernetes events
- Searches for specific event attributes to confirm proper collection
- Validates that k8s_events receiver captures events with correct metadata
- Ensures events from the target namespace are properly processed

## 🧹 Cleanup

The test runs in the `chainsaw-k8seventsreceiver` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses deployment mode (single instance) to collect events
- Filters events to only collect from the `chainsaw-k8seventsreceiver` namespace
- Includes both OTLP receiver for traces and k8s_events receiver for logs
- Requires comprehensive RBAC permissions to access Kubernetes events and other resources
- Supports OpenShift quota resources in addition to standard Kubernetes resources
- Uses ServiceAccount authentication for secure API access