# OBI eBPF Instrumentation Agent

**Status:** *Draft*

**Author:** Ozzy Walsh (@ozwalsh)

**Date:** 2026-06-15

## Objective

Integrate the [OpenTelemetry eBPF Instrumentation (OBI)](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation) agent into the operator as a first-class managed workload, enabling automatic zero-code instrumentation of applications via eBPF without requiring application changes or restarts.

## Summary

This RFC introduces a new cluster-scoped CRD, `ClusterOBIAgent`, that the operator reconciles into a privileged DaemonSet running the OBI eBPF agent. The operator manages the full lifecycle of the agent: namespace creation, RBAC, security context, and configuration.

OBI observes application traffic at the kernel level using eBPF probes, requiring elevated Linux capabilities or privileged container mode. The operator abstracts this complexity and handles platform differences between vanilla Kubernetes and OpenShift.

## Goals and non-goals

### Goals

- Introduce `ClusterOBIAgent` CRD for cluster-wide OBI agent lifecycle management
- Operator manages namespace, DaemonSet, ConfigMap, ServiceAccount, ClusterRole, ClusterRoleBinding (and SCC on OpenShift)
- Support both privileged and least-privilege (capability-based) security modes
- Handle PSA namespace labeling on all platforms; SCC on OpenShift
- Expose OBI agent configuration directly via the CR spec

### Non-goals

- Tenant opt-in controls (pod annotation selectors, per-namespace delegation) — deferred to a follow-up RFC
- Extension of the existing `Instrumentation` CR for eBPF — deferred to a follow-up RFC
- Multi-instance support — one `ClusterOBIAgent` per cluster (singleton, like `ClusterObservability`)

