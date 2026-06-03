# OpenTelemetry Profiles Signal Test

This test validates the OpenTelemetry Profiles signal support, which is currently in **Tech Preview**.

## What This Test Does

The test validates that the OpenTelemetry Collector can:
- Enable the profiles signal via the `service.profilesSupport` feature gate
- Configure a profiles pipeline with OTLP receiver and debug exporter
- Start successfully with profiles support enabled
- Run without errors when profiles pipeline is configured

## Test Scope

This is a **configuration test** that verifies:

✅ **What IS tested**:
- Feature gate `service.profilesSupport` is enabled
- Profiles pipeline exists in collector configuration
- OTLP receiver is configured for profiles
- Debug exporter is configured
- Collector deployment is created and ready
- Collector pod starts without errors
- No fatal errors or panics in logs

❌ **What is NOT tested**:
- Actual profile data transmission (no stable OTLP profile exporters exist yet)
- End-to-end profile data flow
- Profile data processing or storage

## Why This Limitation?

As of December 2024, **OpenTelemetry profiles are in Tech Preview** and:
- There is no stable OTLP profile exporter in standard SDKs (Go, Python, Java)
- Profile data generation requires eBPF profilers or Grafana-specific integrations
- The collector can be configured for profiles, but we can't easily send test data

This test focuses on validating that the **infrastructure and configuration work correctly**, which is valuable for ensuring the collector can be deployed with profiles support when the ecosystem matures.

## Test Resources

### 1. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector resource with profiles pipeline
- **Key Features**:
  - **Feature Gate**: `service.profilesSupport` enabled (required for Tech Preview)
  - **OTLP Receiver**: Configured for both gRPC (4317) and HTTP (4318) protocols
  - **Profiles Pipeline**: Dedicated pipeline with OTLP receiver and debug exporter
  - **Debug Exporter**: For logging profile data when received
  - **Deployment Mode**: Standard deployment

### 2. Verification Script
- **File**: [`check_profiles.sh`](./check_profiles.sh)
- **Purpose**: Validates that the profiles signal is properly configured and operational
- **Verification Criteria**:
  - Feature gate `service.profilesSupport` is enabled
  - Profiles pipeline exists in collector configuration
  - OTLP receiver is configured
  - Debug exporter is configured
  - Collector pod is ready and operational
  - No errors, fatals, or panics in collector logs

### 3. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Steps**:
  1. Create OpenTelemetry Collector with profiles support
  2. Verify configuration and check for errors

## Test Steps

The test follows this sequence:

1. **Create OpenTelemetry Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
2. **Verify Configuration** - Run [`check_profiles.sh`](./check_profiles.sh) to validate setup and check logs

## Running the Test

From the repository root:

```bash
export KUBECONFIG=~/Downloads/kubeconfig
chainsaw test --test-dir tests/e2e-otel/profilesignal/
```

## Key Configuration Notes

### Feature Gate (Tech Preview)
The profiles signal is currently in **Tech Preview** and requires the feature gate to be explicitly enabled:

```yaml
spec:
  args:
    feature-gates: service.profilesSupport
```

### Profiles Pipeline Configuration
The profiles pipeline follows the same structure as other signal types:

```yaml
service:
  pipelines:
    profiles:
      receivers: [otlp]
      exporters: [debug]
```

### OTLP Receiver
The OTLP receiver supports profile data on both protocols:
- **gRPC**: Port 4317 (default)
- **HTTP**: Port 4318 (default)

## Expected Test Output

```
=== Attempt 1 ===
Checking profile signal support in collector...
✓ Found collector pods: otel-profiles-collector-collector-xxx
✓ Feature gate 'service.profilesSupport' is enabled
✓ Pod otel-profiles-collector-collector-xxx is ready
✓ No errors found in collector logs
✓ Profiles pipeline configured in OpenTelemetryCollector resource
✓ OTLP receiver configured
✓ Debug exporter configured

SUCCESS: Profile signal support verified!
Summary:
  ✓ Feature gate 'service.profilesSupport' enabled
  ✓ Profiles pipeline configured with OTLP receiver
  ✓ Debug exporter configured
  ✓ Collector pod is ready and running
  ✓ No errors found in collector logs
```

## Future Enhancements

When OpenTelemetry profiles support matures:

1. **Add Profile Data Generator**: Use eBPF profiler or SDK with OTLP export
2. **Validate Data Flow**: Check that profile data is actually received
3. **Add Profile Validation**: Verify profile data format and content
4. **Backend Integration**: Test with real Grafana Pyroscope or other backends
5. **Profile Querying**: Validate profile data can be queried and visualized

## Additional Information

- **OpenTelemetry Profiles Specification**: [OTel Profiles](https://opentelemetry.io/docs/specs/otel/profiles/)
- **Tech Preview Status**: This feature is in active development and subject to change
- **eBPF Profiler**: [OpenTelemetry eBPF Profiler](https://github.com/open-telemetry/opentelemetry-ebpf-profiler)

## Cleanup

Chainsaw automatically cleans up resources after the test. The test runs in the `chainsaw-profilesignal` namespace.
