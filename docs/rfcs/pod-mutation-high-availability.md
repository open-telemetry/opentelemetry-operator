# Pod Mutation High Availability

## Summary

This RFC proposes enabling high availability (HA) for the pod mutation webhook on OpenShift by including it as a separate OLM-managed deployment in the CSV. This allows independent scaling of the webhook without affecting the operator's leader election.

## Motivation

The OpenTelemetry Operator includes a pod mutation webhook (`/mutate-v1-pod`) that handles:
- Auto-instrumentation injection
- Sidecar collector injection

In production environments, this webhook is critical infrastructure - if it's unavailable or slow, pod creation across the cluster is impacted. Currently, the webhook runs as part of the operator deployment, which has limitations:

1. **Single replica by default**: The operator uses leader election, and while the webhook doesn't require it, they share the same deployment.

2. **OLM scaling limitations**: On OpenShift with OLM, users cannot scale CSV-managed deployments UP - OLM continuously reconciles replicas back to the CSV-defined count. However, OLM allows scaling DOWN (including to 0 for maintenance).

3. **Coupled lifecycle**: Scaling the webhook means scaling the entire operator, which is wasteful since controllers only need one active instance.

### Goals

- Enable HA for the pod mutation webhook on OpenShift (OLM)
- Support scaling down for maintenance/debugging
- Maintain backward compatibility for vanilla Kubernetes deployments
- Ensure proper cleanup when the operator is uninstalled
- Handle TLS certificate provisioning automatically

### Non-Goals

- Changing how CR validation/mutation webhooks work (they remain in the operator)
- Supporting webhook HA on vanilla Kubernetes (no change to community bundle)
- Horizontal Pod Autoscaler (HPA) integration

## Design

### Architecture

On OpenShift with OLM, the CSV includes two deployments:

```
┌─────────────────────────────────────────────────────────────────┐
│                    opentelemetry-operator-system                │
│                                                                 │
│  ┌──────────────────────────┐    ┌──────────────────────────┐  │
│  │ opentelemetry-operator-  │    │ opentelemetry-operator-  │  │
│  │ controller-manager       │    │ pod-webhook              │  │
│  │ (1 replica, leader       │    │ (2 replicas, no leader   │  │
│  │  election)               │    │  election)               │  │
│  │                          │    │                          │  │
│  │ - CR controllers         │    │ - /mutate-v1-pod         │  │
│  │ - CR webhooks            │    │   (instrumentation +     │  │
│  │                          │    │    sidecar injection)    │  │
│  └──────────────────────────┘    └──────────────────────────┘  │
│                                                                 │
│  Both deployments managed by OLM via CSV                        │
└─────────────────────────────────────────────────────────────────┘
```

On vanilla Kubernetes (community bundle), everything remains in a single deployment (no change).

### How It Works

#### OpenShift with OLM

1. **Kustomize overlay** adds a second deployment definition via `config/overlays/openshift/pod-webhook-deployment.yaml`. This becomes part of the CSV during `make bundle`.
2. **Bundle generation** patches the `mpod.kb.io` webhookdefinition to point to the pod-webhook deployment instead of the controller-manager.
3. **OLM manages both deployments** - creation, TLS certificates, and cleanup. The operator does NOT create the pod-webhook deployment at runtime — OLM does, based on the CSV.
4. **Replica controller** (`PodWebhookReconciler`) only adjusts replicas on the OLM-created deployment when `POD_WEBHOOK_REPLICAS` env var is set. It does not create or delete the deployment.
5. **Default: 2 replicas** for HA, with PodDisruptionBudget (`maxUnavailable: 1`) and pod anti-affinity across nodes.
6. **Users can scale DOWN** (including to 0) via `POD_WEBHOOK_REPLICAS` but not UP beyond CSV default.

#### Vanilla Kubernetes

No change - the pod webhook runs as part of the operator deployment, same as before.

### Certificate Management

#### OpenShift (OLM-managed)

OLM handles certificate provisioning for both deployments automatically:
- Creates TLS secrets for webhook servers
- Injects CA bundles into MutatingWebhookConfiguration
- Rotates certificates automatically

#### Vanilla Kubernetes

Uses cert-manager annotations on the MutatingWebhookConfiguration (existing behavior).

### Garbage Collection

#### OpenShift with OLM

All resources are managed by OLM via the CSV:
- When the operator is uninstalled, OLM deletes the CSV
- All CSV-managed resources (both deployments, services, webhooks) are automatically cleaned up
- No manual cleanup required

#### Vanilla Kubernetes

Standard Kubernetes garbage collection via owner references (existing behavior).

### Scaling Behavior

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

#### Scaling Down