## Alternatives Considered
### OBI Sidecar Injection via existing auto-instrumentation injection
OBI supports [sidecar deployment](https://opentelemetry.io/docs/zero-code/obi/setup/kubernetes/#deploy-obi-as-a-sidecar-container).
Although this would fit the existing `Instrumentation` CR mutating web hook model; it is inefficient. Unlike the OpenTelemetry auto-instrumentation libraries, OBI supports instrumenting multiple processes. A sidecar per pod model would duplicate OBI memory usage (userspace agent, BPF maps etc) for each pod instrumented; a DaemonSet avoids this.

Also a DaemonSet model allows the privileged OBI pods to remain under the cluster administrators control, rather than having to deploy privileged sidecar containers in every namespace with pods requiring instrumentation.

### OBI Collector Receiver 
OBI also supports running as a [collector receiver](https://opentelemetry.io/docs/zero-code/obi/configure/collector-receiver/). 

```yaml
receivers:
  obi:
    open_port: '8080'
    discovery:
      instrument:
        - k8s_namespace: tenant-alpha
exporters:
  otlp:
    endpoint: <your-otlp-endpoint>
```

Unfortunately this is not yet available in an existing collector distribution, such as `otelcol-contrib`. Users would need to build their own collector image using `ocb`. Therefore this is unviable currently for initial OBI support in the operator. Eventually the operator could support both a dedicated `DaemonSet` (via the proposal in this RFC) and deployment via a collector `DaemonSet`. Since both options require configuring a `DaemonSet` with necessary privileges, volume mounts etc; the work in this RFC can be reused to later support the collector receiver model.


## Use cases

### Zero-code HTTP trace collection

An administrator installs the operator and wants automatic distributed tracing for all HTTP services in `tenant-alpha` without modifying any application:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: ClusterOBIAgent
metadata:
  name: cluster-obi-agent
spec:
  config:
    otel_traces_export:
      endpoint: http://otel-collector.observability:4318
    discovery:
      - k8s_namespace: tenant-alpha
        k8s_pod_labels:
          instrument: obo
```

### Least-privilege deployment

A security-conscious administrator on a production cluster wants to avoid `privileged: true`:

```yaml
spec:
  privileged: false
  config:
    otel_traces_export:
      endpoint: http://otel-collector.observability:4318
```

The operator automatically computes and injects the minimum capabilities required for the configured mode.

### Adding a capability the operator doesn't compute

A user on a kernel older than 5.11 needs `SYS_RESOURCE` (locked memory limit) which the operator cannot detect at reconcile time:

```yaml
spec:
  privileged: false
  capabilities:
    add: ["SYS_RESOURCE"]
```

## Struct Design

```go
type ClusterOBIAgentSpec struct {
    NodeSelector  map[string]string           `json:"nodeSelector,omitempty"`
    Tolerations   []corev1.Toleration         `json:"tolerations,omitempty"`
    Resources     corev1.ResourceRequirements `json:"resources,omitempty"`

    // +kubebuilder:default=true
    // Privileged controls whether the DaemonSet runs with privileged: true.
    // When false, the operator dynamically computes the minimum capabilities
    // for the configured mode.
    Privileged bool `json:"privileged,omitempty"`

    // Capabilities allows adding to or overriding the operator-computed
    // capability set. Only applies when privileged: false.
    // If Override is set it replaces the operator-computed capabilities
    // entirely. If only Add is set, those capabilities are appended.
    Capabilities *OBICapabilitySpec `json:"capabilities,omitempty"`

    // +kubebuilder:pruning:PreserveUnknownFields
    // Config represents the raw OBI agent YAML configuration.
    Config *apiextensionsv1.JSON `json:"config,omitempty"`
}

type OBICapabilitySpec struct {
    Override []corev1.Capability `json:"override,omitempty"`
    Add      []corev1.Capability `json:"add,omitempty"`
}
```

## Security Model

OBI requires privileged kernel access to attach eBPF probes and inspect process memory. The operator offers two modes.

### Default: `privileged: true`

Matches the [OBI Helm chart default](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-ebpf-instrumentation). Maximum compatibility with kind, Docker-in-Docker, minikube, and managed Kubernetes.

```yaml
securityContext:
  privileged: true
```

### `privileged: false` — operator-computed capabilities

The operator injects the minimum capabilities for the configured mode. The base set for application observability:

| Capability | Purpose |
|---|---|
| `BPF` | Loading eBPF programs |
| `PERFMON` | kprobes, pointer arithmetic in eBPF programs |
| `DAC_READ_SEARCH` | `/proc/self/mem` access for kernel version detection |
| `CHECKPOINT_RESTORE` | `/proc` symlink access for process/system info |
| `SYS_PTRACE` | `/proc/pid/exe` access for symbol scanning |
| `NET_RAW` | AF_PACKET socket for HTTP filter attachment |

Conditionally added based on `spec.config`:

| Capability | When added |
|---|---|
| `NET_ADMIN` | Context propagation enabled (`ebpf.context_propagation: headers\|tcp\|all`) or TC network mode (`network.source: tc`) |

Capabilities the operator cannot determine at reconcile time (node kernel version, runtime seccomp policy) are not injected automatically. Users who need `SYS_ADMIN` or `SYS_RESOURCE` should use `spec.capabilities.add` or `spec.privileged: true`. OBI gracefully degrades when `SYS_ADMIN` is absent (Go context propagation disabled, logged).

The unprivileged security context:

```yaml
securityContext:
  runAsUser: 0
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
    add: [BPF, PERFMON, DAC_READ_SEARCH, CHECKPOINT_RESTORE, SYS_PTRACE, NET_RAW]
```

### Compatibility note

`SYS_PTRACE` interacts with the `RuntimeDefault` seccomp profile applied by default on some distributions (GKE Autopilot, hardened EKS). A seccomp profile and a capability grant are orthogonal — both must permit the syscall. Users on such clusters may need `privileged: true` or a custom seccomp profile. This is the primary reason `privileged: true` remains the default.

## Namespace Management

The operator creates and owns a dedicated `obi-system` namespace for all OBI resources. This is not user-configurable. The operator labels it for Pod Security Admission on all platforms:

```yaml
pod-security.kubernetes.io/enforce: privileged
pod-security.kubernetes.io/audit: privileged
```

PSA is namespace-scoped — there is no per-pod grant in standard Kubernetes. The label is required even for `privileged: false` because PSA's `baseline` profile does not permit `BPF`, `SYS_PTRACE`, or `NET_RAW`.

On OpenShift, PSA and SCC admission run independently. Both must pass, so the namespace label and the SCC are both created.

## RBAC

### Operator permissions (changes to `config/rbac/role.yaml`)

The operator requires the following additions to its existing ClusterRole:

```yaml
# Elevate namespaces from read-only — operator owns the obi-system lifecycle
- apiGroups: [""]
  resources: ["namespaces"]
  verbs:
  - get
  - list
  - watch
+ - create
+ - update
+ - patch
+ - delete

# New — manage the DaemonSet's RBAC resources
+ - apiGroups: ["rbac.authorization.k8s.io"]
+   resources: ["clusterroles", "clusterrolebindings"]
+   verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# New — permissions the DaemonSet's ClusterRole grants; operator role must
# be a superset to satisfy Kubernetes RBAC escalation prevention (no escalate verb needed)
+ - apiGroups: [""]
+   resources: ["nodes", "replicationcontrollers"]
+   verbs: ["get", "list", "watch"]

# New — own CRD
+ - apiGroups: ["opentelemetry.io"]
+   resources: ["clusterobi agents", "clusterobi agents/status", "clusterobi agents/finalizers"]
+   verbs: ["get", "list", "watch", "update", "patch"]
```

The existing role already covers `configmaps`, `serviceaccounts`, `daemonsets`, `pods`, `services`, and `securitycontextconstraints` at the required verb level — no changes needed for those.

### DaemonSet permissions (reconciled by the operator)

The operator creates these cluster-scoped resources in `obi-system`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: obi
rules:
  - apiGroups: [""]
    resources: ["pods", "services", "nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["replicationcontrollers"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
```

### SecurityContextConstraints (OpenShift only)

On OpenShift the operator creates a purpose-built SCC bound to the `obi` service account. The `allowedCapabilities` list mirrors the operator-computed capability set for the configured mode:

```yaml
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: obi
allowHostPID: true
allowPrivilegedContainer: false   # true when spec.privileged: true
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
fsGroup:
  type: RunAsAny
volumes: ["configMap", "hostPath"]
allowedCapabilities:
  - BPF
  - PERFMON
  - SYS_PTRACE
  - DAC_READ_SEARCH
  - CHECKPOINT_RESTORE
  - NET_RAW
  # NET_ADMIN added when context propagation or TC mode enabled
requiredDropCapabilities: ["ALL"]
users:
  - system:serviceaccount:obi-system:obi
```

## Rollout Plan

- Introduce `ClusterOBIAgent` CRD as `v1alpha1`
- Feature gated behind an operator flag initially (`--enable-obi-agent`)
- Graduation to `v1beta1` pending field stability and broader testing
