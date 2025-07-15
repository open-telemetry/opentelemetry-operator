# OpenTelemetry Operator Upgrade E2E Test

This directory contains end-to-end tests for verifying OpenTelemetry Operator upgrade functionality.

## Overview

The test validates that OpenTelemetry Collector and Target Allocator continue to function correctly after an OpenTelemetry Operator upgrade by:

1. **Deploying telemetry infrastructure** - Sets up OpenTelemetry Collectors with Target Allocator enabled, Tempo for trace storage, and Prometheus metrics collection
2. **Generating test data** - Uses telemetrygen to create sample traces and metrics
3. **Verifying data collection** - Queries Tempo to confirm traces are properly stored and validates Target Allocator metrics endpoints
4. **Testing upgrade resilience** - Ensures both OpenTelemetry Collector and Target Allocator remain functional after operator upgrade

**TraceQL Query**: `{ resource.service.name="telemetrygen" }`

## Prerequisites

- OpenShift cluster version >= 4.12
- File based catalog image used for upgrading the operator.
- Operator upgrade CSV and version info.

**Note**: The test automatically installs the OpenTelemetry Operator, creates operand instances (OpenTelemetry Collector, Target Allocator), deploys Tempo, and runs telemetrygen as part of the test execution.

## Running the Test

The test is designed to be run as part of the chainsaw test suite and requires specific values to be passed for the upgrade configuration.

### Required Parameters

The test requires the following values to be provided:
- `upgrade_fbc_image`: File-based catalog (FBC) image for upgrading the operator
- `upgrade_otel_version`: Target OpenTelemetry Operator version 
- `upgrade_operator_csv_name`: CSV name for the operator upgrade

### Running the Test

Use one of the following methods to run the upgrade test:

#### Method 1: Using heredoc (recommended)
```bash
chainsaw test tests/e2e-openshift-upgrade --values - <<EOF
upgrade_fbc_image: brew.registry.redhat.io/rh-osbs/iib:986879
upgrade_otel_version: 0.127.0
upgrade_operator_csv_name: opentelemetry-operator.v0.127.0-1
EOF
```

#### Method 2: Using values file
Create a `values.yaml` file:
```yaml
upgrade_fbc_image: brew.registry.redhat.io/rh-osbs/iib:986879
upgrade_otel_version: 0.127.0
upgrade_operator_csv_name: opentelemetry-operator.v0.127.0-1
```

Then run:
```bash
chainsaw test tests/e2e-openshift-upgrade --values values.yaml
```

**Note**: Replace the example values with your specific upgrade target versions and catalog image.

## Test Flow

1. **Setup Phase**: 
   - Deploy OpenTelemetry Operators from marketplace
   - Create Tempo Monolithic instance for trace storage
   - Enable user workload monitoring
   - Deploy OpenTelemetry Collector for metrics collection
   - Deploy OpenTelemetry Collector with Target Allocator enabled
   - Generate test traces and metrics using telemetrygen

2. **Pre-Upgrade Verification**:
   - Verify traces are being ingested correctly by Tempo
   - Validate Target Allocator metrics endpoints are accessible
   - Assert Target Allocator-specific metrics (collectors discovered, targets allocated, etc.)

3. **Upgrade Phase**: 
   - Create upgrade catalog for OpenTelemetry Operator
   - Perform OpenTelemetry Operator upgrade to specified version

4. **Post-Upgrade Verification**:
   - Assert OpenTelemetry Collector and Target Allocator pods are ready
   - Re-generate test traces and metrics
   - Verify traces are still being ingested correctly by Tempo
   - Re-validate Target Allocator metrics endpoints and specific metrics

5. **Target Allocator Metrics Validation**:
   - Validate core Target Allocator metrics such as:
     - `opentelemetry_allocator_collectors_allocatable`
     - `opentelemetry_allocator_collectors_discovered`
     - `opentelemetry_allocator_targets`
     - `opentelemetry_allocator_targets_per_collector`
     - And other Target Allocator-specific metrics

6. **Cleanup Phase**: Remove test resources
