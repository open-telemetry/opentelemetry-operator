# OpenTelemetry Filter Processor Test

This test demonstrates the OpenTelemetry Filter processor configuration for filtering out unwanted telemetry data.

## 🎯 What This Test Does

The test validates that the Filter processor can:
- Filter traces based on span and resource attributes
- Filter metrics based on metric names and resource attributes  
- Filter logs based on log body content
- Only allow "green" telemetry data while filtering out "red" data

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with Filter processor configuration
- **Key Features**:
  - Filter processor with OTTL expressions for traces, metrics, and logs
  - Error mode set to ignore for continued processing
  - Debug exporter with detailed verbosity for verification
  - Multi-pipeline configuration for all telemetry data types

### 2. Telemetry Data Generator
- **File**: [`generate-telemetry-data.yaml`](./generate-telemetry-data.yaml)
- **Contains**: Job that generates test traces, metrics, and logs
- **Key Features**:
  - Creates telemetry data with both "red" and "green" attributes
  - Tests filter processor capabilities across all data types
  - Includes various attribute combinations for comprehensive testing

### 3. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates filter processor behavior by checking output
- **Verification Criteria**:
  - Confirms "green" data passes through the filter
  - Ensures "red" data is filtered out as expected
  - Validates OTTL expression functionality

### 4. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
2. **Generate Telemetry Data** - Run job from [`generate-telemetry-data.yaml`](./generate-telemetry-data.yaml)
3. **Wait for Processing** - Allow time for telemetry data to be processed through the filter
4. **Check Filtered Data** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Filter Rules Applied

### Trace Filtering:
- **Filter out**: Spans with `traces-env` attribute equal to "red"
- **Filter out**: Spans with resource attribute `traces-colour` equal to "red"
- **Allow through**: Spans with `traces-colour` equal to "green"

### Metric Filtering:
- **Filter out**: Metrics named "gen" with resource attribute `metrics-colour` equal to "red"
- **Allow through**: Metrics with `metrics-colour` equal to "green"

### Log Filtering:
- **Filter out**: Log records containing "drop message" in the body
- **Allow through**: Other log records

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Checks collector output logs for evidence of filter processor behavior
- Validates that expected data ("green" attributes) passes through
- Confirms that unwanted data ("red" attributes) is filtered out

**Expected Data (should be present):**
- `logs-colour: Str(green)` - Green logs passed through
- `metrics-colour: Str(green)` - Green metrics passed through  
- `traces-colour: Str(green)` - Green traces passed through

**Filtered Data (should NOT be present):**
- `logs-colour: Str(red)` - Red logs filtered out
- `metrics-colour: Str(red)` - Red metrics filtered out
- `traces-colour: Str(red)` - Red traces filtered out

## 🧹 Cleanup

The test runs in the default namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses `error_mode: ignore` to continue processing even if filter errors occur
- Filters apply to all three telemetry data types: traces, metrics, and logs
- Filter conditions use OTTL (OpenTelemetry Transformation Language) expressions
- Supports complex conditions with logical operators and functions
- Debug exporter allows verification of filtered data output 