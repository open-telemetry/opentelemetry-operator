# OpenTelemetry Forward Connector Test

This test demonstrates the OpenTelemetry Forward connector configuration for aggregating telemetry data from multiple input pipelines into a single output pipeline.

## 🎯 What This Test Does

The test validates that the Forward connector can:
- Collect traces from multiple input pipelines (blue and green)
- Add different attributes to traces in each pipeline
- Forward all traces to a single output pipeline for unified processing
- Maintain trace data integrity through the forwarding process

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-forward-connector.yaml`](./otel-forward-connector.yaml)
- **Contains**: OpenTelemetryCollector with Forward connector configuration
- **Key Features**:
  - Multiple OTLP receivers for blue and green pipelines
  - Attribute processors for pipeline-specific tagging
  - Forward connector for trace aggregation
  - Unified output pipeline with batch processing

### 2. Trace Generators
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job that generates test traces for both pipelines
- **Key Features**:
  - Creates traces with operation name `lets-go` for blue pipeline
  - Creates traces with operation name `okey-dokey` for green pipeline
  - Tests forward connector aggregation capabilities

### 3. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates forward connector behavior and trace aggregation
- **Verification Criteria**:
  - Confirms pipeline-specific attributes are added correctly
  - Validates traces from both pipelines appear in unified output
  - Checks trace operation names are preserved

### 4. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OTEL Collector** - Deploy from [`otel-forward-connector.yaml`](./otel-forward-connector.yaml)
2. **Generate Traces** - Run job from [`generate-traces.yaml`](./generate-traces.yaml)
3. **Wait for Processing** - Allow time for traces to be processed and forwarded
4. **Check Traces** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Forward Connector Architecture

### Input Pipelines:
1. **Blue Pipeline**: `traces/blue`
   - **Receiver**: `otlp/blue` (default HTTP endpoint)
   - **Processor**: `attributes/blue` (adds `otel_pipeline_tag: "blue"`)
   - **Exporter**: `forward` (sends to forward connector)

2. **Green Pipeline**: `traces/green`
   - **Receiver**: `otlp/green` (HTTP endpoint on port 4319)
   - **Processor**: `attributes/green` (adds `otel_pipeline_tag: "green"`)
   - **Exporter**: `forward` (sends to forward connector)

### Output Pipeline:
- **Pipeline**: `traces` (unified output)
- **Receiver**: `forward` (receives from both input pipelines)
- **Processor**: `batch` (batches traces for efficiency)
- **Exporter**: `debug` (outputs traces for verification)

### Data Flow:
```
Blue OTLP → attributes/blue → forward ↘
                                      → unified traces → batch → debug
Green OTLP → attributes/green → forward ↗
```

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Monitors collector output logs for evidence of forward connector operation
- Validates that traces from both pipelines are processed correctly
- Confirms attribute processors add pipeline-specific tags

**Expected Verification Points:**
- `otel_pipeline_tag: Str(blue)` - Confirms blue pipeline processing
- `otel_pipeline_tag: Str(green)` - Confirms green pipeline processing  
- `Name           : lets-go` - Confirms blue trace operation name
- `Name           : okey-dokey` - Confirms green trace operation name

This validates that:
- Both input pipelines are receiving and processing traces
- Attribute processors are adding pipeline-specific tags
- Forward connector is successfully aggregating traces from both pipelines
- All trace data is preserved through the forwarding process

## 🧹 Cleanup

The test runs in the `chainsaw-forwardconnector` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Demonstrates aggregation pattern where multiple input sources feed into a single output
- Each input pipeline can apply different processing before forwarding
- Forward connector acts as a bridge between input and output pipelines
- Enables unified processing and export of traces from multiple sources
- Useful for scenarios requiring trace aggregation from different ingestion points 