# OpenTelemetry OIDC Authentication Extension Test

This test demonstrates the OpenTelemetry OIDC Authentication Extension configuration for securing collector-to-collector communication using OAuth2/OIDC protocols.

## 🎯 What This Test Does

The test validates that the OIDC Authentication Extension can:
- Secure OTLP communication between collectors using OIDC authentication
- Use Hydra as an OAuth2/OIDC provider for token management
- Configure server-side OIDC authentication for incoming requests
- Configure client-side OAuth2 authentication for outgoing requests
- Handle TLS certificate validation for secure communication

## 📋 Test Resources

The test uses the following key resources that are included in this directory:

### 1. Hydra OIDC Provider
- **File**: [`install-hydra.yaml`](./install-hydra.yaml)
- **Contains**: Deployment and Service for Ory Hydra OAuth2/OIDC provider
- **Key Features**:
  - OAuth2/OIDC token issuer and validation service
  - Provides authentication services for the test environment
  - Configured for development/testing scenarios

### 2. Hydra Configuration Setup
- **File**: [`setup-hydra.yaml`](./setup-hydra.yaml)
- **Contains**: Job for configuring OAuth2 client in Hydra
- **Key Features**:
  - Creates OAuth2 client registration in Hydra
  - Configures client credentials and audience settings
  - Prepares authentication environment for collectors

### 3. Certificate Management
- **File**: [`generate_certs.sh`](./generate_certs.sh)
- **Purpose**: Generates TLS certificates for secure communication
- **Creates**: CA certificate, server certificate, and private keys
- **Storage**: Certificates stored in ConfigMap for collector access

### 4. OIDC Server Collector
- **File**: [`install-otel-oidc-server.yaml`](./install-otel-oidc-server.yaml)
- **Contains**: OpenTelemetryCollector with OIDC authentication extension
- **Key Features**:
  - OIDC extension for validating incoming authentication tokens
  - TLS-secured OTLP receiver with certificate authentication
  - Debug exporter for trace verification
  - Certificate volume mounts for TLS operation

### 5. OIDC Client Collector
- **File**: [`install-otel-oidc-client.yaml`](./install-otel-oidc-client.yaml)
- **Contains**: OpenTelemetryCollector with OAuth2 client extension
- **Key Features**:
  - OAuth2 client extension for acquiring access tokens
  - TLS-secured OTLP exporter with certificate validation
  - OTLP receiver for trace ingestion
  - Certificate volume mounts for TLS operation

### 6. Trace Generator
- **File**: [`generate-traces.yaml`](./generate-traces.yaml)
- **Contains**: Job for generating test traces
- **Key Features**:
  - Generates traces to test authenticated communication
  - Targets the client collector endpoint
  - Validates end-to-end authentication flow

### 7. Verification Script
- **File**: [`check_logs.sh`](./check_logs.sh)
- **Purpose**: Validates that OIDC authentication works correctly
- **Verification Criteria**:
  - Confirms traces are successfully transmitted with authentication
  - Validates server collector receives authenticated requests
  - Ensures OAuth2 client can obtain and use access tokens

### 8. Chainsaw Test Definition
- **File**: [`chainsaw-test.yaml`](./chainsaw-test.yaml)
- **Contains**: Complete test workflow orchestration
- **Includes**: Test steps, assertions, and cleanup procedures

## 🚀 Test Steps

The test follows this sequence as defined in [`chainsaw-test.yaml`](./chainsaw-test.yaml):

1. **Install Hydra** - Deploy from [`install-hydra.yaml`](./install-hydra.yaml)
2. **Setup OAuth2 Client** - Configure from [`setup-hydra.yaml`](./setup-hydra.yaml)
3. **Generate Certificates** - Execute [`generate_certs.sh`](./generate_certs.sh) and create ConfigMap
4. **Create OIDC Server Collector** - Deploy from [`install-otel-oidc-server.yaml`](./install-otel-oidc-server.yaml)
5. **Create OIDC Client Collector** - Deploy from [`install-otel-oidc-client.yaml`](./install-otel-oidc-client.yaml)
6. **Generate Traces** - Run from [`generate-traces.yaml`](./generate-traces.yaml)
7. **Verify Authentication** - Execute [`check_logs.sh`](./check_logs.sh) validation script

## 🔍 Authentication Flow

### OIDC Server Configuration:
- **Extension**: `oidc`
- **Issuer URL**: `http://hydra:4444`
- **Audience**: `tenant1-oidc-client`
- **TLS**: Uses server certificate for secure communication
- **Authentication**: Validates incoming OIDC tokens

### OAuth2 Client Configuration:
- **Extension**: `oauth2client`
- **Client ID**: `tenant1-oidc-client`
- **Client Secret**: `ZXhhbXBsZS1hcHAtc2VjcmV0` (base64 encoded)
- **Token URL**: `http://hydra:4444/oauth2/token`
- **Audience**: `tenant1-oidc-client`
- **TLS**: Uses CA certificate for server validation

### Certificate Management:
- **CA Certificate**: Used for TLS validation
- **Server Certificate**: Used by server collector for TLS termination
- **Certificate Storage**: Mounted from ConfigMap to both collectors

## 🔍 Verification

The verification is handled by [`check_logs.sh`](./check_logs.sh), which:
- Confirms Hydra OIDC provider is running and configured
- Validates OAuth2 client is properly registered with Hydra
- Checks TLS certificates are generated and mounted correctly
- Ensures OIDC server collector can authenticate incoming requests
- Verifies OAuth2 client collector can obtain and use access tokens
- Confirms traces are successfully transmitted with proper authentication

## 🧹 Cleanup

The test runs in the `chainsaw-oidcauthextension` namespace and all resources are cleaned up automatically when the test completes.

## 📝 Key Configuration Notes

- Uses fixed namespace (`chainsaw-oidcauthextension`) for OIDC authentication setup
- Demonstrates end-to-end OAuth2/OIDC authentication between collectors
- Integrates TLS security with OIDC authentication for comprehensive security
- Uses Hydra as a standard-compliant OAuth2/OIDC provider
- Shows both server-side authentication validation and client-side token acquisition
- Validates secure collector-to-collector communication patterns
- Requires certificate generation and proper volume mounting for TLS operation