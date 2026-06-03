# OpenTelemetry OTLP JSON File Receiver Test

This test demonstrates the OpenTelemetry OTLP JSON File receiver configuration for reading telemetry data from JSON files.

## 🎯 What This Test Does

The test validates a complete file-based telemetry pipeline:
- One collector exports traces to a JSON file using the file exporter
- Another collector reads that JSON file using the OTLP JSON file receiver
- Traces are then forwarded to Tempo for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Tempo Instance
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Contains**: TempoMonolithic deployment with Jaeger UI
- **Key Features**:
  - Jaeger UI enabled for trace visualization and querying
  - Route enabled for external access
  - Receives traces from OTLP JSON file receiver

### 2. Persistent Volume Claim
- **File**: [`create-pvc.yaml`](./create-pvc.yaml)
- **Contains**: PersistentVolumeClaim for shared storage
- **Key Features**:
  - 2Gi storage for file exchange between collectors
  - ReadWriteOnce access mode for file-based communication
  - Shared volume for JSON file storage

### 3. File Exporter Collector
- **File**: [`fileexporter-otel-collector.yaml`](./fileexporter-otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with file exporter
- **Key Features**:
  - OTLP receiver for trace ingestion
  - File exporter writing to shared persistent volume
  - Debug exporter for trace verification
  - Volume mount for file output

### 4. OTLP JSON File Receiver Collector
- **File**: [`otlpjsonfilereceiver-otel-collector.yaml`](./otlpjsonfilereceiver-otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with OTLP JSON file receiver
- **Key Features**:
  - OTLP JSON file receiver monitoring JSON files
  - OTLP exporter forwarding traces to Tempo
  - Read-only volume mount for file input
  - Pod affinity for shared volume access

### 5. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating test traces
- **Key Features**:
  - Generates 5 test traces using telemetrygen
  - Targets file exporter collector endpoint
  - Service name "from-otlp-jsonfile" for identification

### 6. Trace Verification
- **File**: [`verify-traces.yaml`](./verify-traces.yaml)
- **Contains**: Job for verifying traces in Tempo
- **Key Features**:
  - Queries Jaeger API for trace verification
  - Validates all 5 traces are present
  - Confirms service name filtering works correctly

### 7. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Tempo Instance** - Deploy from [`install-tempo.yaml`](./install-tempo.yaml)
2. **Create PVC** - Deploy from [`create-pvc.yaml`](./create-pvc.yaml)
3. **Create File Exporter Collector** - Deploy from [`fileexporter-otel-collector.yaml`](./fileexporter-otel-collector.yaml)
4. **Create OTLP JSON File Receiver Collector** - Deploy from [`otlpjsonfilereceiver-otel-collector.yaml`](./otlpjsonfilereceiver-otel-collector.yaml)
5. **Generate Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
6. **Verify Traces** - Execute from [`verify-traces.yaml`](./verify-traces.yaml)

## 🔍 File-Based Pipeline Configuration

### File Exporter Configuration:
- **Output Path**: `/telemetry-data/telemetrygen-traces.json`
- **Format**: JSON format compatible with OTLP JSON file receiver
- **Storage**: Shared persistent volume for cross-collector access

### OTLP JSON File Receiver Configuration:
- **Include Pattern**: `/telemetry-data/*.json`
- **File Monitoring**: Watches for JSON files matching the pattern
- **Processing**: Reads JSON files and converts to OTLP format

### Pod Affinity:
- Ensures both collectors run on the same node for shared volume access
- Required for persistent volume sharing between pods
- Uses label matching for proper scheduling

## 🔍 Verification

The verification is handled by [`verify-traces.yaml`](./verify-traces.yaml), which:
- Queries Tempo via Jaeger API for traces with service name "from-otlp-jsonfile"
- Validates exactly 5 traces are received and stored
- Confirms the complete file-based pipeline works end-to-end
- Ensures JSON file format is properly parsed by the receiver

## 🧹 Cleanup

The test runs in the default namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses shared PVC for file-based communication between collectors
- File exporter writes to `/telemetry-data/telemetrygen-traces.json`
- OTLP JSON file receiver monitors `/telemetry-data/*.json` pattern
- Pod affinity ensures both collectors run on the same node for shared volume access
- Read-only mount for OTLP JSON file receiver to prevent accidental file modification
- Demonstrates end-to-end file-based telemetry pipeline with Tempo integration
- Validates JSON file format compatibility between file exporter and OTLP JSON file receiver