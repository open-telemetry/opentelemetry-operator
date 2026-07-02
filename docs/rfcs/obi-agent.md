# OBI eBPF Instrumentation Agent

**Status:** *Draft*

**Author:** Ozzy Walsh (@ozwalsh)

**Date:** 2026-06-15

## Objective

Integrate the [OpenTelemetry eBPF Instrumentation (OBI)](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation) agent into the operator as a first-class managed workload, enabling automatic zero-code instrumentation of applications via eBPF without requiring application changes or restarts.

## Summary

This RFC enhances the Collector CR to automatically configure the pod spec for OBI when an `obi` receiver is present in the collector configuration. The operator detects the receiver and injects the required host-level settings (hostPID, security context, volumes) so that users only need to express their intent — what to discover and where to export — without manually configuring Linux capabilities, volume mounts, or platform-specific security machinery.

This is complementary to existing SDK auto-instrumentation. OBI instruments at the kernel level via eBPF; the existing `Instrumentation` CR injects language-specific SDKs via pod mutation. OBI detects SDK-instrumented applications and avoids duplicating signals ([OBI blog post](https://opentelemetry.io/blog/2025/obi-announcing-first-release/#should-i-just-use-opentelemetry-ebpf-instrumentation-for-everything)).

## Goals and non-goals

### Goals

- Facilitate deploying the OBI collector receiver via the Collector CR
- Extend the component parser framework to support receiver-driven volume and pod-spec injection
- Support both privileged and least-privilege (capability-based) security modes

### Non-goals

- New CRDs — OBI is deployed via the existing Collector CR
- Tenant opt-in controls (pod annotation selectors, per-namespace delegation) — deferred; OBI's built-in Kubernetes-aware discovery (namespace, pod labels, pod annotations) may be sufficient without operator-level tenant delegation
- Replacing the existing `Instrumentation` CR for SDK injection — OBI and SDKs are complementary

## Alternatives considered

### Dedicated `ClusterOBIAgent` CRD

A cluster-scoped CRD managing the OBI DaemonSet lifecycle, namespace, RBAC, and security context. 

The Collector CR approach is preferred because it reuses existing infrastructure (DaemonSet builder, config parsing, RBAC generation) and avoids introducing a new resource type.

### OBI sidecar via existing auto-instrumentation pod mutating webhook

OBI supports [sidecar deployment](https://opentelemetry.io/docs/zero-code/obi/setup/kubernetes/#deploy-obi-as-a-sidecar-container). Although this fits the existing `Instrumentation` CR model, it is inefficient. Unlike OTel auto-instrumentation libraries, OBI instruments multiple processes from a single agent. A sidecar-per-pod model duplicates the OBI memory footprint (userspace agent, BPF maps) for each pod; a DaemonSet avoids this.

A DaemonSet also keeps privileged OBI pods under cluster administrator control rather than deploying privileged sidecar containers in tenant namespaces.

### `Instrumentation` CR with `spec.obi` for tenant delegation

An `Instrumentation` CR with `spec.obi: {}` could allow namespace owners to opt their workloads into OBI instrumentation, with the operator aggregating these into OBI's discovery config. However, OBI already has built-in Kubernetes-aware discovery that filters by namespace, pod labels, and pod annotations natively. Pod labels may be sufficient for tenant opt-in without any CR involvement. This is deferred pending real-world feedback on whether OBI's native selectors are adequate.

## Custom collector image requirement

Previously it was only possible to use the obi receiver by building a custom collector image.

The obi receiver has (as of 2026-07-02) been [added](https://github.com/open-telemetry/opentelemetry-collector-releases/pull/1386) to the
`otelcol-contrib` collector distribution; once a new release is cut; this should be available for users to utilize with the otel operator.

## Use cases

### Zero-code HTTP trace collection

An administrator installs the operator and wants automatic distributed tracing for all HTTP services in `tenant-alpha` without modifying any application:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: obi-collector
  namespace: obi-system
spec:
  mode: daemonset
  image: $CUSTOM_COLLECTOR_IMAGE_WITH_OBI
  config:
    receivers:
      obi:
        discovery:
          instrument:
            - k8s_namespace: tenant-alpha
              k8s_pod_annotations:
                obi.instrument: "true"
            - k8s_namespace: tenant-beta
              k8s_pod_annotations:
                obi.instrument: "true"
        attributes:
          kubernetes:
            enable: "true"
    processors:
      routing:
        attribute_source: resource
        from_attribute: k8s.namespace.name
        table:
          - value: tenant-alpha
            exporters: [otlp/tenant-alpha]
          - value: tenant-beta
            exporters: [otlp/tenant-beta]
    exporters:
      otlp/tenant-alpha:
        endpoint: otelcol.tenant-alpha.svc.cluster.local:4317
        tls:
          insecure: true
      otlp/tenant-beta:
        endpoint: otelcol.tenant-beta.svc.cluster.local:4317
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [obi]
          processors: [routing]
          exporters: [otlp/tenant-alpha, otlp/tenant-beta]
```

The operator auto-configures hostPID, privileged security context, and required volumes. The user only specifies the discovery selectors and export destinations.

### Least-privilege deployment

A security-conscious administrator on a production cluster wants to avoid `privileged: true` and sets the required capabilities directly on the collector CR:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: obi-collector
  namespace: obi-system
spec:
  mode: daemonset
  image: $CUSTOM_COLLECTOR_IMAGE_WITH_OBI
  securityContext:
    privileged: false
    runAsUser: 0
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
      add: [BPF, PERFMON, DAC_READ_SEARCH, CHECKPOINT_RESTORE, SYS_PTRACE, NET_RAW]
  config:
    receivers:
      obi:
        ...
```

When the user provides `securityContext`, the operator does not override it — the user's explicit configuration takes precedence.

## Implementation design

### Extending the component parser framework

The operator's component parser framework (`internal/components/`) already supports per-receiver auto-configuration of RBAC rules, environment variables, ports, and probes. Volumes, volume mounts, and pod-level settings (hostPID, security context) are not currently supported — which is why receivers like hostmetrics, filelog, and journald require manual volume configuration.

This RFC extends the framework with new hooks, following the existing pattern:

| Existing hook | Returns | Consumed by |
|---|---|---|
| `GetRBACRules()` | `[]PolicyRule` | `rbac.go` |
| `GetEnvironmentVariables()` | `[]EnvVar` | `container.go` |
| `Ports()` | `[]ServicePort` | `container.go` |

| New hook | Returns | Consumed by |
|---|---|---|
| `GetVolumes()` | `[]Volume` | `volume.go` |
| `GetVolumeMounts()` | `[]VolumeMount` | `container.go` |
| `GetPodSpec()` | `PodSpecPatch` | `daemonset.go` |

`GetVolumes()` and `GetVolumeMounts()` follow the same decomposed pattern as existing hooks. `GetPodSpec()` covers pod-level concerns (hostPID, security context) that don't fit the volume/mount granularity.

New builder methods:

```go
WithVolumeGen(volumeGen VolumeGenerator[ComponentConfigType])
WithVolumeMountGen(volumeMountGen VolumeMountGenerator[ComponentConfigType])
WithPodSpecGen(podSpecGen PodSpecGenerator[ComponentConfigType])
```

Orchestration functions are added to `internal/otelconfig/config.go` following the existing `getRbacRulesForComponentKinds()` pattern.

### Detection

The operator detects the OBI receiver by calling the existing `otelconfig.GetEnabledComponents()` function during manifest reconciliation. 

### Pod spec injection

When the `obi` receiver is present in the collector config, the OBI receiver parser returns defaults via the new hooks:

| Field | Default injected |
|---|---|
| `spec.hostPID` | `true` |
| `spec.securityContext.privileged` | `true` |
| `spec.securityContext.readOnlyRootFilesystem` | `true` |
| `spec.volumes` | `hostPath /sys/fs/cgroup` + `emptyDir /var/run/obi` |
| `spec.volumeMounts` | Corresponding mounts for the above volumes |

### Override behavior

Defaults are applied only when the user has not already set the field. If `spec.securityContext` is present on the CR, the operator leaves it untouched — this is how the least-privilege capability set in the use case above works without special handling.

The same rule applies to `spec.hostPID` and any volumes: user-provided values are never overwritten.

## RBAC

### Operator permissions (changes to `config/rbac/role.yaml`)

Kubernetes RBAC escalation prevention requires the operator's own ClusterRole to be a superset of any permissions it grants to collector ServiceAccounts. The operator already covers `pods`, `services`, and `replicasets` at `list`/`watch`. The only addition required is:

```yaml
+ - apiGroups: [""]
+   resources: ["nodes"]
+   verbs: ["list", "watch"]
```

### Collector CR permissions

The OBI receiver parser returns RBAC rules via the existing `GetRBACRules()` hook:

```yaml
  - apiGroups: [""]
    resources: ["pods", "services", "nodes"]
    verbs: ["list", "watch"]
  - apiGroups: ["apps"]
    resources: ["replicasets"]
    verbs: ["list", "watch"]
```

### SecurityContextConstraints (OpenShift only)

On OpenShift the collector pods will not start without an SCC that permits `hostPID` and the required security context. This follows the same pattern as existing privileged receivers (filelog, hostmetrics) — the cluster administrator creates the SCC and binds it to the collector ServiceAccount as a prerequisite. The operator does not reconcile the SCC in this phase; automation can be added later if there is sufficient demand.

## Rollout plan

### Phase 1: OBI receiver support

Add the OBI receiver parser with the full set of hooks (volumes, volume mounts, pod spec, RBAC, env vars). Gated behind a feature gate (`operator.obi-receiver`, Alpha, disabled by default).

### Phase 2: Feature gate graduation

Graduate the feature gate to Beta (enabled by default) once the reconciler is stable and has e2e coverage.
