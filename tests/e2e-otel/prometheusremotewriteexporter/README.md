# OpenTelemetry Prometheus Remote Write Exporter Test

This test demonstrates the OpenTelemetry Prometheus Remote Write exporter configuration for sending metrics to Prometheus.

## 🎯 What This Test Does

The test validates that the Prometheus Remote Write exporter can:
- Export metrics to Prometheus using the remote write protocol
- Use TLS authentication with custom CA certificates
- Convert OpenTelemetry metrics to Prometheus format
- Send metrics over HTTPS with proper certificate validation

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Prometheus Deployment
- **File**: [`deploy-prometheus.yaml`](./deploy-prometheus.yaml)
- **Contains**: Prometheus instance with remote write receiver enabled
- **Key Features**:
  - Remote write receiver enabled for accepting metrics
  - TLS configuration with custom CA certificate
  - Prometheus storage for metric persistence
  - Service for external access

### 2. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with Prometheus Remote Write exporter
- **Key Features**:
  - OTLP receiver for metric ingestion
  - Prometheus Remote Write exporter with TLS configuration
  - Custom CA certificate mounted from Kubernetes secret
  - Resource to telemetry conversion enabled

### 3. Metrics Generator
- **File**: [`generate-metrics.yaml`](./generate-metrics.yaml)
- **Contains**: Job for generating test metrics
- **Key Features**:
  - Generates test metrics using telemetrygen
  - Targets OpenTelemetry collector endpoint
  - Creates metrics for remote write validation

### 4. Metrics Verification
- **File**: [`check-metrics.yaml`](./check-metrics.yaml)
- **Contains**: Job for verifying metrics in Prometheus
- **Key Features**:
  - Queries Prometheus API for metric verification
  - Validates metrics are properly stored
  - Confirms remote write protocol functionality

### 5. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Deploy Prometheus** - Deploy from [`deploy-prometheus.yaml`](./deploy-prometheus.yaml)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Generate Metrics** - Run from [`generate-metrics.yaml`](./generate-metrics.yaml)
4. **Verify Metrics** - Execute from [`check-metrics.yaml`](./check-metrics.yaml)

## 🔍 Prometheus Remote Write Configuration

### TLS Configuration:
- **CA Certificate**: Custom CA certificate mounted from Kubernetes secret
- **Certificate File**: `/certs/ca.crt` mounted from secret volume
- **Endpoint**: `https://prometheus:9090/api/v1/write`
- **Verification**: TLS certificate validation enabled

### Conversion Settings:
- **Resource to Telemetry Conversion**: Enabled to include resource attributes as labels
- **Protocol**: Prometheus remote write protocol over HTTPS
- **Target**: Prometheus instance with remote write receiver

### Secret Management:
- CA certificate stored in Kubernetes secret `prometheus-ca`
- Certificate mounted as volume in collector pod
- Read-only access for security

## 🔍 Verification

The verification is handled by [`check-metrics.yaml`](./check-metrics.yaml), which:
- Queries Prometheus API to confirm metrics are stored
- Validates that metrics sent via remote write protocol are accessible
- Ensures TLS authentication works correctly
- Confirms resource attributes are converted to Prometheus labels

## 🧹 Cleanup

The test runs in the default namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses HTTPS endpoint with custom CA certificate for secure communication
- Mounts CA certificate from Kubernetes secret for TLS validation
- Enables resource to telemetry conversion for comprehensive labeling
- Demonstrates secure metric export to Prometheus with certificate-based authentication
- Tests complete pipeline from OTLP ingestion to Prometheus storage
- Validates Prometheus remote write protocol implementation