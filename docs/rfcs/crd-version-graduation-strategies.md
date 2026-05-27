# CRD Version Graduation Strategies

This is a supporting document for the [Instrumentation v1beta1 RFC](instrumentation-v1beta1.md). It outlines how Kubernetes handles multiple CRD versions, strategies for operator maintainers, and lessons learned from other projects.

## Background

When graduating a CRD from `v1alpha1` to `v1beta1` (or `v1beta1` to `v1`), operators face a choice: how to handle the transition for existing users? Kubernetes supports serving multiple versions of the same CRD simultaneously, but this comes with complexity.

## Kubernetes CRD Versioning Basics

### Storage Version

Only one version can be the **storage version** - the version persisted in etcd. All other versions are converted to/from this version.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  versions:
    - name: v1alpha1
      served: true
      storage: false  # Not stored, converted from v1beta1
    - name: v1beta1
      served: true
      storage: true   # Stored in etcd
```

### Served Versions

The `served` field controls whether the API server accepts requests for that version.

**When `served: true`:**
- Clients can create, read, update, and delete resources using that version (e.g. `instrumentations.v1alpha1.opentelemetry.io`)
- Resources are auto-converted to/from the storage version

**When `served: false`:**
- API server returns 404 for that version's endpoint
- `kubectl get instrumentations.v1alpha1.opentelemetry.io` fails
- Existing resources in etcd are still accessible via served versions - with `strategy: None`, the API server just swaps the `apiVersion` field (requires identical schemas)
- New resources cannot be created using that version

### Conversion Strategies

| Strategy | When to Use |
|----------|-------------|
| `None` | Schemas are identical (only apiVersion differs) |
| `Webhook` | Schemas differ (field renames, restructuring, removals) |

If schemas differ and you use `None`, data won't map correctly between versions:
- **Renamed fields**: `foo` in `v1alpha1` won't appear in `bar` in `v1beta1`  — appears empty
- **Restructured fields**: `spec.exporter.endpoint` won't map to `spec.envConfig.exporter.endpoint`
- **Removed fields**: Data preserved in etcd but invisible in new schema

#### CRD conversions examples

**Example: No conversion (identical schemas)**

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  conversion:
    strategy: None
```

**Example: Webhook conversion**

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  conversion:
    strategy: Webhook
    webhook:
      conversionReviewVersions: ["v1"]  # ConversionReview API versions the webhook accepts (not CRD versions)
      clientConfig:
        service:
          namespace: opentelemetry-operator-system
          name: opentelemetry-operator-webhook
          path: /convert
```

## Strategy 1: Conversion Webhook

Implement a webhook that converts between versions automatically.

### Pros

- **Seamless migration**: Users can continue using old version, resources auto-convert
- **No forced migration**: Users upgrade at their own pace
- **Backwards compatible**: Old tools/scripts continue working

### Cons

- **Deployment complexity**: Webhook requires TLS certificates, secrets, and firewall rules on GKE private clusters (default firewall only allows ports 443/10250 from control plane to nodes — webhooks on other ports require custom firewall rules)
- **Maintenance burden**: Must maintain conversion logic
- **Helm complexity**: Webhooks need complex orchestration in Helm charts (OpenTelemetry operator solved this with templated CRDs — see [UPGRADING.md](https://github.com/open-telemetry/opentelemetry-helm-charts/blob/main/charts/opentelemetry-operator/UPGRADING.md))

### OpenTelemetry Collector `v1alpha1` → `v1beta1` Experience

The OpenTelemetry Operator implemented a conversion webhook for OpenTelemetryCollector v1alpha1 → v1beta1. Key issues encountered:

**Helm chart complications:**
- Webhook service name must be templated for custom Helm release names ([helm-charts#1167](https://github.com/open-telemetry/opentelemetry-helm-charts/issues/1167))
- Users get "service opentelemetry-operator-webhook not found" errors ([helm-charts#1199](https://github.com/open-telemetry/opentelemetry-helm-charts/issues/1199))

*Why this happens:* CRDs are cluster-scoped with hardcoded webhook service references, but Helm prefixes resource names with the release name (e.g., `helm install my-otel ...` creates `my-otel-opentelemetry-operator-webhook`). The CRD references `opentelemetry-operator-webhook`, but the actual service has a different name.

**OLM install mode restriction:**
- Only `AllNamespaces` install mode supported (operator watches all namespaces) — CRDs are cluster-scoped, so conversion webhooks must handle resources from all namespaces, incompatible with `OwnNamespace` mode. OLM v1 is moving away from install modes entirely, but the fundamental constraint remains: conversion webhooks are cluster-scoped.

## Strategy 2: Identical Schemas

Make all breaking changes while still in alpha, then graduate with identical schemas.

Prometheus Operator chose this approach for ScrapeConfig graduation after experiencing pain with conversion webhooks for AlertmanagerConfig. See [ScrapeConfig Graduation Proposal](https://prometheus-operator.dev/docs/proposals/accepted/scrapeconfig-graduation/#path-for-graduation).

When using `strategy: None`, no separate controllers are needed per version:

1. User creates resource using any served version (e.g., `v1alpha1`)
2. API server converts to storage version by changing `apiVersion` field
3. Resource is persisted in etcd as storage version (e.g., `v1beta1`)
4. Controller watches only the storage version using a single Go struct
5. When user reads with old version, API server converts back on the fly

The operator code remains unchanged — it reconciles only the storage version. The API server handles all version transformations transparently.

### Approach

1. Make all breaking changes in `v1alpha1` while it's still alpha (breaking changes are expected)
2. When schema is finalized, graduate to `v1beta1` with identical schema
3. Use conversion strategy `None` — only `apiVersion` changes
4. No conversion webhook needed

### Pros

- No conversion webhook complexity
- No maintenance burden for conversion logic
- Clear expectations — both versions behave identically
- Simple Helm/deployment — no webhook TLS/firewall concerns

### Cons

- **Breaking changes in alpha** — users on v1alpha1 must update their manifests
- **No automatic migration** — users must manually update `apiVersion`

## Cert-Manager `cmctl` Approach

Cert-manager used conversion webhooks for their core CRDs (`Certificate`, `Issuer`, `ClusterIssuer`, `CertificateRequest`) during the transition period while multiple versions were served. They had breaking changes between versions:

- **API group rename**: `certmanager.k8s.io` → `cert-manager.io`
- **Field removals**: `certificate.spec.acme`, `issuer.spec.http01`, `issuer.spec.dns01`
- **Field restructuring**: challenge solver configuration moved to new location

In addition to the runtime conversion webhook, they provide `cmctl convert` — an offline CLI tool for migrating stored manifests before upgrading.

**Version progression:** `v1alpha2` → `v1alpha3` → `v1beta1` → `v1`

| cert-manager | Storage | Served | Notes |
|--------------|---------|--------|-------|
| v1.0 - v1.3 | `v1` | `v1`, `v1beta1`, `v1alpha3`, `v1alpha2` | All versions served |
| v1.4 - v1.5 | `v1` | `v1`, `v1beta1`, `v1alpha3`, `v1alpha2` | Old versions deprecated |
| v1.6 | `v1` | `v1` only | Old versions no longer served |
| v1.7+ | `v1` | `v1` only | Old versions removed from CRD |

### How It Works

```bash
# Convert a single file
cmctl convert -f old-certificate.yaml > new-certificate.yaml

