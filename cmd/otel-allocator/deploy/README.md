# Standalone Target Allocator Deployment

This directory contains [kustomize](https://kustomize.io/) manifests for deploying
the Target Allocator **without** the OpenTelemetry Operator. Use this when you want
to run the Target Allocator as a standalone component, managing your own collector
fleet.

## Quick Start

1. Create a `ConfigMap` named `target-allocator` with your configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: target-allocator
data:
  targetallocator.yaml: |
    allocation_strategy: consistent-hashing
    filter_strategy: relabel-config
    collector_selector:
      matchLabels:
        app: my-collectors
    config:
      scrape_configs:
        - job_name: my-app
          static_configs:
            - targets: ["my-app:8080"]
```

2. Create a kustomize overlay referencing this base:

```yaml
# my-overlay/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: my-namespace
namePrefix: my-

resources:
  - github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/deploy
  - configmap.yaml  # your ConfigMap from step 1

images:
  - name: target-allocator
    newName: ghcr.io/open-telemetry/opentelemetry-operator/target-allocator
    newTag: v0.120.0

patches:
  - target:
      kind: ClusterRoleBinding
      name: target-allocator
    patch: |
      - op: replace
        path: /subjects/0/namespace
        value: my-namespace
```

3. Apply:
```bash
kubectl apply -k my-overlay/
```

## What's Included

| Resource | Name | Description |
|----------|------|-------------|
| ServiceAccount | `target-allocator` | Identity for the TA pods |
| ClusterRole | `target-allocator` | Read access to pods, nodes, services, endpoints |
| ClusterRoleBinding | `target-allocator` | Binds the ClusterRole to the ServiceAccount |
| Deployment | `target-allocator` | Runs the TA (1 replica by default) |
| Service | `target-allocator` | Exposes port 80 → 8080 for collectors |

## What You Need to Provide

- A `ConfigMap` named `target-allocator` containing `targetallocator.yaml`
- Collectors configured to poll the TA's `/jobs` endpoint

## Customization

Use standard kustomize features to customize:

- **Image**: `kustomize edit set image target-allocator=<your-image>`
- **Namespace**: set `namespace:` in your overlay
- **Replicas**: add a replica patch for the Deployment
- **ClusterRoleBinding namespace**: patch the `subjects[0].namespace` field
