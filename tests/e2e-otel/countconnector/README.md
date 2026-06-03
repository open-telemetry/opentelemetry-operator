# OpenTelemetry Count Connector Test

This test demonstrates the OpenTelemetry Count connector configuration for counting telemetry data and exposing the counts as metrics.

## 🎯 What This Test Does

The test validates that the Count connector can:
- Count logs, metrics, and traces by specified attributes
- Generate count metrics for each telemetry type (logs, datapoints, spans)
- Export count metrics to Prometheus for monitoring
- Use OpenShift user workload monitoring for metric verification

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with Count connector configuration
- **Key Features**:
  - Deployment mode with metrics observability enabled
  - Count connector for logs, datapoints, and spans
  - Prometheus exporter for metrics exposure
  - Multi-pipeline architecture with count connector routing

### 2. User Workload Monitoring Setup
- **File**: [`workload-monitoring.yaml`](./workload-monitoring.yaml)
- **Contains**: ConfigMap for enabling OpenShift user workload monitoring
- **Purpose**: Enables Prometheus monitoring for user workloads in OpenShift

### 3. Telemetry Data Generator
- **File**: [`generate-telemetry-data.yaml`](./generate-telemetry-data.yaml)
- **Contains**: Job that generates test logs, metrics, and traces
- **Key Features**:
  - Generates telemetry data with `telemetrygentype` attributes
  - Creates data for count connector testing
  - Includes proper labeling for verification

### 4. Verification Script
- **File**: [`check_metrics.sh`](./check_metrics.sh)
- **Purpose**: Validates count metrics are properly generated and exposed
- **Verification Criteria**:
  - `dev_log_count_total{telemetrygentype="logs"}` = 1
  - `dev_metrics_datapoint_total{telemetrygentype="metrics"}` = 1
  - `dev_span_count_total{telemetrygentype="traces"}` = 10
  - Uses OpenShift Thanos querier for metric verification

### 5. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Enable User Workload Monitoring** - Deploy from [`workload-monitoring.yaml`](./workload-monitoring.yaml)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)  
3. **Generate Telemetry Data** - Run job from [`generate-telemetry-data.yaml`](./generate-telemetry-data.yaml)
4. **Verify Metrics** - Execute [`check_metrics.sh`](./check_metrics.sh) validation script

## 🔍 Count Connector Configuration

### Counting Rules:
1. **Log Count**: `dev.log.count`
   - Counts log records by `telemetrygentype` attribute
   - Default value: `unspecified_environment`
   - Expected result: `dev_log_count_total{telemetrygentype="logs"}` = 1

2. **Metrics Datapoint Count**: `dev.metrics.datapoint`  
   - Counts metric datapoints by `telemetrygentype` attribute
   - Default value: `unspecified_environment`
   - Expected result: `dev_metrics_datapoint_total{telemetrygentype="metrics"}` = 1

3. **Span Count**: `dev.span.count`
   - Counts trace spans by `telemetrygentype` attribute
   - Default value: `unspecified_environment`
   - Expected result: `dev_span_count_total{telemetrygentype="traces"}` = 10

### Pipeline Architecture:
- **Input Pipelines**: Separate pipelines for traces, metrics, and logs all export to the count connector
- **Count Connector**: Generates count metrics for each telemetry type
- **Output Pipeline**: `metrics/count` receives count metrics and exports to Prometheus

### Export Configuration:
- **Prometheus Endpoint**: `0.0.0.0:8889`
- **Resource to Telemetry Conversion**: Enabled to include resource attributes as labels
- **Debug Exporter**: Provides detailed output for verification

## 🔍 Verification

The verification is handled by [`check_metrics.sh`](./check_metrics.sh), which:
- Queries OpenShift's Thanos API for count metrics
- Uses OpenShift user workload monitoring token for authentication
- Polls until expected metric values are found
- Validates specific count metrics with proper attribute grouping

**Expected Metrics:**
- `dev_log_count_total{telemetrygentype="logs"}` = 1
- `dev_metrics_datapoint_total{telemetrygentype="metrics"}` = 1  
- `dev_span_count_total{telemetrygentype="traces"}` = 10
- `metric_count_total{telemetrygentype="metrics"}` = 1

## 🧹 Cleanup

The test runs in the default namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses count connector to transform telemetry data into count metrics
- Demonstrates attribute-based counting with configurable grouping keys
- Integrates with OpenShift user workload monitoring for verification
- Supports counting across all three telemetry data types simultaneously
- Provides default values for missing attributes in counting logic
- Exports count metrics in Prometheus format for external monitoring systems 