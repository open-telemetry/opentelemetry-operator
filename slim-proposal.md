# ClusterOBIAgent CRD

## What

The `ClusterOBIAgent` is a cluster-scoped CRD that enables the opentelemetry-operator to manage and automate OpenTelemetry eBPF Instrumentation (OBI) agents.

## Example Usage

Admin creates a `ClusterOBIAgent`:

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
      - k8s_namespace: tenant-beta
        k8s_pod_labels:
          instrument: obo
          critical: "true"
```

### Namespace Management

The operator creates and owns a dedicated namespace (`obi-system`) for all OBI resources. This namespace is not user-configurable. The operator manages the full lifecycle of this namespace — creation, labeling, and deletion when the `ClusterOBIAgent` is removed.

On all platforms the namespace is labeled for Pod Security Admission:

```yaml
pod-security.kubernetes.io/enforce: privileged
pod-security.kubernetes.io/audit: privileged
```

This is required because PSA is namespace-scoped — there is no per-pod PSA grant in standard Kubernetes. Without these labels the API server's admission controller rejects the DaemonSet pod regardless of the container's `securityContext`. On OpenShift, the labels satisfy PSA while the SCC (created separately) handles OpenShift's own admission.

### Security Context & Capabilities

The DaemonSet defaults to `privileged: true` for maximum compatibility (kind, Docker-in-Docker, minikube, managed Kubernetes). This matches the [eBPF instrumentation Helm chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-ebpf-instrumentation) default.

When the user sets `spec.privileged: false`, the operator switches to a least-privilege security context with the minimum capabilities needed for the configured mode:

**Base capabilities (application observability — the default):**

| Capability | Purpose |
|---|---|
| `BPF` | Loading eBPF programs |
| `PERFMON` | kprobes, pointer arithmetic in eBPF programs |
| `DAC_READ_SEARCH` | `/proc/self/mem` access for kernel version detection |
| `CHECKPOINT_RESTORE` | `/proc` symlink access for process/system info |
| `SYS_PTRACE` | `/proc/pid/exe` access for symbol scanning |
| `NET_RAW` | AF_PACKET socket for HTTP filter attachment |

**Conditionally added by the operator based on config:**

| Capability | When added |
|---|---|
| `NET_ADMIN` | Context propagation enabled (`ebpf.context_propagation: headers\|tcp\|all`) or network observability in TC mode (`network.source: tc`) |

**Not dynamically managed by the operator:**

| Capability | Why |
|---|---|
| `SYS_ADMIN` | Needed for Go library-level context propagation (`bpf_probe_write_user`), but OBI checks this at runtime and gracefully degrades (disables Go context propagation, logs a message). Also needed on some distros where `perf_event_paranoid >= 3`. The operator can't detect either condition at reconcile time. Users who need it should use `spec.capabilities.add` or `privileged: true`. |
| `SYS_RESOURCE` | Only needed on kernels < 5.11 to increase locked memory. Kernel version is a node property unknown to the operator at reconcile time. |

The unprivileged container also sets `runAsUser: 0`, `readOnlyRootFilesystem: true`, and `drop: ["ALL"]`.

Users who need to add or override capabilities can use `spec.capabilities`.

### RBAC & Security

<details>
<summary>ServiceAccount, ClusterRole, ClusterRoleBinding</summary>

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: obi
  namespace: obi-system # operator-managed
---
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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: obi
subjects:
  - kind: ServiceAccount
    name: obi
    namespace: obi-system # operator-managed
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: obi
```

</details>

<details>
<summary>SecurityContextConstraints (OpenShift only)</summary>

On OpenShift the controller creates a purpose-built SCC instead of relying on `privileged`. The `allowedCapabilities` list includes the base 6 plus any conditionally required capabilities based on the CR's config (e.g. `NET_ADMIN` if context propagation is enabled).

