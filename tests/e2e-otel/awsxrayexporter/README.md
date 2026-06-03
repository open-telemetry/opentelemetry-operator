# OpenTelemetry AWS X-Ray Exporter Test

This test demonstrates the OpenTelemetry AWS X-Ray exporter configuration for sending traces to AWS X-Ray service.

## 🎯 What This Test Does

The test validates that the AWS X-Ray exporter can:
- Export traces to AWS X-Ray using AWS credentials
- Configure worker processes and retry settings for reliable delivery
- Include telemetry metadata and AWS log group associations
- Process traces from HotROD application and telemetry generators

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. AWS Credentials Setup Script
- **File**: [`create-aws-creds-secret.sh`](./create-aws-creds-secret.sh)
- **Purpose**: Creates Kubernetes secret with AWS credentials for X-Ray authentication
- **Required Environment Variables**:
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`

### 2. OpenTelemetry Collector Configuration
- **File**: [`otel-collector.yaml`](./otel-collector.yaml)
- **Contains**: OpenTelemetryCollector with AWS X-Ray exporter configuration
- **Key Features**:
  - AWS X-Ray exporter with performance tuning (2 workers, 30s timeout, 2 retries)
  - Region-specific endpoint configuration (us-east-2)
  - Telemetry metadata and AWS log group associations
  - OTLP receiver for trace ingestion

### 3. HotROD Application Deployment
- **File**: [`install-hotrod.yaml`](./install-hotrod.yaml)
- **Contains**: HotROD (Rides on Demand) demo application deployment
- **Purpose**: Generates realistic distributed traces for X-Ray export testing

### 4. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job that generates additional test traces
- **Purpose**: Creates synthetic trace data to validate X-Ray export functionality

### 5. Verification Script
- **File**: [`check_traces.sh`](./check_traces.sh)
- **Purpose**: Validates that traces are successfully exported to AWS X-Ray
- **Verification**: Checks AWS CloudWatch X-Ray console for received traces

### 6. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Create AWS Credentials Secret** - Execute [`create-aws-creds-secret.sh`](./create-aws-creds-secret.sh)
2. **Create OTEL Collector** - Deploy from [`otel-collector.yaml`](./otel-collector.yaml)
3. **Install HotROD App** - Deploy from [`install-hotrod.yaml`](./install-hotrod.yaml)
4. **Generate Traces** - Run job from [`generate-traces.yaml`](./generate-traces.yaml)
5. **Check AWS X-Ray** - Execute [`check_traces.sh`](./check_traces.sh) validation script

## 🔍 AWS X-Ray Configuration

### Authentication:
- **Access Method**: AWS Access Key ID and Secret Access Key from Kubernetes secret
- **Region**: `us-east-2`
- **Endpoint**: `https://xray.us-east-2.amazonaws.com`

### Performance Settings:
- **Number of Workers**: 2 parallel workers for trace export
- **Request Timeout**: 30 seconds per request
- **Max Retries**: 2 retry attempts on failure
- **SSL Verification**: Enabled (`no_verify_ssl: false`)

### X-Ray Specific Settings:
- **Local Mode**: Disabled (sends directly to AWS X-Ray service)
- **Index All Attributes**: Disabled (selective attribute indexing)
- **AWS Log Groups**: Associated with `ikanse=tracing-test` log group

### Telemetry Configuration:
- **Telemetry Enabled**: True
- **Include Metadata**: True
- **Hostname**: `ocp-otel-collector`
- **Instance ID**: `otel-collector-xray`

## 🔍 Verification

The verification is handled by [`check_traces.sh`](./check_traces.sh), which:
- Validates traces are successfully exported to AWS X-Ray
- Checks AWS CloudWatch X-Ray console for received traces
- Confirms traces can be queried and visualized in AWS X-Ray service

## 🧹 Cleanup

The test runs in the default namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses AWS credentials from Kubernetes secret for authentication
- Configured for us-east-2 region with specific X-Ray endpoint
- Includes performance tuning with multiple workers and retry logic
- Associates traces with specific AWS log groups for correlation
- Enables telemetry reporting with collector identification
- Tests complete pipeline from OTLP ingestion to AWS X-Ray storage 