Only scaling down is supported. To reduce the pod-webhook replicas, set the `POD_WEBHOOK_REPLICAS` environment variable in the Subscription (0 or 1). Scale-up beyond CSV default (2) is ignored to avoid race conditions with OLM.

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
    - name: POD_WEBHOOK_REPLICAS
      value: "1"
```

The operator includes a controller that watches the pod-webhook deployment and scales it down to the configured value. After an upgrade (when OLM resets replicas to the CSV default of 2), the operator automatically scales it back down to the configured value (0 or 1).

### Pod Disruption Budget

A `PodDisruptionBudget` with `maxUnavailable: 1` is included in the OpenShift overlay. This ensures that during voluntary disruptions (node drains, cluster upgrades), Kubernetes evicts only one pod at a time and waits for the replacement to become Ready before evicting the next.

`maxUnavailable: 1` is preferred over `minAvailable: 1` because it works correctly at any replica count. With `minAvailable: 1` and a single replica, node drains would be blocked since evicting the only pod would drop available below the minimum. With `maxUnavailable: 1`, a single-replica deployment can still be drained.

Note: PDB only gates the eviction API (voluntary disruptions). Direct scaling via the controller or `kubectl scale` bypasses PDB entirely.

### Pod Anti-Affinity

The pod-webhook deployment uses `preferredDuringSchedulingIgnoredDuringExecution` pod anti-affinity on `kubernetes.io/hostname`. This tells the scheduler to prefer placing the 2 webhook pods on different nodes, ensuring a single node failure doesn't take down both webhook replicas.

`preferred` (soft) anti-affinity is used instead of `required` (hard) so that scheduling is not blocked on single-node or resource-constrained clusters (e.g., development environments).

## Implementation

### Bundle Generation

1. **Kustomize overlay** (`config/overlays/openshift/pod-webhook-deployment.yaml`):
   - Defines the `opentelemetry-operator-pod-webhook` deployment
   - Sets 2 replicas by default

2. **Makefile bundle target**:
   - Uses kustomize to include the pod-webhook deployment
   - Patches the `mpod.kb.io` webhookdefinition to reference the pod-webhook deployment

### Pod Webhook Replica Controller

The `internal/controllers/podwebhook_controller.go` watches the pod-webhook deployment and scales it down to the configured replica count (from `POD_WEBHOOK_REPLICAS` env var). Only scale-down is supported (0 or 1). Scale-up beyond CSV default is ignored.

### Files Modified

- `config/overlays/openshift/pod-webhook-deployment.yaml` - Pod webhook deployment for OpenShift (includes pod anti-affinity)
- `config/overlays/openshift/pod-webhook-pdb.yaml` - PodDisruptionBudget for the pod webhook
- `config/overlays/openshift/kustomization.yaml` - Includes pod-webhook deployment and PDB
- `Makefile` - Patches webhookdefinition after bundle generation
- `internal/controllers/podwebhook_controller.go` - Replica controller for upgrade survival
- `internal/config/env.go` - Reads `POD_WEBHOOK_REPLICAS` env var

### RBAC

The operator already has RBAC permissions for Deployments (used by other controllers), so no additional RBAC is needed for this feature. The pod-webhook replica controller reuses existing permissions.

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
```

The standalone webhook deployment is automatically created with 2 replicas.

To scale down for maintenance:
```bash
kubectl scale deployment opentelemetry-operator-pod-webhook --replicas=0
```

### Vanilla Kubernetes

No change - install via Helm chart or kustomize as before. The pod webhook runs in the operator deployment.

## Alternatives Considered

### 1. Operator-managed webhook deployment

**Approach**: Operator creates/manages the webhook deployment at runtime based on env vars.

**Why rejected**: 
- Complex garbage collection (cross-scope owner references don't work reliably)
- Required additional RBAC for MutatingWebhookConfiguration
- More code to maintain

### 2. OLM Deployment Scaling

**Approach**: Use OLM's `Subscription.spec.config.replicas` to scale the operator.

**Why rejected**: OLM doesn't support `replicas` in subscription config. OLM actively reconciles deployments back to CSV-defined replicas (for scaling UP).

### 3. HPA on Operator Deployment

**Approach**: Users add HPA targeting the operator deployment.

**Why rejected**: OLM fights with HPA over replica count for scaling UP. Also scales controllers unnecessarily.

## References

- [GitHub Issue #5010](https://github.com/open-telemetry/opentelemetry-operator/issues/5010)
- [OLM Webhook Documentation](https://olm.operatorframework.io/docs/advanced-tasks/adding-admission-and-conversion-webhooks/)
- [OLM Deployment Management](https://olm.operatorframework.io/docs/tasks/creating-operator-manifests/)
