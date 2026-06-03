# OpenTelemetry Google Managed Prometheus Exporter Test

This test demonstrates OpenTelemetry Collector configurations for exporting metrics to Google Cloud Managed Prometheus using two authentication methods.

## 🎯 What This Test Does

The test validates two authentication approaches for Google Managed Prometheus integration:
- **Service Account (SA)**: Uses Google Cloud Service Account key for authentication
- **Workload Identity Federation (WIF)**: Uses secure authentication without long-lived credentials

Both approaches demonstrate:
- Exporting OpenTelemetry metrics to Google Cloud Monitoring
- Using kubernetes attributes processor for metadata enrichment
- Avoiding attribute name collisions with Prometheus reserved names
- Processing both OTLP and Kubelet stats metrics

## 📋 Test Resources

The test uses the following key resources organized in two authentication method directories:

### 1. Service Account (SA) Authentication Method

#### Setup and Cleanup Scripts
- **File**: [`gcp-sa/gcp-sa-create.sh`](./gcp-sa/gcp-sa-create.sh)
- **Purpose**: Creates Google Cloud Service Account and configures authentication
- **File**: [`gcp-sa/gcp-sa-delete.sh`](./gcp-sa/gcp-sa-delete.sh)
- **Purpose**: Cleans up Service Account resources

#### OpenTelemetry Collector Configuration
- **File**: [`gcp-sa/otel-sa-collector.yaml`](./gcp-sa/otel-sa-collector.yaml)
- **Contains**: OpenTelemetryCollector with Service Account authentication
- **Key Features**:
  - Service Account key-based authentication via mounted secret
  - OTLP HTTP exporter to Google Cloud Monitoring
  - Comprehensive processing pipeline with collision avoidance
  - Kubernetes attributes processor for metadata enrichment

#### Metrics Generators
- **File**: [`gcp-sa/metrics-generator-app.yaml`](./gcp-sa/metrics-generator-app.yaml)
- **Contains**: Application metrics generator for testing
- **File**: [`gcp-sa/metrics-generator-kubeletstats.yaml`](./gcp-sa/metrics-generator-kubeletstats.yaml)
- **Contains**: Kubelet stats metrics collector for testing

### 2. Workload Identity Federation (WIF) Authentication Method

#### Setup and Cleanup Scripts
- **File**: [`gcp-wif/gcp-wif-create.sh`](./gcp-wif/gcp-wif-create.sh)
- **Purpose**: Sets up Workload Identity Federation authentication
- **File**: [`gcp-wif/gcp-wif-delete.sh`](./gcp-wif/gcp-wif-delete.sh)
- **Purpose**: Cleans up WIF resources

#### OpenTelemetry Collector Configuration
- **File**: [`gcp-wif/otel-wif-collector.yaml`](./gcp-wif/otel-wif-collector.yaml)
- **Contains**: OpenTelemetryCollector with Workload Identity Federation
- **Key Features**:
  - Secure authentication without long-lived credentials
  - Same processing pipeline as SA method
  - Enhanced security through WIF integration

#### Metrics Generators
- **File**: [`gcp-wif/metrics-generator-app.yaml`](./gcp-wif/metrics-generator-app.yaml)
- **Contains**: Application metrics generator for WIF testing
- **File**: [`gcp-wif/metrics-generator-kubeletstats.yaml`](./gcp-wif/metrics-generator-kubeletstats.yaml)
- **Contains**: Kubelet stats metrics collector for WIF testing

## 🚀 Test Steps

The test is organized into two separate authentication methods, each with its own workflow:

### Service Account Method:
1. **Create GCP Service Account** - Execute [`gcp-sa/gcp-sa-create.sh`](./gcp-sa/gcp-sa-create.sh)
2. **Deploy OTEL Collector** - Deploy from [`gcp-sa/otel-sa-collector.yaml`](./gcp-sa/otel-sa-collector.yaml)
3. **Generate App Metrics** - Deploy from [`gcp-sa/metrics-generator-app.yaml`](./gcp-sa/metrics-generator-app.yaml)
4. **Generate Kubelet Metrics** - Deploy from [`gcp-sa/metrics-generator-kubeletstats.yaml`](./gcp-sa/metrics-generator-kubeletstats.yaml)
5. **Verify Export** - Check metrics appear in Google Cloud Monitoring
6. **Cleanup** - Execute [`gcp-sa/gcp-sa-delete.sh`](./gcp-sa/gcp-sa-delete.sh)

### Workload Identity Federation Method:
1. **Setup WIF** - Execute [`gcp-wif/gcp-wif-create.sh`](./gcp-wif/gcp-wif-create.sh)
2. **Deploy OTEL Collector** - Deploy from [`gcp-wif/otel-wif-collector.yaml`](./gcp-wif/otel-wif-collector.yaml)
3. **Generate App Metrics** - Deploy from [`gcp-wif/metrics-generator-app.yaml`](./gcp-wif/metrics-generator-app.yaml)
4. **Generate Kubelet Metrics** - Deploy from [`gcp-wif/metrics-generator-kubeletstats.yaml`](./gcp-wif/metrics-generator-kubeletstats.yaml)
5. **Verify Export** - Check metrics appear in Google Cloud Monitoring
6. **Cleanup** - Execute [`gcp-wif/gcp-wif-delete.sh`](./gcp-wif/gcp-wif-delete.sh)

## 🔍 Configuration Highlights

### Authentication:
- **SA Method**: Uses mounted service account key file
- **WIF Method**: Uses secure authentication without long-lived credentials

### Processing Pipeline:
1. **k8sattributes** - Enriches metrics with Kubernetes metadata
2. **memory_limiter** - Prevents memory exhaustion  
3. **resource/set_gcp_defaults** - Adds GCP project and location information
4. **transform/collision** - Avoids Prometheus reserved attribute conflicts
5. **metricstarttime** - Handles metric reset points
6. **batch** - Optimizes export efficiency

### Collision Avoidance:
The transform processor renames attributes that conflict with Prometheus reserved names:
- `location` → `exported_location`
- `cluster` → `exported_cluster`
- `namespace` → `exported_namespace`
- `job` → `exported_job`
- `instance` → `exported_instance`
- `project_id` → `exported_project_id`

## 🧹 Cleanup

Each authentication method includes cleanup scripts to remove Google Cloud resources and Kubernetes objects.

## 📝 Key Configuration Notes

- **Two Authentication Methods**: Demonstrates both SA and WIF approaches for different security requirements
- **Collision Prevention**: Transforms attributes to avoid Prometheus reserved name conflicts
- **Kubernetes Integration**: Enriches metrics with comprehensive Kubernetes metadata
- **Resource Management**: Includes memory limiting and batch processing for efficiency
- **Security Options**: WIF method provides enhanced security without storing credentials 