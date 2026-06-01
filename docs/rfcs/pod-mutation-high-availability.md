# Pod Mutation High Availability

## Summary

This RFC proposes enabling high availability (HA) for the pod mutation webhook by deploying it as a standalone Deployment separate from the operator. This allows independent scaling of the webhook without affecting the operator's leader election or requiring changes to OLM-managed deployments.

## Motivation

The OpenTelemetry Operator includes a pod mutation webhook (`/mutate-v1-pod`) that handles:
- Auto-instrumentation injection
- Sidecar collector injection

In production environments, this webhook is critical infrastructure - if it's unavailable or slow, pod creation across the cluster is impacted. Currently, the webhook runs as part of the operator deployment, which has limitations:

1. **Single replica by default**: The operator uses leader election, and while the webhook doesn't require it, they share the same deployment.

2. **OLM scaling limitations**: On OpenShift with OLM, users cannot scale CSV-managed deployments - OLM continuously reconciles replicas back to the CSV-defined count. `Subscription.spec.config` doesn't support `replicas`.

3. **Coupled lifecycle**: Scaling the webhook means scaling the entire operator, which is wasteful since controllers only need one active instance.

### Goals

- Enable independent scaling of the pod mutation webhook
- Support HA deployments on OpenShift (OLM) without manual intervention
- Maintain backward compatibility for existing deployments
- Ensure proper cleanup when the operator is uninstalled
- Handle TLS certificate provisioning automatically

### Non-Goals

- Changing how CR validation/mutation webhooks work (they remain in the operator)
- Supporting webhook HA without cert-manager or OpenShift service-ca
- Horizontal Pod Autoscaler (HPA) integration

## Design

### Architecture

When `ENABLE_STANDALONE_WEBHOOK=true`, the operator creates and manages:

```
┌─────────────────────────────────────────────────────────────────┐
│                    opentelemetry-operator-system                │
│                                                                 │
│  ┌──────────────────────────┐    ┌──────────────────────────┐  │
│  │ opentelemetry-operator-  │    │ opentelemetry-operator-  │  │
│  │ controller-manager       │    │ pod-webhook              │  │
│  │ (1 replica, leader       │    │ (N replicas, no leader   │  │
│  │  election)               │    │  election)               │  │
│  │                          │    │                          │  │
│  │ - CR controllers         │    │ - /mutate-v1-pod         │  │
│  │ - CR webhooks            │    │   (instrumentation +     │  │
│  │ - Webhook deployment     │    │    sidecar injection)    │  │
│  │   reconciliation         │    │                          │  │
│  └──────────────────────────┘    └──────────────────────────┘  │
│              │                              ▲                   │
│              │ manages                      │                   │
│              └──────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘
```

### Resources Created

#### OpenShift with OLM (Operator-Managed)

The operator creates and manages the following resources:

1. **Service** (`opentelemetry-operator-pod-webhook`)
   - Exposes port 443 targeting the webhook pods
   - Includes `service.beta.openshift.io/serving-cert-secret-name` annotation for automatic TLS provisioning

2. **Deployment** (`opentelemetry-operator-pod-webhook`)
   - Runs the operator binary with `auto-instrumentation` subcommand
   - Configurable replicas via `STANDALONE_WEBHOOK_REPLICAS` env var
   - Mounts TLS certificates from the service-ca generated secret

3. **MutatingWebhookConfiguration** (`opentelemetry-operator-pod-webhook`)
   - Registers `/mutate-v1-pod` path
   - Includes `service.beta.openshift.io/inject-cabundle=true` annotation for CA bundle injection

#### Kubernetes with cert-manager (User-Managed)

On Kubernetes, the operator does NOT create webhook resources. Users who want HA can deploy the webhook manually using manifests from the Helm chart repository. The required resources are:

1. **Deployment** - Webhook deployment with configurable replicas
2. **Service** - Exposes webhook on port 443
3. **MutatingWebhookConfiguration** - With `cert-manager.io/inject-ca-from` annotation
4. **Certificate** - cert-manager Certificate CR for TLS provisioning

The operator does NOT create cert-manager resources in code.

### Certificate Management

#### OpenShift (service-ca)

On OpenShift, the service-ca operator handles certificate provisioning automatically:

