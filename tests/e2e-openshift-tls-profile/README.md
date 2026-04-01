# OpenShift TLS Security Profile Integration Test

This test verifies that the OpenTelemetry Operator correctly injects and enforces the cluster-wide TLS security profile from the OpenShift APIServer into OpenTelemetry Collector components.

## Test Overview

The test validates end-to-end TLS profile injection by:

1. Deploying a multi-component collector with TLS-enabled receivers and exporters
2. Verifying the cluster's Intermediate TLS profile is injected into the collector ConfigMap
3. Functionally verifying TLS settings via `nmap ssl-enum-ciphers`
4. Changing the cluster TLS profile (to Modern or Custom) and verifying the collector picks up the change
5. Verifying that user-specified TLS values are preserved and not overwritten by the operator

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed with TLS profile support (`--tls-use-cluster-profile` and `--tls-configure-operands` flags)
- `oc` and `kubectl` CLI tools configured
- `chainsaw` test runner (v0.2.13+)
- `yq` (for YAML-to-JSON conversion in verification scripts)
- Cluster admin permissions (test patches the cluster-wide APIServer resource)

## How TLS Profile Injection Works

The operator applies the cluster's TLS security profile at **reconciliation time** during ConfigMap generation, not in the webhook. This means:

- The CR (OpenTelemetryCollector) does **not** store TLS profile settings (`min_version`, `cipher_suites`)
- The generated ConfigMap contains the injected TLS settings
- When the cluster TLS profile changes, the operator restarts and regenerates ConfigMaps with the new profile
- Verification scripts read from the **ConfigMap**, not the CR

## Test Scenarios

| ID | Scenario | Step |
|----|----------|------|
| S1 | Intermediate profile injected into components with TLS block | 02 |
| S2 | No injection for components without TLS block | 02 |
| S3 | Multi-protocol receiver (OTLP gRPC + HTTP) handled correctly | 02 |
| S4 | Single-endpoint receiver (Zipkin) handled correctly | 02 |
| S5 | Exporter with TLS (Prometheus) handled correctly | 02 |
| S6 | User-specified TLS values preserved | 10 |
| S7 | Changed profile (Modern/Custom) injected after update | 06 |
| S8 | Dynamic profile change detected after operator restart | 05-06 |

## Test Steps

### Step 0: Setup
- Generate a self-signed TLS certificate for the collector
- Deploy a `tls-scanner` pod with `nmap` for functional TLS verification

### Step 1: Deploy Collector
- Apply a collector with TLS-enabled OTLP (gRPC + HTTP), Zipkin receiver, and Prometheus exporter
- Assert deployment reaches ready state

### Step 2: Verify Intermediate Profile (ConfigMap)
- **Script:** [`02-verify-intermediate.sh`](tls-profile/02-verify-intermediate.sh)
- Verifies `min_version: "1.2"` and `cipher_suites` are injected into the ConfigMap for:
  - OTLP gRPC receiver (has TLS block)
  - Zipkin receiver (has TLS block)
  - Prometheus exporter (has TLS block)
- Verifies no TLS injection for:
  - OTLP HTTP receiver (no TLS block)
  - Debug exporter (no TLS block)

### Step 3: Functional TLS Verification (Intermediate)
- **Script:** [`03-verify-nmap-intermediate.sh`](tls-profile/03-verify-nmap-intermediate.sh)
- Uses `tls-scanner` to verify the collector accepts TLS connections
- Uses `nmap ssl-enum-ciphers` to verify both TLSv1.2 and TLSv1.3 are offered

### Step 4: Patch APIServer TLS Profile
- **Script:** [`04-patch-profile.sh`](tls-profile/04-patch-profile.sh)
- Saves current APIServer TLS profile for later restoration
- Patches to **Modern** (TLS 1.3 only) on OCP >= 4.14
- Falls back to **Custom** (TLS 1.0 + GCM ciphers) on older OCP versions
- Stores expected values in a ConfigMap for subsequent verification steps

### Step 5: Wait for Operator to Re-reconcile with New Profile
- Waits for the operator pod to be ready after the TLS profile change triggers a restart
- Polls the deployment's volume reference until it points to a ConfigMap with the expected `min_version`
- Waits for the deployment rollout to complete with the new pods
- The operator automatically re-reconciles existing collectors on restart, regenerating ConfigMaps with the updated TLS settings and updating the deployment's volume reference

### Step 6: Verify Changed Profile (ConfigMap)
- **Script:** [`06-verify-changed-profile.sh`](tls-profile/06-verify-changed-profile.sh)
- Reads expected values from the `tls-profile-expected` ConfigMap
- Verifies `min_version` and `cipher_suites` match the new profile

### Step 7: Functional TLS Verification (Changed)
- **Script:** [`07-verify-nmap-changed.sh`](tls-profile/07-verify-nmap-changed.sh)
- Verifies via `nmap` that the collector enforces the changed profile
- For Modern: only TLSv1.3 (no TLSv1.2)
- For Custom TLS 1.0: TLSv1.2+ (GCM ciphers require TLS 1.2)

### Step 8: Revert APIServer TLS Profile
- **Script:** [`11-revert-apiserver.sh`](tls-profile/11-revert-apiserver.sh)
- Removes the `tlsSecurityProfile` from the APIServer spec

### Step 9: Deploy User Override Collector
- Waits for operator readiness after TLS profile revert
- Deploys a collector with user-specified TLS overrides:
  - gRPC: user sets `min_version: "1.3"`, does NOT set `cipher_suites`
  - HTTP: user sets `cipher_suites`, does NOT set `min_version`

### Step 10: Verify User Overrides
- **Script:** [`10-verify-user-override.sh`](tls-profile/10-verify-user-override.sh)
- Verifies user-specified `min_version: "1.3"` is preserved on gRPC
- Verifies cluster `cipher_suites` are injected on gRPC (user didn't set them)
- Verifies cluster `min_version: "1.2"` is injected on HTTP (user didn't set it)
- Verifies user-specified `cipher_suites` are preserved on HTTP

## Running the Test

```bash
# Run with resource cleanup disabled (for debugging)
chainsaw test --skip-delete tests/e2e-openshift-tls-profile

# Run with automatic cleanup
chainsaw test tests/e2e-openshift-tls-profile
```

## Manual Cleanup

If the test fails or is run with `--skip-delete`, clean up resources manually:

```bash
kubectl delete namespace chainsaw-otel-tls-profile --ignore-not-found
kubectl delete clusterrole tls-scanner-pods-reader-otel-tls-profile --ignore-not-found
kubectl delete clusterrolebinding tls-scanner-pods-reader-otel-tls-profile --ignore-not-found
kubectl delete scc tls-scanner-scc-otel-tls-profile --ignore-not-found
oc patch apiserver cluster --type json \
  -p '[{"op":"remove","path":"/spec/tlsSecurityProfile"}]' 2>/dev/null || true
```

## Important Notes

- This test **patches the cluster-wide APIServer TLS profile**. It runs with `concurrent: false` and includes catch handlers to revert the APIServer on failure.
- The operator restarts when the TLS profile changes. Steps 5 and 9 include polling logic to wait for the operator pod to be ready before proceeding.
- The test uses a `tls-scanner` pod with a privileged SCC for `nmap` scanning. The SCC is scoped to the test namespace service account.
