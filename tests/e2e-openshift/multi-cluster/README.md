# Multi-Cluster TLS Communication Test

This test demonstrates secure multi-cluster telemetry communication between OpenTelemetry Collectors using mutual TLS (mTLS) authentication via OpenShift Routes.

## Test Overview

This test sets up:
1. **Receiver cluster**: Deploys an OpenTelemetry Collector that receives traces via OTLP with TLS
2. **Sender cluster**: Deploys an OpenTelemetry Collector that sends traces to the receiver cluster
3. **TLS certificates**: Generates and configures mTLS certificates for secure communication
4. **Tempo storage**: Stores received traces in a Tempo instance
5. **Route exposure**: Exposes the receiver via OpenShift Routes with passthrough TLS

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Tempo Operator installed
- `oc` CLI tool configured
- `openssl` and `jq` utilities
- Appropriate cluster permissions

## Configuration Resources

### Namespace Setup

The test creates two namespaces:
- `chainsaw-multi-cluster-receive` - For the receiver cluster components
- `chainsaw-multi-cluster-send` - For the sender cluster components

### TLS Certificate Generation

The test generates TLS certificates using an automated script:

**Script:** [`generate_certs.sh`](./generate_certs.sh)

The script performs:
- Creates server and CA certificates using OpenSSL
- Configures Subject Alternative Names (SANs) for OpenShift routes
- Distributes certificates to both sender and receiver namespaces
- Sets up ConfigMaps for certificate mounting

### Tempo Instance

The receiver cluster includes a Tempo instance for trace storage:

**Configuration:** [`01-create-tempo.yaml`](./01-create-tempo.yaml)

Creates a TempoMonolithic instance for trace persistence in the receiver cluster.

### OTLP Receiver Collector

The receiver collector accepts OTLP traffic over TLS and forwards to Tempo:

**Configuration:** [`02-otlp-receiver.yaml`](./02-otlp-receiver.yaml)

Configures a collector that:
- Accepts OTLP traffic over mTLS (HTTP and gRPC)
- Uses OpenShift Routes with passthrough TLS termination
- Forwards traces to the Tempo instance
- Mounts TLS certificates from ConfigMap

### RBAC Configuration

The sender requires RBAC permissions to collect cluster information:

**Configuration:** [`03-otlp-sender.yaml`](./03-otlp-sender.yaml)

Creates RBAC resources for:
- Service account for the sender collector
- ClusterRole with permissions for infrastructure and pod monitoring
- ClusterRoleBinding to associate the service account with permissions

### OTLP Sender Collector

The sender collector is created dynamically using a script that discovers the receiver routes:

**Script:** [`create_otlp_sender.sh`](./create_otlp_sender.sh)

The script performs:
- Discovers receiver route endpoints using `oc` and `jq`
- Creates sender collector configuration with discovered endpoints
- Configures mTLS authentication for secure communication
- Sets up dual exporters for HTTP and gRPC protocols

### Trace Generator Jobs

The test creates jobs to generate traces for both HTTP and gRPC protocols:

**Configuration:** [`04-generate-traces.yaml`](./04-generate-traces.yaml)

Creates two jobs:
- `generate-traces-http`: Sends 100 traces via OTLP HTTP
- `generate-traces-grpc`: Sends 100 traces via OTLP gRPC
- Both jobs include protocol-specific attributes for identification

## Deployment Steps

1. **Create namespaces:**
   ```bash
   oc apply -f 00-create-namespaces.yaml
   ```

2. **Install Tempo in receiver namespace:**
   ```bash
   oc apply -f 01-create-tempo.yaml
   ```

3. **Deploy OTLP receiver collector:**
   ```bash
   oc apply -f 02-otlp-receiver.yaml
   ```

4. **Create RBAC and sender collector:**
   ```bash
   oc apply -f 03-otlp-sender.yaml
   ```

5. **Generate and configure TLS certificates:**
   ```bash
   ./generate_certs.sh
   ```

6. **Create dynamic sender collector:**
   ```bash
   ./create_otlp_sender.sh
   ```

7. **Generate test traces:**
   ```bash
   oc apply -f 04-generate-traces.yaml
   ```

8. **Verify traces are received:**
   ```bash
   oc apply -f verify-traces.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Receiver Namespace
- **Tempo Instance**: `tempo-multicluster`
- **Collector**: `otlp-receiver-collector`
- **Routes**: HTTP and gRPC routes with passthrough TLS
- **ConfigMap**: `chainsaw-certs` with TLS certificates

### Sender Namespace
- **Service Account**: `chainsaw-multi-cluster`
- **Collector**: `otel-sender-collector`
- **Jobs**: Trace generator jobs for HTTP and gRPC
- **ConfigMap**: `chainsaw-certs` with TLS certificates

## Verification

The test verifies:
- ✅ Both namespaces are created
- ✅ Tempo instance is ready in receiver namespace
- ✅ OTLP receiver collector is deployed and ready
- ✅ Routes are created with passthrough TLS termination
- ✅ RBAC permissions are configured for sender
- ✅ OTLP sender collector is deployed and ready
- ✅ TLS certificates are generated and mounted
- ✅ Trace generation jobs complete successfully
- ✅ Traces are received and stored in Tempo

## Key Features

- **Mutual TLS Authentication**: Full mTLS configuration between sender and receiver
- **Passthrough TLS**: Routes use passthrough termination for end-to-end encryption
- **Dual Protocol Support**: Both HTTP and gRPC OTLP protocols over TLS
- **Certificate Management**: Automated certificate generation and distribution
- **Cross-Cluster Communication**: Secure telemetry communication between clusters
- **Route Discovery**: Dynamic discovery of receiver routes for sender configuration

## Configuration Notes

- Routes use `passthrough` termination to maintain end-to-end TLS
- Certificates include Subject Alternative Names (SANs) for OpenShift route domains
- The sender collector dynamically discovers receiver route endpoints
- Both HTTP (port 443) and gRPC (port 443) endpoints are exposed via routes
- TLS certificates are shared between sender and receiver via ConfigMaps 