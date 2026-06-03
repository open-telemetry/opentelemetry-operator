# OpenTelemetry Routing Connector Test

This test demonstrates the OpenTelemetry Routing connector configuration for conditionally routing telemetry data to different pipelines and backends.

## 🎯 What This Test Does

The test validates that the Routing connector can:
- Route traces to different Tempo instances based on tenant attributes
- Use conditional statements to determine routing destinations
- Support default routing for traces that don't match specific conditions
- Forward traces to red, blue, or green Tempo instances based on X-Tenant attribute

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Tempo Instances
- **File**: [`install-tempo.yaml`](./install-tempo.yaml)
- **Contains**: Three separate TempoMonolithic instances
- **Key Features**:
  - `red` - for traces with X-Tenant=red
  - `blue` - for traces with X-Tenant=blue
  - `green` - for traces with no matching tenant (default)
  - Jaeger UI enabled for trace verification

### 2. OpenTelemetry Collector with Routing Connector
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with routing connector configuration
- **Key Features**:
  - OTLP receiver for trace ingestion
  - Routing connector with conditional routing logic
  - Multiple OTLP exporters for different Tempo instances
  - Multi-pipeline architecture for tenant isolation

### 3. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating test traces with different tenant attributes
- **Key Features**:
  - Generates traces with various X-Tenant values
  - Tests routing logic for all tenant scenarios
  - Creates traces for red, blue, and default routing

### 4. Trace Verification
- **File**: [`verify-traces.yaml`](./verify-traces.yaml)
- **Contains**: Job for verifying traces are routed correctly
- **Key Features**:
  - Queries each Tempo instance for expected traces
  - Validates routing logic works correctly
  - Confirms tenant isolation

### 5. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create Tempo Instances** - Deploy from [`install-tempo.yaml`](./install-tempo.yaml)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Generate Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
4. **Verify Traces** - Execute from [`verify-traces.yaml`](./verify-traces.yaml)

## 🔍 Routing Logic

### Routing Rules:
1. **Red Route**: `route() where attributes["X-Tenant"] == "red"`
   - Routes to `traces/red` pipeline → `tempo-red` instance
   
2. **Blue Route**: `route() where attributes["X-Tenant"] == "blue"`
   - Routes to `traces/blue` pipeline → `tempo-blue` instance

3. **Default Route**: `default_pipelines: [traces/green]`
   - Routes traces without matching X-Tenant to `tempo-green` instance

### Pipeline Architecture:
- **Input Pipeline**: `traces/in` - Receives all traces via OTLP
- **Routing Connector**: Routes traces based on X-Tenant attribute
- **Output Pipelines**: 
  - `traces/red` - Sends to red Tempo instance
  - `traces/blue` - Sends to blue Tempo instance  
  - `traces/green` - Sends to green Tempo instance (default)

### Error Handling:
- **Error Mode**: `ignore` - Continue processing even if routing errors occur
- **Fallback**: Default pipeline handles traces that don't match any routing rules

## 🔍 Verification

The verification is handled by [`verify-traces.yaml`](./verify-traces.yaml), which:
- Confirms traces with `X-Tenant=red` are stored in the red Tempo instance
- Validates traces with `X-Tenant=blue` are stored in the blue Tempo instance
- Ensures traces without X-Tenant or with other values are stored in the green Tempo instance
- Verifies all routing rules function correctly and traces reach their intended destinations

## 🧹 Cleanup

The test runs in the `chainsaw-routecnctr` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses conditional routing based on trace attributes for multi-tenant scenarios
- Supports default routing for traces that don't match specific conditions
- Demonstrates fan-out from single input to multiple output pipelines
- Each routing rule can target different processing pipelines and exporters
- Enables tenant isolation by routing traces to separate backend systems
- Uses attribute-based routing with conditional expressions for flexible routing logic