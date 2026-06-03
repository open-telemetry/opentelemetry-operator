# OpenTelemetry Journald Receiver Test

This test demonstrates the OpenTelemetry Journald receiver configuration for collecting systemd journal logs.

## 🎯 What This Test Does

The test validates that the Journald receiver can:
- Collect systemd journal logs from host system
- Access journal files with privileged permissions
- Export collected journal logs to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-journaldreceiver.yaml`](./otel-journaldreceiver.yaml)
- **Contains**: DaemonSet OpenTelemetryCollector with Journald receiver
- **Key Features**:
  - DaemonSet deployment for node-level journal log collection
  - Privileged security context with SELinux type `spc_t`
  - Host journal directory mount for accessing systemd logs
  - Specific systemd unit filtering (kubelet.service, crio.service)
  - Debug exporter for verification
  - Master node tolerations for complete coverage

### 2. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates Journald receiver functionality
- **Verification**: Checks for systemd journal logs and specific fields

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-journaldreceiver.yaml`](./otel-journaldreceiver.yaml)
2. **Wait for Log Collection** - Allow time for journal logs to be collected
3. **Verify Log Collection** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which validates the collection of systemd journal logs by checking for specific fields:
- `_SYSTEMD_UNIT` - Systemd unit information
- `_UID` - User ID
- `_HOSTNAME` - Hostname
- `_SYSTEMD_INVOCATION_ID` - Systemd invocation ID
- `_SELINUX_CONTEXT` - SELinux context

## 🧹 Cleanup

The test runs in the `chainsaw-journald` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses DaemonSet mode to collect journal logs from all nodes
- Requires privileged security context and special SELinux type (spc_t)
- Uses Red Hat OpenShift Distributed Tracing collector image
- Mounts host journal directory (`/var/log/journal`) as read-only
- Filters logs to specific systemd units: kubelet.service and crio.service
- Drops most capabilities for security while maintaining necessary access
- Tolerates master node taints for comprehensive coverage
- Requires privileged SCC (Security Context Constraint) in OpenShift 