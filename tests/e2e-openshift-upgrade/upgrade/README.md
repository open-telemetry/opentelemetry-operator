# OpenTelemetry Operator Upgrade E2E Test

This directory contains end-to-end tests for verifying OpenTelemetry Operator upgrade functionality.

## Overview

The test validates that OpenTelemetry Collector and Target Allocator continue to function correctly after an OpenTelemetry Operator upgrade by:

1. **Deploying telemetry infrastructure** - Sets up OpenTelemetry collectors and Tempo for trace storage
2. **Generating test traces** - Uses telemetrygen to create sample traces
3. **Verifying trace ingestion** - Queries Tempo to confirm traces are properly stored and queryable

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

1. **Setup Phase**: Deploy OpenTelemetry collectors, Target Allocator, Tempo, and telemetrygen
2. **Upgrade Phase**: Perform OpenTelemetry Operator upgrade 
3. **Verification Phase**: Run `verify-traces` job to confirm collectors and target allocator still work
4. **Cleanup Phase**: Remove test resources
