# OpenTelemetry Tail Sampling Processor Test

This test demonstrates the OpenTelemetry Tail Sampling processor configuration for intelligent trace sampling based on complete trace information.

## 🎯 What This Test Does

The test validates that the Tail Sampling processor can:
- Wait for complete trace spans before making sampling decisions
- Apply multiple sampling policies (status code, latency, span count, service name)
- Forward sampled traces to Tempo for storage and verification
- Process traces from both generated telemetry and HotROD application

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Tempo Instance
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Contains**: TempoMonolithic deployment for trace storage
- **Key Features**:
  - Jaeger UI enabled for trace visualization and querying
  - Receives sampled traces from OpenTelemetry collector
  - Provides trace storage backend for verification

### 2. OpenTelemetry Collector with Tail Sampling
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with tail sampling processor
- **Key Features**:
  - OTLP receiver for trace ingestion (gRPC and HTTP)
  - Tail sampling processor with multiple policies
  - OTLP exporter forwarding sampled traces to Tempo
  - Debug exporter for trace verification

### 3. HotROD Application
- **File**: [`install-hotrod.yaml`](./install-hotrod.yaml)
- **Contains**: Deployment for HotROD (Rides on Demand) demo application
- **Key Features**:
  - Generates realistic traces with various characteristics
  - Provides complex trace patterns for testing sampling policies
  - Creates traces with different latencies, span counts, and services

### 4. HotROD Traffic Generation
- **File**: [`hotrod-traces.yaml`](./hotrod-traces.yaml)
- **Contains**: Job for generating HotROD application traffic
- **Key Features**:
  - Creates realistic user interactions with HotROD app
  - Generates diverse trace patterns for sampling validation
  - Triggers various service calls and operations

### 5. Additional Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating additional test traces
- **Key Features**:
  - Generates supplementary traces using telemetrygen
  - Creates traces with specific attributes for policy testing
  - Adds more trace volume for comprehensive testing

### 6. Trace Verification
- **File**: [`verify-traces.yaml`](./verify-traces.yaml)
- **Contains**: Job for verifying sampled traces in Tempo
- **Key Features**:
  - Queries Tempo via Jaeger API for trace verification
  - Validates that sampling policies work correctly
  - Confirms expected traces are sampled and stored

### 7. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Tempo Instance** - Deploy from [`install-tempo.yaml`](./install-tempo.yaml)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Install HotROD App** - Deploy from [`install-hotrod.yaml`](./install-hotrod.yaml)
4. **Generate HotROD Traces** - Run from [`hotrod-traces.yaml`](./hotrod-traces.yaml)
5. **Generate Additional Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
6. **Verify Traces** - Execute from [`verify-traces.yaml`](./verify-traces.yaml)

## 🔍 Sampling Policies Applied

### 1. Status Code Policy
- **Name**: `status-code-policy`
- **Type**: `status_code`
- **Rule**: Sample traces with ERROR status codes
- **Purpose**: Ensure all error traces are captured

### 2. Latency Policy
- **Name**: `latency-policy`
- **Type**: `latency`
- **Rule**: Sample traces with latency between 5000ms and 10000ms
- **Purpose**: Capture slow but not extremely slow requests

### 3. Span Count Policy
- **Name**: `span-count-policy`
- **Type**: `span_count`
- **Rule**: Sample traces with 39-50 spans
- **Purpose**: Focus on complex traces with many operations

### 4. Service Name Policy
- **Name**: `service-name-policy`
- **Type**: `string_attribute`
- **Rule**: Sample traces from services named "customer"
- **Purpose**: Ensure traces from critical service are sampled

## 🔧 Configuration Parameters

- **Decision Wait**: 30 seconds - Time to wait for complete trace before sampling decision
- **Num Traces**: 50,000 - Maximum number of traces to keep in memory
- **Expected New Traces Per Sec**: 20 - Expected trace rate for memory management
- **Policy Evaluation**: OR logic - Trace is sampled if ANY policy matches

## 🔍 Verification

The verification is handled by [`verify-traces.yaml`](./verify-traces.yaml), which:
- Queries Tempo for traces that match the sampling policies
- Validates that error traces are consistently sampled
- Confirms latency-based sampling works correctly
- Ensures service-specific sampling is applied
- Verifies span count filtering functions properly

## 🧹 Cleanup

The test runs in the `chainsaw-tailsmp` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses tail-based sampling requiring complete trace information before decisions
- Multiple policies with OR logic - any matching policy triggers sampling
- Integrates with Tempo for trace storage and Jaeger UI for verification
- Handles both generated telemetry and realistic application traces
- Balances comprehensive error capture with selective sampling of normal traces
- Decision wait time ensures complete traces are available for policy evaluation