```yaml
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: obi
allowHostPID: true
allowPrivilegedContainer: false
readOnlyRootFilesystem: true
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
fsGroup:
  type: RunAsAny
volumes:
  - configMap
  - hostPath
allowedCapabilities:
  - BPF
  - PERFMON
  - SYS_PTRACE
  - DAC_READ_SEARCH
  - CHECKPOINT_RESTORE
  - NET_RAW
  # Conditionally added based on config:
  # - NET_ADMIN         (context propagation or TC network mode)
  # - SYS_ADMIN         (Go library-level context propagation or perf_event_paranoid >= 3)
  # - SYS_RESOURCE      (kernel < 5.11)
requiredDropCapabilities:
  - ALL
users:
  - system:serviceaccount:obi-system:obi # operator-managed namespace
```

</details>

<details>
<summary>Rendered DaemonSet (default — privileged: true)</summary>

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: obi
  namespace: obi-system # operator-managed
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: obi
      app.kubernetes.io/managed-by: opentelemetry-operator
  template:
    metadata:
      labels:
        app.kubernetes.io/name: obi
        app.kubernetes.io/managed-by: opentelemetry-operator
    spec:
      serviceAccountName: obi
      hostPID: true
      containers:
        - name: obi
          image: docker.io/otel/ebpf-instrument:v0.9.0
          securityContext:
            privileged: true
          volumeMounts:
            - name: var-run-obi
              mountPath: /var/run/obi
            - name: cgroup
              mountPath: /sys/fs/cgroup
              readOnly: true
            - name: bpf-maps
              mountPath: /sys/fs/bpf
            - name: obi-config
              mountPath: /etc/obi
              readOnly: true
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: OTEL_EBPF_CONFIG_PATH
              value: /etc/obi/config.yaml
      volumes:
        - name: var-run-obi
          hostPath:
            path: /var/run/obi
            type: DirectoryOrCreate
        - name: cgroup
          hostPath:
            path: /sys/fs/cgroup
        - name: bpf-maps
          hostPath:
            path: /sys/fs/bpf
        - name: obi-config
          configMap:
            name: obi-config
```

</details>

<details>
<summary>Rendered DaemonSet (privileged: false, application mode)</summary>

When `spec.privileged: false`, the operator injects fine-grained capabilities instead:

```yaml
          securityContext:
            runAsUser: 0
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
              add:
                - BPF
                - PERFMON
                - SYS_PTRACE
                - DAC_READ_SEARCH
                - CHECKPOINT_RESTORE
                - NET_RAW
```

If context propagation is enabled in the config, `NET_ADMIN` is added automatically.

</details>

## API surface

```go
type ClusterOBIAgentSpec struct {
    NodeSelector  map[string]string           `json:"nodeSelector,omitempty"`
    Tolerations   []corev1.Toleration         `json:"tolerations,omitempty"`
    Resources     corev1.ResourceRequirements `json:"resources,omitempty"`

    // +kubebuilder:default=true
    // Privileged controls whether the DaemonSet runs with privileged: true.
    // When false, the operator dynamically computes the minimum capabilities for the configured mode.
    Privileged bool `json:"privileged,omitempty"`

    // Capabilities allows adding to or overriding the operator-computed capability set.
    // Only applies when privileged: false.
    // If Override is set it replaces the operator-computed capabilities entirely — the user
    // owns the full add/drop policy. If only Add is set, those capabilities are appended to
    // the operator-computed set.
    Capabilities *OBICapabilitySpec `json:"capabilities,omitempty"`

    // +kubebuilder:pruning:PreserveUnknownFields
    // Config represents the raw OBI agent YAML configuration.
    Config           *apiextensionsv1.JSON `json:"config,omitempty"`
    TenantDelegation TenantDelegationSpec  `json:"tenantDelegation,omitempty"`
}

type OBICapabilitySpec struct {
    // Override replaces the operator-computed capability set entirely.
    Override []corev1.Capability `json:"override,omitempty"`
    // Add appends capabilities on top of the operator-computed set.
    Add      []corev1.Capability `json:"add,omitempty"`
}

type TenantDelegationSpec struct {
    NamespacesAllowList []string `json:"namespacesAllowList,omitempty"`
}

type OBIInstrumentationSpec struct {
    PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
}
```

The controller reconciles each `ClusterOBIAgent` into: Namespace, DaemonSet, ConfigMap, ServiceAccount, ClusterRole, ClusterRoleBinding (and SCC on OpenShift).
