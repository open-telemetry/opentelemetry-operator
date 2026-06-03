# OpenTelemetry AWS CloudWatch Logs Exporter Test

This test demonstrates the OpenTelemetry AWS CloudWatch Logs and EMF (Embedded Metric Format) exporters for sending logs and metrics to AWS CloudWatch.

## 🎯 What This Test Does

The test validates that the AWS CloudWatch exporters can:
- Export logs to AWS CloudWatch Logs with configurable log groups and streams
- Export metrics to AWS CloudWatch using EMF (Embedded Metric Format)
- Use AWS credentials for authentication and access control
- Process both application logs via sidecar and direct metrics

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. AWS Credentials Setup
- **File**: [`create-aws-creds-secret.sh`](./create-aws-creds-secret.sh)
- **Purpose**: Creates Kubernetes secret with AWS credentials
- **Sources**: CLUSTER_PROFILE_DIR, kube-system aws-creds secret, or AWS CLI configuration
- **Output**: Creates `aws-credentials` secret in test namespace

### 2. OpenTelemetry Collector Configuration  
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: Main collector with AWS CloudWatch and EMF exporters
- **Key Features**:
  - AWS CloudWatch Logs exporter for log aggregation
  - AWS EMF (Embedded Metric Format) exporter for metrics
  - Configurable log groups and streams via template parameters
  - Batch processing for optimal performance

### 3. OpenTelemetry Sidecar Configuration
- **File**: [`otel-filelog-sidecar.yaml`](./otel-filelog-sidecar.yaml)  
- **Contains**: Sidecar collector for application log collection
- **Key Features**:
  - FileLog receiver for container log collection
  - OTLP exporter to forward logs to main collector
  - DaemonSet deployment for node-level log collection

### 4. Application Log Generator
- **File**: [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)
- **Contains**: ReplicationController and ConfigMap for log generation
- **Purpose**: Generates test logs for CloudWatch validation

### 5. Metrics Generator
- **File**: [`generate-metrics.yaml`](./generate-metrics.yaml)
- **Contains**: Job that generates test metrics
- **Purpose**: Creates metrics for EMF export validation

### 6. Verification Script
- **File**: [`check_logs_metrics.sh`](./check_logs_metrics.sh)
- **Purpose**: Validates logs and metrics are properly exported to AWS CloudWatch
- **Verification**: Checks CloudWatch Logs and CloudWatch Metrics for expected data

### 7. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create AWS Credentials Secret** - Run [`create-aws-creds-secret.sh`](./create-aws-creds-secret.sh)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)  
3. **Create OTEL Sidecar** - Deploy from [`otel-filelog-sidecar.yaml`](./otel-filelog-sidecar.yaml)
4. **Create Log Generator App** - Deploy from [`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)
5. **Generate Metrics** - Run job from [`generate-metrics.yaml`](./generate-metrics.yaml)
6. **Check CloudWatch** - Execute [`check_logs_metrics.sh`](./check_logs_metrics.sh) validation script

## 🔍 AWS CloudWatch Configuration

### Authentication:
- **Access Method**: AWS Access Key ID and Secret Access Key from Kubernetes secret
- **Region**: `us-east-2`
- **Endpoint**: `https://logs.us-east-2.amazonaws.com`

### CloudWatch Logs Configuration:
- **Log Group**: Dynamic name based on namespace (`tracing-{namespace}`)
- **Log Stream**: Dynamic name (`tracing-{namespace}-stream-emf`)
- **Raw Log**: True (preserve original log format)
- **Log Retention**: 1 day
- **Tags**: `tracing-otel: true`

### EMF (Embedded Metric Format) Configuration:
- **Namespace**: `Tracing-EMF`
- **Dimension Rollup**: `ZeroAndSingleDimensionRollup`
- **Output Destination**: `cloudwatch`
- **Resource to Telemetry Conversion**: Enabled
- **Max Retries**: 1
- **Detailed Metrics**: Disabled

## 🔍 Verification

The verification is handled by [`check_logs_metrics.sh`](./check_logs_metrics.sh), which:
- Validates log groups and streams are created in AWS CloudWatch Logs
- Confirms application logs are successfully exported and visible in CloudWatch
- Verifies metrics are exported in EMF format and visible in CloudWatch Metrics
- Uses AWS CLI commands to query CloudWatch APIs for validation

## 🧹 Cleanup

The test runs in a dynamically created namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses template bindings for dynamic log group and stream names
- Includes both logs and metrics pipelines with different exporters
- Configures OpenShift-specific RBAC and security context constraints
- Uses sidecar pattern for application log collection
- Tests complete pipeline from log generation to AWS CloudWatch storage
- Demonstrates EMF format for CloudWatch metrics with custom namespacing 