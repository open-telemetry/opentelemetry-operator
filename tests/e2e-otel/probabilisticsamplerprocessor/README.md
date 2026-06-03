# Probabilistic Sampler Processor Test Suite

This directory contains comprehensive end-to-end tests for the OpenTelemetry Probabilistic Sampler Processor, focusing on trace telemetry sampling capabilities.

## 🎯 Test Coverage

### Sampling Modes
- **Hash Seed Mode**: Deterministic sampling using FNV hash function with configurable seed
- **Proportional Mode**: W3C TraceContext compliant proportional sampling

### Key Scenarios Tested
1. **Hash Seed Sampling**: 25% sampling with deterministic seed-based decisions
2. **Proportional Sampling**: 50% sampling with W3C-compliant threshold encoding
3. **TraceState Validation**: Proper encoding of sampling decisions in trace headers
4. **Debug Export**: Detailed trace output for validation and testing
5. **Service Integration**: OpenTelemetry Operator service discovery
6. **Security Compliance**: OpenShift-compatible security contexts

## 🚀 Running Tests

### Prerequisites
- Kubernetes cluster with OpenTelemetry Operator
- Chainsaw testing framework
- Export KUBECONFIG: `export KUBECONFIG=~/Downloads/kubeconfig`

## 📋 Test Files Overview

### Core Test Files
- `simple-test.yaml` - Main Chainsaw test with both sampling modes
- `otel-collector-hash-seed.yaml` - Hash seed mode collector configuration
- `otel-collector-proportional.yaml` - Proportional mode collector configuration
- `generate-traces-hash-seed.yaml` - Trace generation job for hash seed testing
- `generate-traces-proportional.yaml` - Trace generation job for proportional testing

### Assertion Files
- `assert-otel-collector.yaml` - Collector deployment validation
- `generate-traces-assert.yaml` - Job completion validation

## 🔧 Configuration Examples

### Hash Seed Mode
```yaml
processors:
  probabilistic_sampler:
    mode: hash_seed
    sampling_percentage: 25
    hash_seed: 12345
    sampling_precision: 4
    fail_closed: true
```

### Proportional Mode
```yaml
processors:
  probabilistic_sampler:
    mode: proportional
    sampling_percentage: 50
    sampling_precision: 4
    fail_closed: true
```

### Trace Pipeline
```yaml
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [probabilistic_sampler]
      exporters: [debug]
```

## 📊 Validation Methods

1. **Trace Processing**: Confirms traces are being received and processed
2. **Sampling Decisions**: Verifies sampling is applied based on configuration  
3. **TraceState Encoding**: Validates W3C-compliant tracestate headers
4. **Export Pipeline**: Ensures traces reach configured exporters
5. **Service Connectivity**: Confirms telemetrygen can connect to collector

## 🔍 Expected Results

```
=== Hash Seed Mode Results ===
Traces found in debug output: 6
Tracestate entries found: 6
✓ SUCCESS: Probabilistic sampler is working correctly!

=== Proportional Mode Results ===
Traces found in debug output: 7  
Tracestate entries found: 7
✓ SUCCESS: Proportional mode is working correctly!
```

## ✅ Success Criteria

Tests pass when:
- Collector pods are running and ready
- Trace generation jobs complete successfully
- Debug output shows processed traces
- TraceState headers contain sampling information
- Both hash_seed and proportional modes function correctly