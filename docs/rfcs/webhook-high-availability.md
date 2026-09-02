# Webhook High Availability

## Summary

This RFC proposes enabling high availability (HA) for the operator's webhooks by running them in a separate deployment. This allows independent scaling of webhooks without affecting the operator's leader election.

This follows the same pattern used by other Kubernetes operators:
- [cert-manager](https://github.com/cert-manager/cert-manager/tree/master/cmd/webhook) — dedicated `cert-manager-webhook` deployment serves mutating, validating, and conversion webhooks
- [Kyverno](https://github.com/kyverno/kyverno/tree/main/cmd/kyverno) — separate `admission-controller` deployment handles all admission webhooks
- [Gatekeeper (OPA)](https://github.com/open-policy-agent/gatekeeper/tree/master/cmd/manager) — webhook deployment separate from audit controller
- [Knative Serving](https://github.com/knative/serving/tree/main/cmd/webhook) — dedicated `webhook` deployment for all validation and defaulting

## Motivation

The OpenTelemetry Operator includes several webhooks:
- **Pod mutation** (`/mutate-v1-pod`): auto-instrumentation and sidecar injection
- **CR defaulting/mutation**: defaulting webhooks for Instrumentation, OpenTelemetryCollector, OpAMPBridge, TargetAllocator
- **CR validation**: validating webhooks for all CR types (create/update/delete)
- **CR conversion**: conversion webhook for OpenTelemetryCollector

In production environments, these webhooks are critical infrastructure — if they're unavailable or slow, pod creation and CR operations across the cluster are impacted. Currently, all webhooks run as part of the operator deployment, which has limitations:

1. **Single replica by default**: The operator uses leader election, and while webhooks don't require it, they share the same deployment.

2. **Coupled lifecycle**: Scaling the webhooks means scaling the entire operator, which is wasteful since controllers only need one active instance.

### Goals

- Enable HA for all webhooks
- Support independent scaling of the webhook deployment
- Maintain backward compatibility (single-deployment mode continues to work)
- Ensure proper cleanup when the operator is uninstalled

### Non-Goals

- Horizontal Pod Autoscaler (HPA) integration

## Design

### Architecture

The operator binary supports a `webhook-server` subcommand that runs only the webhooks without the controllers. This enables deploying the webhooks as a separate deployment:

```
┌─────────────────────────────────────────────────────────────────┐
│                    opentelemetry-operator-system                │
│                                                                 │
│  ┌──────────────────────────┐    ┌──────────────────────────┐  │
│  │ opentelemetry-operator-  │    │ opentelemetry-operator-  │  │
│  │ controller-manager       │    │ webhook                  │  │
│  │ (1 replica, leader       │    │ (2+ replicas, no leader  │  │
│  │  election)               │    │  election)               │  │
│  │                          │    │                          │  │
│  │ - CR controllers         │    │ - /mutate-v1-pod         │  │
│  │ - Webhooks (registered   │    │ - CR defaulting webhooks  │  │
│  │   but no traffic routed  │    │ - CR validating webhooks  │  │
│  │   to this deployment)    │    │ - CR conversion webhook   │  │
│  └──────────────────────────┘    └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

The controller-manager still registers webhooks (`ENABLE_WEBHOOKS` defaults to `true`), but the kustomize overlay patches the webhook Service selector to route all traffic to the dedicated webhook deployment. Alternatively, `ENABLE_WEBHOOKS=false` can be set on the controller-manager to disable its webhook registration entirely.

### Webhook Subcommand

The operator binary accepts a `webhook-server` subcommand:

```bash
# Run the full operator (controllers + webhooks) — default, backward compatible
opentelemetry-operator

# Run only the webhooks
opentelemetry-operator webhook-server
```

When running in webhook-only mode, the binary:
- Registers all webhook handlers (pod mutation, CR defaulting, CR validation, CR conversion)
- Does NOT start any controllers or leader election
- Serves the webhook endpoints on the configured port

### Certificate Management

#### With cert-manager (vanilla Kubernetes)

Uses cert-manager annotations on webhook configurations (existing behavior). Both deployments share the same certificate.

#### With OLM (OpenShift)

OLM handles certificate provisioning for both deployments automatically:
- Creates TLS secrets for webhook servers
- Injects CA bundles into webhook configurations (Mutating, Validating, Conversion)
- Rotates certificates automatically

### Garbage Collection

Standard Kubernetes garbage collection via owner references. When deployed via OLM, all CSV-managed resources (both deployments, services, webhooks) are automatically cleaned up on uninstall.

### Pod Disruption Budget

A `PodDisruptionBudget` with `maxUnavailable: 1` ensures that during voluntary disruptions (node drains, cluster upgrades), Kubernetes evicts only one pod at a time and waits for the replacement to become Ready before evicting the next.

`maxUnavailable: 1` is preferred over `minAvailable: 1` because it works correctly at any replica count. With `minAvailable: 1` and a single replica, node drains would be blocked since evicting the only pod would drop available below the minimum. With `maxUnavailable: 1`, a single-replica deployment can still be drained.

Note: PDB only gates the eviction API (voluntary disruptions). Direct scaling via `kubectl scale` bypasses PDB entirely.

### Pod Anti-Affinity

The webhook deployment uses `preferredDuringSchedulingIgnoredDuringExecution` pod anti-affinity on `kubernetes.io/hostname`. This tells the scheduler to prefer placing webhook pods on different nodes, ensuring a single node failure doesn't take down all webhook replicas.

`preferred` (soft) anti-affinity is used instead of `required` (hard) so that scheduling is not blocked on single-node or resource-constrained clusters (e.g., development environments).

## Implementation

### Webhook Subcommand

- `main.go` — adds the `webhook-server` cobra subcommand that starts only the webhook server

### Webhook Deployment (Kustomize / Helm)

- `config/overlays/openshift/webhook-deployment.yaml` — webhook deployment definition (includes pod anti-affinity)
- `config/overlays/openshift/webhook-pdb.yaml` — PodDisruptionBudget for the webhook
- `config/overlays/openshift/webhook-mutating-patch.yaml` — patches MutatingWebhookConfiguration to use webhook service
- `config/overlays/openshift/webhook-validating-patch.yaml` — patches ValidatingWebhookConfiguration to use webhook service
- `config/overlays/openshift/webhook-service-patch.yaml` — changes webhook service selector to target webhook deployment pods
- `config/overlays/openshift/kustomization.yaml` — includes webhook deployment, PDB, and all webhook patches

### Files Modified

- `main.go` — `webhook-server` subcommand
- `internal/controllers/csv_webhook_controller.go` — CSV replica controller for upgrade survival (OpenShift)
- `internal/config/env.go` — reads `OPENSHIFT_WEBHOOK_REPLICAS` env var

The kustomize overlay structure (`config/manifests/openshift/kustomization.yaml` referencing `config/overlays/openshift/`) causes `operator-sdk generate bundle` to automatically set `deploymentName` to the webhook deployment in the OpenShift CSV. No post-generation Makefile patching is needed.

### RBAC

The CSV webhook replica controller requires RBAC for `operators.coreos.com/clusterserviceversions` with verbs `get`, `list`, `watch`, `update`, `patch`. This is a new RBAC requirement specific to the OpenShift OLM integration.

## OpenShift with OLM

On OpenShift, OLM manages operator deployments and has specific behavior around scaling that requires additional handling.

### OLM-Specific Architecture

The CSV includes both deployments. OLM handles:
- Deployment creation and lifecycle
- TLS certificate provisioning and rotation
- Webhook configuration CA bundle injection
- Cleanup on uninstall

The kustomize overlay structure (`config/overlays/openshift/`) causes `operator-sdk generate bundle` to automatically set all webhooks' `deploymentName` to the webhook deployment in the OpenShift CSV.

### OLM Scaling Behavior

Due to OLM's design:
- **Scale UP**: Usually reverted (OLM sees deployment as unhealthy during pod startup and reinstalls from CSV)
- **Scale DOWN**: Usually sticks (deployment stays healthy, so OLM doesn't trigger reinstall)
- **Upgrades reset replicas**: Manual scaling does NOT survive operator upgrades. OLM resets deployments to CSV-defined replicas during upgrade.

This means:
- Default 2 replicas provides HA out-of-the-box
- Users can scale to 1 or 0 for debugging/maintenance
- Users needing more than 2 replicas must wait for a CSV update

#### Why OLM Reverts Scale-Up But Not Scale-Down

OLM does not have explicit scale-enforcement logic. The asymmetry is a side effect of how
OLM's health check interacts with Kubernetes deployment rollout behavior.

**The reconciliation loop** ([deployment.go], [operator.go]):

1. OLM watches deployments owned by a CSV via an informer. Any change to the deployment
   requeues the owning CSV for reconciliation.
2. When the CSV is in the [`Succeeded` phase][succeeded-phase], OLM calls [`CheckInstalled()`][check-installed] which runs two checks:
   - **Spec-hash check**: compares a SHA256 hash stored in the [`olm.deployment-spec-hash`][hash-label]
     label on the deployment against a [hash computed from the CSV spec][hash-compute]. Since
     `kubectl scale` only changes `.spec.replicas` and does not modify labels, the hashes
     always match. This check never triggers a revert on its own.
   - **Availability check** ([`DeploymentStatus()`][deployment-status]): verifies that the
     deployment's `Available` condition is `True`.

**Scaling UP (e.g. 1 → 5) — reverted:**

When new pods are added, the Kubernetes deployment controller must schedule and start them.
During this window the deployment's `Available` condition becomes `False`
(`Deployment does not have minimum availability`). OLM sees this and triggers the cascade:

```
Succeeded → Failed (ComponentUnhealthy) → Pending (NeedsReinstall) → InstallReady → Install()
```

The [`Install()`][install] call executes [`CreateOrUpdateDeployment()`][create-or-update] which
performs a full `Update()` on the deployment object using the spec from the CSV — overwriting
`.spec.replicas` back to the CSV-defined value.

Note: this is timing-dependent. If all new pods become ready before OLM's next reconciliation
tick, OLM will see a healthy deployment with matching hashes and will not revert. In practice,
the informer-triggered reconciliation is fast enough that OLM almost always catches the
deployment in an unavailable state.

**Scaling DOWN (e.g. 2 → 0) — respected:**

When pods are removed, the Kubernetes deployment controller terminates excess pods while the
remaining pods (if any) continue running. The deployment's `Available` condition stays `True`
throughout the scale-down because the remaining replicas still satisfy the availability
requirement. Even scaling to 0 converges almost instantly from the API's perspective
(`status.replicas=0`, `updatedReplicas=0`, no outdated replicas).

Since `DeploymentStatus()` returns `ready=true` and the hash check passes, `CheckInstalled()`
reports the deployment as healthy. OLM keeps the CSV in `Succeeded` and does not trigger a
reinstall.

[deployment.go]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/deployment.go
[operator.go]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/operators/olm/operator.go
[succeeded-phase]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/operators/olm/operator.go#L2369
[check-installed]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/deployment.go#L218
[hash-label]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/deployment.go#L21
[hash-compute]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/deployment.go#L183-L187
[deployment-status]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/status_viewer.go#L13
[install]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/controller/install/deployment.go#L93
[create-or-update]: https://github.com/operator-framework/operator-lifecycle-manager/blob/v0.42.0/pkg/api/wrappers/deployment_install_client.go#L112-L125

**Summary:**

| Action | Deployment Available | OLM response |
|---|---|---|
| Scale up (pods starting) | `False` (temporarily) | Reinstall from CSV spec (reverts replicas) |
| Scale up (pods ready before OLM reconciles) | `True` | No action (replicas stick) |
| Scale down | `True` (always) | No action (replicas stick) |

### Webhook Replica Controller

The `internal/controllers/csv_webhook_controller.go` watches ClusterServiceVersion resources and modifies the webhook deployment replica count within the CSV spec. The desired replica count is read from the `OPENSHIFT_WEBHOOK_REPLICAS` env var (default: 2). The controller accepts any replica count, but in practice OLM typically reverts scale-up (see OLM Scaling Behavior above).

To reduce the webhook replicas, set the `OPENSHIFT_WEBHOOK_REPLICAS` environment variable in the Subscription:

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
    - name: OPENSHIFT_WEBHOOK_REPLICAS
      value: "1"
```

The operator includes a controller that watches ClusterServiceVersion resources and adjusts the webhook deployment replica count in the CSV spec. After an upgrade (when OLM resets replicas to the CSV default of 2), the operator automatically scales it back to the configured value.

## Alternatives Considered

### 1. Operator-managed webhook deployment

**Approach**: Operator creates/manages the webhook deployment at runtime based on env vars.

**Why rejected**: 
- Complex garbage collection (cross-scope owner references don't work reliably)
- Required additional RBAC for MutatingWebhookConfiguration
- More code to maintain

## References

- [GitHub Issue #5010](https://github.com/open-telemetry/opentelemetry-operator/issues/5010)
- [OLM Webhook Documentation](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/)
- [OLM Deployment Management](https://olm.operatorframework.io/docs/tasks/creating-operator-manifests/)