# Convert and apply directly
cmctl convert -f old-certificate.yaml | kubectl apply -f -

# Convert entire directory
cmctl convert -f ./manifests/ --output-dir ./converted/
```

The tool:
1. Parses input YAML with old API version
2. Maps old fields to new field names/locations
3. Applies defaults for new required fields
4. Outputs valid YAML for the new API version

### References

- [Migrating Deprecated API Resources](https://cert-manager.io/docs/releases/upgrading/remove-deprecated-apis/) — official migration guide
- [Upgrading from v0.16 to v1.0](https://cert-manager.io/docs/installation/upgrading/upgrading-0.16-1.0/) — major version upgrade guide
- [Issue #4686: Make cmctl upgrade old API versions](https://github.com/cert-manager/cert-manager/issues/4686) — discussion on migration tooling

## Storage Version Migration

When the storage version changes, existing resources in etcd remain in the old format until updated. The CRD's `status.storedVersions` tracks which versions still have objects in etcd:

```bash
kubectl get crd instrumentations.opentelemetry.io -o jsonpath='{.status.storedVersions}'
# Output: ["v1alpha1","v1beta1"]
```

You cannot remove a version from the CRD while it still appears in `storedVersions`.

### Migration Options

**Manual migration:**
```bash
# Empty patch forces read→convert→write cycle
kubectl get instrumentations -A -o name | xargs -I {} kubectl patch {} -p '{}'

# Or use get + apply
kubectl get instrumentations -A -o yaml | kubectl apply -f -
```

**Automated options:**

| Approach | Description |
|----------|-------------|
| [cmctl upgrade migrate-api-version](https://cert-manager.io/docs/reference/cmctl/) | Cert-manager CLI command; some Helm charts run this in CRD install jobs |
| [kube-storage-version-migrator](https://github.com/kubernetes-sigs/kube-storage-version-migrator) | Kubernetes SIG project; auto-detects storage version changes and migrates |
| [OpenShift migrator operator](https://github.com/openshift/cluster-kube-storage-version-migrator-operator) | Built into OpenShift; requires manual migration request creation |

See [Kubernetes docs: Upgrade existing objects to a new stored version](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version).

## References

- [Kubernetes CRD Versioning](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/)
- [Prometheus Operator ScrapeConfig Graduation](https://prometheus-operator.dev/docs/proposals/accepted/scrapeconfig-graduation/)
- [Prometheus Operator AlertmanagerConfig v1beta1 Issue](https://github.com/prometheus-operator/prometheus-operator/issues/4677)
- [Helm Charts v1beta1 Missing Issue](https://github.com/prometheus-community/helm-charts/issues/5168)
- [Kubernetes Bug: Conversion for Unserved Versions](https://github.com/kubernetes/kubernetes/issues/129979)