1. The Service annotation `service.beta.openshift.io/serving-cert-secret-name` triggers secret creation
2. The MutatingWebhookConfiguration annotation `service.beta.openshift.io/inject-cabundle=true` triggers CA bundle injection
3. Certificates are valid for 26 months and auto-rotated when < 13 months remain
4. Controller-runtime's `CertWatcher` automatically reloads certificates without pod restart

#### Kubernetes with cert-manager (Optional Manual Deployment)

On vanilla Kubernetes with cert-manager, the standalone webhook is **NOT operator-managed**. This is an optional feature - users who want HA can deploy the webhook manually.

Manifests for Kubernetes deployment may be provided in the [opentelemetry-helm-charts](https://github.com/open-telemetry/opentelemetry-helm-charts) repository. The deployment would include:

1. Deployment - Webhook deployment with configurable replicas
2. Service - Exposes webhook on port 443
3. MutatingWebhookConfiguration - With `cert-manager.io/inject-ca-from` annotation
4. Certificate - cert-manager Certificate CR for TLS provisioning

cert-manager provisions the TLS secret, and controller-runtime's `CertWatcher` automatically reloads certificates without pod restart.

This approach:
- Keeps operator code simple (only OpenShift/OLM path is operator-managed)
- Leverages Helm chart flexibility for Kubernetes deployments
- Gives users full control over webhook configuration and scaling
- Avoids operator needing RBAC for cert-manager resources

#### Certificate Rotation

Both approaches support automatic certificate rotation without pod restarts:

- Controller-runtime's webhook server uses `certwatcher.CertWatcher`
- CertWatcher uses `fsnotify` to watch certificate files and polls every 10 seconds as fallback
- When certificates change, the in-memory cache is updated via `GetCertificate` callback
- This is equivalent to or better than OLM's approach (which requires deployment restart)

#### TLS Profile Changes (OpenShift Only)

On OpenShift, when the cluster's TLS security profile changes (e.g., from Intermediate to Modern via the APIServer CR), the standalone webhook deployment automatically restarts to apply the new TLS settings:

1. A `SecurityProfileWatcher` monitors the APIServer CR for TLS profile changes
2. When a change is detected, the watcher triggers a graceful shutdown
3. Kubernetes restarts the pod, which picks up the new TLS configuration
4. This ensures all webhook connections use the cluster's security policy uniformly

This behavior mirrors how the main operator handles TLS profile changes.

### Garbage Collection

#### OpenShift with OLM

Resources are cleaned up automatically via Kubernetes owner references:

| Resource | Owner | Scope |
|----------|-------|-------|
| Service | Operator Deployment | Namespace |
| Deployment | Operator Deployment | Namespace |
| MutatingWebhookConfiguration | Operator CSV (ClusterServiceVersion) | Cluster |

**Note on OLM resource naming:** OLM generates dynamic names for ClusterRoles and ClusterRoleBindings (e.g., `opentelemetry-operator.v-<hash>`), so the code cannot rely on hardcoded names. For cluster-scoped owner references, the operator uses the CSV which has a predictable name pattern based on the operator version.

When the operator is uninstalled:
- OLM deletes the CSV and operator Deployment
- Kubernetes garbage collector cascades deletion to owned resources
- No manual cleanup required

#### Kubernetes with cert-manager

Since resources are user-managed (via Helm chart or manual manifests), cleanup is also user-managed.

### Configuration

Environment variables (set via `Subscription.spec.config.env` on OpenShift):

| Variable | Description | Default |
|----------|-------------|---------|
| `ENABLE_STANDALONE_WEBHOOK` | Enable standalone webhook deployment | `false` |
| `STANDALONE_WEBHOOK_REPLICAS` | Number of webhook replicas | `1` |
| `RELATED_IMAGE_OPERATOR` | Operator image for webhook deployment | (required) |

CLI flags (for `make deploy`):

| Flag | Description |
|------|-------------|
| `--enable-standalone-webhook` | Enable standalone webhook deployment |
| `--standalone-webhook-replicas` | Number of webhook replicas |
| `--operator-image` | Operator image for webhook deployment |

### Platform Detection

The operator only manages the standalone webhook on OpenShift. On other platforms, users deploy the webhook manually:

```go
func Reconcile(ctx context.Context, params Params) error {
    if !params.Config.EnableStandaloneWebhook {
        return nil
    }

    // Only OpenShift is supported for operator-managed standalone webhook
    if !isOpenShift(params.Config) {
        logger.Info("Standalone webhook is only operator-managed on OpenShift. On Kubernetes, deploy manually using kustomize.")
        return nil
    }
    // Create Service, Deployment, MutatingWebhookConfiguration for OpenShift...
}
```

### Webhook Handoff

When standalone webhook is enabled:

1. Operator creates the standalone webhook resources
2. Operator removes `mpod.kb.io` from its own MutatingWebhookConfiguration
3. All pod mutation requests route to the standalone webhook
4. CR webhooks (Instrumentation, OpenTelemetryCollector, etc.) remain in the operator

## Deployment Scenarios

### OpenShift with OLM (Recommended for HA)

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: opentelemetry-operator
  namespace: openshift-operators
spec:
  channel: stable
  name: opentelemetry-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
  config:
    env:
    - name: ENABLE_STANDALONE_WEBHOOK
      value: "true"
    - name: STANDALONE_WEBHOOK_REPLICAS
      value: "3"
```

### Kubernetes with cert-manager (Optional)

On Kubernetes, the standalone webhook is an optional feature for users who need HA. Deployment manifests may be provided via Helm chart in the [opentelemetry-helm-charts](https://github.com/open-telemetry/opentelemetry-helm-charts) repository.

This approach gives users full control over webhook scaling and lifecycle.

### Kubernetes without cert-manager (Not Supported)

The standalone webhook requires TLS certificates. Without OpenShift service-ca or cert-manager, users must:
1. Manually create TLS secrets
2. Manually configure webhook CA bundles
3. Handle certificate rotation

This is not recommended and not officially supported.

## Alternatives Considered

### 1. OLM Deployment Scaling

**Approach**: Use OLM's `Subscription.spec.config.replicas` to scale the operator.

**Why rejected**: OLM doesn't support `replicas` in subscription config. OLM actively reconciles deployments back to CSV-defined replicas.

### 2. HPA on Operator Deployment

**Approach**: Users add HPA targeting the operator deployment.

**Why rejected**: OLM fights with HPA over replica count. Also scales controllers unnecessarily.

### 3. Webhook in CSV webhookdefinitions

**Approach**: Define the webhook deployment in the CSV, let OLM manage it.

**Why rejected**: OLM would manage replicas, defeating the HA goal. Users couldn't scale independently.

### 4. Separate Helm Chart / Kustomize Overlay

**Approach**: Provide separate manifests for HA webhook deployment.

**Why rejected**: Poor user experience, requires manual installation and lifecycle management.

### 5. Dynamic Certificate Generation (like cert-manager webhook)

**Approach**: Webhook generates its own certificates at runtime.

**Why rejected**: Requires significant code changes. Controller-runtime expects pre-provisioned certs.

## Implementation

### Files Modified

- `internal/webhookdeployment/webhookdeployment.go` - Core reconciliation logic (OpenShift only)
- `internal/config/config.go` - Configuration fields
- `internal/config/env.go` - Environment variable parsing
- `internal/config/cli.go` - CLI flag parsing
- `main.go` - Conditional webhook registration, TLS profile watcher for standalone webhook
- `config/rbac/role.yaml` - RBAC for webhook management

### RBAC Requirements

```yaml
# For MutatingWebhookConfiguration management (OpenShift only)
- apiGroups: [admissionregistration.k8s.io]
  resources: [mutatingwebhookconfigurations]
  verbs: [get, list, watch, create, update, patch, delete]

# For CSV lookup (used as owner reference for cluster-scoped resources)
- apiGroups: [operators.coreos.com]
  resources: [clusterserviceversions]
  verbs: [get, list, watch]
```

Note: No cert-manager RBAC is required since the operator does not create cert-manager resources.

## Testing

### Unit Tests

- Webhook deployment resource generation
- Platform detection logic
- Owner reference configuration

### E2E Tests

- OpenShift: Verify webhook deployment with service-ca certs
- Kubernetes: Verify webhook deployment with cert-manager
- Verify pod mutation works through standalone webhook
- Verify cleanup on operator uninstall
- Verify certificate rotation without pod restart

## References

- [GitHub Issue #5010](https://github.com/open-telemetry/opentelemetry-operator/issues/5010)
- [OpenShift Service CA Operator](https://github.com/openshift/service-ca-operator)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [Controller-runtime CertWatcher](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/certwatcher)
- [OLM Webhook Documentation](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/)
