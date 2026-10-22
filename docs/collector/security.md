# Security and RBAC

Security considerations when running collectors under the operator, and the admission and RBAC
controls available to constrain what a collector can do.

## Privileged pod settings in the collector spec

> [!WARNING]
> The `OpenTelemetryCollector` spec exposes host-level pod settings — `hostPath` volumes,
> `hostNetwork`, `hostPID`, privileged security contexts — and the operator passes them to the
> generated workload as written. Anyone who can create an `OpenTelemetryCollector` gets everything
> those settings allow, which in `mode: daemonset` means root on every node. Treat `create` on
> `opentelemetrycollectors.opentelemetry.io` as equivalent to `create` on `daemonsets.apps`.

### What the spec exposes

These fields are copied into the generated PodSpec verbatim. The operator applies no security
filtering to them, and the validating webhook does not inspect them.


| Field                                                    | What it grants                                                                                    |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `spec.volumes` with `hostPath`                           | Read/write access to any path on the node's filesystem                                            |
| `spec.hostNetwork`                                       | The node's network namespace, including loopback-only node services                               |
| `spec.hostPID`                                           | The node's PID namespace, and visibility into every process on it                                 |
| `spec.securityContext`                                   | `privileged`, added capabilities, `runAsUser: 0`                                                  |
| `spec.podSecurityContext`                                | Pod-level UID/GID, sysctls, `fsGroup`                                                             |
| `spec.initContainers`, `spec.additionalContainers`       | Arbitrary images and commands in the collector's pod, with their own security contexts and mounts |
| `spec.serviceAccount`                                    | The pod runs with an existing ServiceAccount's token, and so with its permissions                 |
| `spec.tolerations`, `spec.nodeSelector`, `spec.affinity` | Placement on control-plane or otherwise restricted nodes                                          |


Both served API versions, `v1alpha1` and `v1beta1`, expose them.

### Why they exist

Node-level telemetry requires node-level access:

- `filelog` reads container logs from `/var/log/pods`, which requires a `hostPath` volume.
- `hostmetrics` reads the host's `/proc` and `/sys`, and needs `hostPID` to attribute metrics to
processes outside its own namespace.
- eBPF components such as the [OBI eBPF receiver](../use-cases/obi-ebpf-receiver.md) need
`/sys/fs/cgroup` from the host plus capabilities like `BPF`, `PERFMON` and `SYS_PTRACE`.
- Receivers that bind node ports, such as statsd, need `hostNetwork`.

Rejecting these by default would break the operator's primary use case. Running a collector with them
set is normal.

### Restricting the fields

If unprivileged users need to create `OpenTelemetryCollector` resources, block the fields at admission.

#### Pod Security Admission

Labelling the namespace `baseline` or `restricted` blocks `hostPath`, `hostNetwork`, `hostPID` and
privileged containers on the resulting pods:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-a
  labels:
    pod-security.kubernetes.io/enforce: baseline
    pod-security.kubernetes.io/enforce-version: latest
```

The CR is still admitted and only the pods are rejected, so the user sees a healthy-looking resource
whose DaemonSet produces no pods, with the reason in the DaemonSet's events. PSA also does not
constrain `spec.serviceAccount`, `spec.tolerations` or `spec.nodeSelector`.

#### ValidatingAdmissionPolicy

Rejecting the CR gives a clear error at apply time. Built into Kubernetes, GA since 1.30:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: otel-collector-restrict-privileged
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
      - apiGroups: ["opentelemetry.io"]
        apiVersions: ["v1alpha1", "v1beta1"]
        operations: ["CREATE", "UPDATE"]
        resources: ["opentelemetrycollectors"]
  validations:
    - expression: "!has(object.spec.hostNetwork) || !object.spec.hostNetwork"
      message: "spec.hostNetwork is not allowed"
    - expression: "!has(object.spec.hostPID) || !object.spec.hostPID"
      message: "spec.hostPID is not allowed"
    - expression: "!has(object.spec.volumes) || object.spec.volumes.all(v, !has(v.hostPath))"
      message: "hostPath volumes are not allowed"
    - expression: >-
        !has(object.spec.securityContext) ||
        !has(object.spec.securityContext.privileged) ||
        !object.spec.securityContext.privileged
      message: "privileged security contexts are not allowed"
    - expression: >-
        !has(object.spec.additionalContainers) ||
        object.spec.additionalContainers.all(c,
          !has(c.securityContext) ||
          !has(c.securityContext.privileged) ||
          !c.securityContext.privileged)
      message: "privileged additionalContainers are not allowed"
    - expression: >-
        !has(object.spec.initContainers) ||
        object.spec.initContainers.all(c,
          !has(c.securityContext) ||
          !has(c.securityContext.privileged) ||
          !c.securityContext.privileged)
      message: "privileged initContainers are not allowed"
    - expression: >-
        !has(object.spec.targetAllocator) ||
        !has(object.spec.targetAllocator.securityContext) ||
        !has(object.spec.targetAllocator.securityContext.privileged) ||
        !object.spec.targetAllocator.securityContext.privileged
      message: "privileged security contexts are not allowed on the embedded target allocator"
```

Bind it to the namespaces holding untrusted collectors, leaving platform namespaces free to run
privileged node collectors:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: otel-collector-restrict-privileged
spec:
  policyName: otel-collector-restrict-privileged
  validationActions: ["Deny"]
  matchResources:
    namespaceSelector:
      matchLabels:
        otel-collector-privileged: "false"
```

#### Kyverno

The same expressions work in a `ClusterPolicy`, which adds an `Audit` mode for finding out what a
policy would break before enforcing it:

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: otel-collector-restrict-privileged
spec:
  validationFailureAction: Enforce
  background: false
  rules:
    - name: block-privileged-pod-fields
      match:
        any:
          - resources:
              kinds:
                - opentelemetry.io/v1alpha1/OpenTelemetryCollector
                - opentelemetry.io/v1beta1/OpenTelemetryCollector
              namespaces: ["tenant-*"]
      validate:
        cel:
          expressions:
            - expression: "!has(object.spec.hostNetwork) || !object.spec.hostNetwork"
              message: "spec.hostNetwork is not allowed"
            - expression: "!has(object.spec.hostPID) || !object.spec.hostPID"
              message: "spec.hostPID is not allowed"
            - expression: "!has(object.spec.volumes) || object.spec.volumes.all(v, !has(v.hostPath))"
              message: "hostPath volumes are not allowed"
```