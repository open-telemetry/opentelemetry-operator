# OpenTelemetry HostMetrics Receiver Test

This test demonstrates the OpenTelemetry HostMetrics receiver configuration for collecting system metrics from Kubernetes nodes.

## 🎯 What This Test Does

The test validates that the HostMetrics receiver can:
- Collect comprehensive system metrics from each node using a DaemonSet deployment
- Access host filesystem for accurate metric collection
- Export collected metrics to a debug exporter for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-hostmetricsreceiver.yaml`](./otel-hostmetricsreceiver.yaml)
- **Contains**: DaemonSet OpenTelemetryCollector with HostMetrics receiver
- **Key Features**:
  - DaemonSet deployment for node-level metric collection
  - Privileged security context for host filesystem access
  - Host filesystem mount at `/hostfs` for accurate metrics
  - Comprehensive metric scrapers (CPU, memory, disk, network, etc.)
  - Debug exporter for verification
  - Master node tolerations for complete coverage

### 2. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates HostMetrics receiver functionality
- **Verification**: Checks for comprehensive host system metrics in collector output

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OpenTelemetry Collector** - Deploy from [`otel-hostmetricsreceiver.yaml`](./otel-hostmetricsreceiver.yaml)
2. **Wait for Metrics Collection** - Allow time for host metrics to be collected
3. **Verify Metrics Collection** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which validates the collection of comprehensive host metrics including:

**Process Metrics:**
- `process.pid` - Process ID
- `process.parent_pid` - Parent process ID
- `process.executable.name` - Executable name
- `process.executable.path` - Executable path
- `process.command` - Command line
- `process.cpu.time` - Process CPU time
- `process.disk.io` - Process disk I/O
- `process.memory.usage` - Process memory usage
- `process.memory.virtual` - Process virtual memory

**System CPU Metrics:**
- `system.cpu.load_average.1m` - 1-minute load average
- `system.cpu.load_average.5m` - 5-minute load average
- `system.cpu.load_average.15m` - 15-minute load average
- `system.cpu.time` - CPU time

**System Disk Metrics:**
- `system.disk.io` - Disk I/O bytes
- `system.disk.io_time` - Disk I/O time
- `system.disk.merged` - Merged disk operations
- `system.disk.operation_time` - Disk operation time
- `system.disk.operations` - Disk operations count
- `system.disk.pending_operations` - Pending disk operations
- `system.disk.weighted_io_time` - Weighted I/O time

**System Filesystem Metrics:**
- `system.filesystem.inodes.usage` - Inode usage
- `system.filesystem.usage` - Filesystem usage

**System Memory Metrics:**
- `system.memory.usage` - Memory usage

**System Network Metrics:**
- `system.network.connections` - Network connections
- `system.network.dropped` - Dropped packets
- `system.network.errors` - Network errors
- `system.network.io` - Network I/O bytes
- `system.network.packets` - Network packets

**System Paging Metrics:**
- `system.paging.faults` - Page faults
- `system.paging.operations` - Paging operations

**System Process Metrics:**
- `system.processes.count` - Process count
- `system.processes.created` - Processes created

## 🧹 Cleanup

The test runs in the `chainsaw-hostmetrics` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses DaemonSet mode to collect metrics from all nodes
- Mounts entire host filesystem at `/hostfs` for accurate metric collection
- Requires privileged access and SYS_ADMIN capability
- Collects metrics every 10 seconds from all available scrapers
- Tolerates master node taints for comprehensive coverage
- Uses read-only host filesystem mount with HostToContainer propagation 