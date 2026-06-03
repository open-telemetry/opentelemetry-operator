# OpenTelemetry Transform Processor Test

This test demonstrates the OpenTelemetry Transform processor configuration for modifying telemetry data attributes and content.

## 🎯 What This Test Does

The test validates that the Transform processor can:
- Modify resource attributes using keep_keys, set, limit, and truncate_all operations
- Transform span attributes and names based on conditions
- Forward transformed traces to Tempo for verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Tempo Instance
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Contains**: TempoMonolithic deployment for trace storage
- **Key Features**:
  - Jaeger UI enabled for trace visualization and querying
  - Multitenancy disabled for simplified configuration
  - Receives transformed traces from OpenTelemetry collector

### 2. OpenTelemetry Collector with Transform Processor
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with transform processor configuration
- **Key Features**:
  - OTLP receiver for trace ingestion
  - Transform processor with resource and span transformations
  - OTLP exporter forwarding transformed traces to Tempo
  - Error mode set to ignore for fault tolerance

### 3. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating test traces with specific attributes
- **Key Features**:
  - Generates traces using telemetrygen with target attributes
  - Creates traces with attributes that will be transformed
  - Provides input data for transformation testing

### 4. Trace Verification
- **File**: [`verify-traces.yaml`](./verify-traces.yaml)
- **Contains**: Job for verifying transformed traces in Tempo
- **Key Features**:
  - Queries Tempo via Jaeger API for trace verification
  - Validates that transformations are applied correctly
  - Confirms transformed traces are stored successfully

### 5. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Tempo Instance** - Deploy from [`install-tempo.yaml`](./install-tempo.yaml)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Generate Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
4. **Verify Traces** - Execute from [`verify-traces.yaml`](./verify-traces.yaml)

## 🔍 Transform Rules Applied

### Resource Context Transformations:
- **keep_keys**: Retains only `service.name`, `X-Tenant`, and `otel.library.name` attributes
- **set**: Changes `X-Tenant` from "green" to "blue"
- **limit**: Limits attributes to 100 maximum
- **truncate_all**: Truncates all attribute values to 4096 characters

### Span Context Transformations:
- **IP Address**: Changes `net.sock.peer.addr` "1.2.3.4" to `net.peer.ip` "5.6.7.8"
- **Service Names**: 
  - "telemetrygen-server" → "modified-server"
  - "telemetrygen-client" → "modified-client"
- **Operation Name**: "okey-dokey-0" → "modified-operation"
- **limit**: Limits span attributes to 100 maximum
- **truncate_all**: Truncates all span attribute values to 4096 characters

## 🔍 Transform Processor Configuration

### Error Handling:
- **Error Mode**: `ignore` - Continue processing even if transformation errors occur
- **Fault Tolerance**: Prevents pipeline failures due to transformation issues

### Conditional Transformations:
- Uses `where` clauses for targeted attribute modifications
- Applies transformations only when specific conditions are met
- Supports complex conditional logic for attribute manipulation

### Context Levels:
- **Resource Context**: Transformations applied to resource-level attributes
- **Span Context**: Transformations applied to individual span attributes and names

## 🔍 Verification

The verification is handled by [`verify-traces.yaml`](./verify-traces.yaml), which:
- Queries Tempo for transformed traces via Jaeger API
- Validates that attribute transformations are applied correctly
- Confirms conditional transformations work as expected
- Ensures transformed traces are properly stored and accessible

## 🧹 Cleanup

The test runs in the `chainsaw-tprocssr` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses `error_mode: ignore` to continue processing even if transformation errors occur
- Applies transformations at both resource and span contexts
- Uses conditional statements with `where` clauses for targeted transformations
- Demonstrates attribute filtering, modification, limiting, and truncation
- Integrates with Tempo for end-to-end trace verification
- Supports complex transformation logic with conditional execution