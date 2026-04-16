# Standalone Target Allocator Deployment

This directory contains [kustomize](https://kustomize.io/) manifests for deploying
the Target Allocator **without** the OpenTelemetry Operator. Use this when you want
to run the Target Allocator as a standalone component, managing your own collector
fleet.

These manifests are aligned with the
[opentelemetry-target-allocator Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-target-allocator)
in terms of RBAC rules, labels, probe configuration, and resource structure.

## Quick Start

Create a kustomize overlay that sets the image and namespace:

```yaml
# my-overlay/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: monitoring

resources:
  - github.com/open-telemetry/opentelemetry-operator/config/target-allocator

images:
  - name: target-allocator
    newName: ghcr.io/open-telemetry/opentelemetry-operator/target-allocator
    newTag: v0.148.0   # set to desired version

patches:
  - target:
      kind: ClusterRoleBinding
      name: target-allocator
    patch: |
      - op: replace
        path: /subjects/0/namespace
        value: monitoring
```

Apply:
```bash
kubectl apply -k my-overlay/
```

## What's Included

| Resource | Name | Description |
|----------|------|-------------|
| ServiceAccount | `target-allocator` | Identity for the TA pods |
| ClusterRole | `target-allocator` | Prometheus Operator CRDs, core resource discovery |
| ClusterRoleBinding | `target-allocator` | Binds the ClusterRole to the ServiceAccount |
| ConfigMap | `target-allocator` | Default TA configuration (Prometheus CR discovery) |
| Deployment | `target-allocator` | Runs the TA (1 replica, image set via overlay) |
| Service | `target-allocator` | Exposes port 80 → 8080 for collectors |

## Default Configuration

The included ConfigMap enables Prometheus CR discovery by default (matching the
Helm chart defaults):

- `allocation_strategy: consistent-hashing`
- `filter_strategy: relabel-config`
- `prometheus_cr.enabled: true` — discovers ServiceMonitor and PodMonitor CRs
- Empty selectors — accepts all ServiceMonitors and PodMonitors

## Customization

Use standard kustomize features to customize:

- **Image**: `kustomize edit set image target-allocator=<your-image>`
- **Namespace**: set `namespace:` in your overlay
- **Replicas**: add a replica patch for the Deployment
- **Configuration**: patch the ConfigMap with your `targetallocator.yaml`
- **ClusterRoleBinding namespace**: patch the `subjects[0].namespace` field

## Helm Chart

For more advanced configuration (StatefulSet mode, extra volumes, custom
annotations, etc.) consider the
[opentelemetry-target-allocator Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-target-allocator):

```bash
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm install my-ta open-telemetry/opentelemetry-target-allocator
```
