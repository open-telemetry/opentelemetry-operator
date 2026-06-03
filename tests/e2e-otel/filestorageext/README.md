# OpenTelemetry File Storage Extension Test

This test demonstrates the OpenTelemetry File Storage Extension configuration for persistent state management in telemetry pipelines.

## 🎯 What This Test Does

The test validates that the File Storage Extension can:
- Provide persistent storage for the filelog receiver to track file read positions
- Store state information across collector restarts
- Use compaction for storage optimization
- Handle file synchronization (fsync) for data durability
- Process application logs from files using a sidecar pattern

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Main OpenTelemetry Collector Configuration
- **File**: [`otel-filestorageext.yaml`](./otel-filestorageext.yaml)
- **Contains**: Primary OpenTelemetry Collector with debug exporter
- **Key Features**:
  - Deployment mode for receiving forwarded logs
  - OTLP receiver for log ingestion
  - Debug exporter with detailed verbosity for verification

### 2. Sidecar Collector with File Storage Extension
- **File**: [`app-plaintest-logs.yaml`](./app-plaintest-logs.yaml)
- **Contains**: Application deployment with sidecar collector configuration
- **Key Features**:
  - File storage extension for persistent state management
  - Filelog receiver with regex parsing for structured logs
  - Compaction configuration for storage optimization
  - Volume mounts for log data and storage directories
  - OTLP exporter for forwarding logs to main collector

### 3. Verification Scripts
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that logs are properly collected and processed through the file storage extension
- **File**: [`check_filestorageext.sh`](./check_filestorageext.sh)
- **Purpose**: Verifies file storage extension creates required state files in storage and compaction directories

### 4. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OTEL Collector** - Deploy from [`otel-filestorageext.yaml`](./otel-filestorageext.yaml)
2. **Create Log Generator App** - Deploy from [`app-plaintest-logs.yaml`](./app-plaintest-logs.yaml)
3. **Wait for Log Collection** - Allow processing time for logs to flow through the pipeline
4. **Check Collected Logs** - Execute [`check_logs.sh`](./check_logs.sh) validation script
5. **Confirm File Storage Extension** - Execute [`check_filestorageext.sh`](./check_filestorageext.sh) validation script

## 🔍 File Storage Extension Configuration

### Storage Settings:
- **Storage Directory**: `/filestorageext/data`
- **Timeout**: 1 second for storage operations
- **Fsync**: Enabled for data durability and crash consistency

### Compaction Configuration:
- **On Start**: Enabled to optimize storage on collector startup
- **Compaction Directory**: `/filestorageext/compaction`
- **Max Transaction Size**: 65,536 bytes per transaction

### Integration with Filelog Receiver:
- **Storage Reference**: `storage: file_storage`
- **File Pattern**: `/log-data/*.log`
- **State Persistence**: Tracks file read positions across restarts

### Log Processing:
- **Parser**: Regex parser for structured log extraction
- **Timestamp Parsing**: Extracts timestamps from log entries
- **Severity Parsing**: Maps log levels to OpenTelemetry severity

## 🔍 Verification

The test verification includes two checks handled by separate scripts:

### 1. Log Processing Verification:
[`check_logs.sh`](./check_logs.sh) confirms that:
- Logs are successfully collected from files by the filelog receiver
- Logs are properly parsed using regex operators
- Logs are forwarded to the main collector via OTLP

### 2. File Storage Extension Verification:
[`check_filestorageext.sh`](./check_filestorageext.sh) verifies that:
- File storage extension creates `receiver_filelog_` state files in `/filestorageext/data`
- Compaction directory `/filestorageext/compaction` contains the expected state files
- Both storage and compaction directories function correctly for persistent state management

## 🧹 Cleanup

The test runs in a dynamically created namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses sidecar deployment pattern for log collection with file storage
- Demonstrates persistent state management for filelog receiver
- Configures both storage and compaction directories for optimal performance
- Enables fsync for data durability in production-like scenarios
- Shows integration between file storage extension and log processing pipeline
- Validates state file creation in both main storage and compaction directories 