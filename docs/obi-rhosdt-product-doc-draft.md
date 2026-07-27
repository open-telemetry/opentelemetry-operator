# Application Observability with eBPF Instrumentation (OBI)

**Product:** Red Hat build of OpenTelemetry (RHOSDT)
**Component:** OBI receiver for the OpenTelemetry Collector
**Initial release status:** Technology Preview

---

## Overview

Application Observability with eBPF Instrumentation (OBI) provides automatic, zero-code distributed tracing and RED metrics for applications running on OpenShift. OBI uses eBPF programs to observe network traffic and runtime behavior at the kernel level, producing OpenTelemetry-compatible traces and metrics without requiring application code changes or SDK integration.

OBI runs as a receiver within an OpenTelemetry Collector deployed as a DaemonSet. Each node's collector observes network traffic for annotated workloads and emits trace spans and derived metrics to configured exporters.

---

## Support level definitions

| Level | Definition |
|-------|------------|
| Generally Available (GA) | Fully supported, suitable for production use |
| Technology Preview (TP) | Not supported with Red Hat production SLAs. Provides early access for testing and feedback. See [Technology Preview scope of support](https://access.redhat.com/support/offerings/techpreview) |
| Not Available (NA) | Not available in this release |

---

## Feature support

### Tracing and metrics

| Feature | Status | Notes |
|---------|--------|-------|
| HTTP/1.x request tracing (spans) | TP | Passive network observation |
| HTTP/2 and gRPC request tracing (spans) | TP | Passive network observation |
| Database protocol tracing (SQL, Redis, Kafka) | TP | Passive network observation |
| RED metrics derived from traces | TP | Request rate, error rate, duration |
| Network flow metrics | TP | TCP/IP-level flow data |

### Runtime integration

| Feature | Status | Notes |
|---------|--------|-------|
| Go runtime integration (generic tracers) | TP | Read-only uprobes |
| Java runtime integration | TP | Read-only uprobes |
| Python runtime integration | TP | Read-only uprobes |
| Node.js runtime integration | TP | Read-only uprobes |
| Ruby runtime integration | TP | Read-only uprobes |
| Go library-level uprobes | NA | Known performance regression upstream |

### Context propagation and log enrichment

| Feature | Status | Notes |
|---------|--------|-------|
| Passive traceparent reading (SDK-instrumented apps) | TP | Reads existing headers, no injection |
| Active context propagation (traceparent injection) | NA | Requires `CAP_SYS_ADMIN` and `CAP_NET_ADMIN`, which are not granted |
| Log enrichment (trace-log correlation via injection) | NA | Requires `CAP_SYS_ADMIN`, which is not granted |

### Workload discovery

| Feature | Status | Notes |
|---------|--------|-------|
| Annotation-based workload discovery | TP | `obi.instrument: "true"` pod annotation |
| Namespace-scoped discovery rules | TP | v2 config: `rules[].match.kubernetes.namespace_glob` |
| Kubernetes attribute enrichment | TP | Namespace, pod, deployment, node attributes |

### Deployment and integration

| Feature | Status | Notes |
|---------|--------|-------|
| OBI receiver in OpenTelemetryCollector CR (DaemonSet mode) | TP | Standard collector CR with `obi` receiver |
| OpenShift SCC auto-provisioning | TP | Operator creates a dedicated SCC for the OBI collector |
| Unprivileged mode (capability-based) | TP | Requires 6 specific Linux capabilities; see [Security model](#security-model) |
| Privileged mode | NA | Operator rejects `privileged: true` when OBI receiver is present |

---

## Security model

OBI uses eBPF programs that require elevated Linux capabilities to load and attach. The operator enforces a specific capability set that enables full observability features while preventing OBI from writing into application memory or modifying network traffic.

### Enforced capability set

When an OpenTelemetryCollector CR includes an `obi` receiver, the operator enforces the following security context:

```yaml
securityContext:
  runAsUser: 0
  capabilities:
    add:
      - BPF              # Load and attach eBPF programs
      - DAC_READ_SEARCH  # Read /proc entries for process discovery
      - CHECKPOINT_RESTORE # BPF operations on newer kernels
      - PERFMON           # Perf events for eBPF program types
      - NET_RAW           # Raw socket access for packet inspection
      - SYS_PTRACE        # Read process memory for symbol resolution
    drop:
      - ALL
```

### What is blocked and why

The following capabilities are explicitly excluded:

| Capability | What it enables | Why it is blocked |
|------------|----------------|-------------------|
| `CAP_SYS_ADMIN` | `bpf_probe_write_user` — write into application memory buffers | Prevents application memory corruption, log buffer injection, and ioctl return value modification |
| `CAP_NET_ADMIN` | `sk_msg` / `sockops` — inject headers into socket streams | Prevents HTTP/gRPC header injection that can corrupt multiplexed connections |

This is enforced by the Linux kernel, not by OBI's application code. The kernel's BPF verifier rejects programs that call `bpf_probe_write_user` without `CAP_SYS_ADMIN` at program load time. The kernel refuses to load `sk_msg` and `sockops` program types without `CAP_NET_ADMIN`. Even if OBI attempts to load these programs, the kernel blocks them before any BPF bytecode executes.

OBI detects the capability restrictions at startup and gracefully degrades:

- Go tracer reloads without write-user programs
- Socket-level context propagation skips initialization
- Log enricher skips initialization entirely

### Distributed tracing without context propagation

Disabling context propagation does not eliminate distributed tracing:

| Scenario | Behavior |
|----------|----------|
| Application already propagates `traceparent` (e.g., OTel SDK) | Full distributed traces — OBI passively reads existing headers |
| No SDK, no propagation | Per-service traces and RED metrics; no cross-service trace correlation |

Applications that already use OpenTelemetry SDKs or other W3C Trace Context propagation get full distributed traces with zero risk. OBI reads the existing headers without modifying them.

### Data access scope

With the enforced capability set, OBI can **read** network traffic and process metadata across all workloads on the node, including request/response payloads. This is inherent to eBPF-based network observability. Sensitive data in transit (credentials, tokens, PII) is visible to OBI's eBPF programs.

OBI does not export raw payload content. It extracts protocol-level metadata (HTTP method, path, status code, gRPC service/method) and produces structured trace spans and metrics.

---

## Operator enforcement

### Webhook validation

The operator validates OpenTelemetryCollector CRs that include an `obi` receiver:

- Rejects CRs that set `privileged: true`
- Rejects CRs that add `CAP_SYS_ADMIN` or `CAP_NET_ADMIN`
- Rejects CRs that use a mode other than `daemonset`
- Enforces the capability set listed above via webhook mutation

This enforcement is controlled by the `operator.obi.enforceSecurityPolicy` feature gate, which is enabled by default in RHOSDT builds.

### OpenShift SCC

On OpenShift, the operator automatically creates a SecurityContextConstraint for the OBI collector's ServiceAccount:

- Allows the 6 required capabilities
- Allows `hostPID: true` (required for eBPF process discovery)
- Allows `hostPath` volumes for `/sys/fs/cgroup`
- Drops all other capabilities
- Denies privilege escalation
- Enforces `readOnlyRootFilesystem: true`

---

## Configuration

OBI is configured as a receiver within an OpenTelemetryCollector CR:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: obi-collector
spec:
  mode: daemonset
  config:
    receivers:
      obi:
        discovery:
          instrument:
            - k8s_namespace: my-namespace
              k8s_pod_annotations:
                obi.instrument: "true"
        attributes:
          kubernetes:
            enable: "true"
    exporters:
      otlp:
        endpoint: tempo-distributor:4317
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [obi]
          exporters: [otlp]
```

### Workload opt-in

Annotate pods to enable OBI instrumentation:

```yaml
metadata:
  annotations:
    obi.instrument: "true"
```

Only annotated pods in namespaces matching the discovery configuration are instrumented.

---

## Requirements

| Requirement | Value |
|-------------|-------|
| OpenShift Container Platform | 4.17+ |
| Kernel | RHEL 9 kernel (5.14+) with BPF backports |
| Node access | `hostPID: true`, hostPath `/sys/fs/cgroup` |
| Collector mode | DaemonSet (one per node) |

---

## Known limitations

- OBI is upstream pre-v1.0 software. Configuration surface and behavior may change between RHOSDT releases.
- eBPF kprobes for TCP operations fire system-wide on each node, not only for instrumented workloads. Overhead is proportional to total TCP request rate on the node.
- No published performance benchmarks exist upstream. Monitor node CPU overhead after deployment.
- Go library-level uprobes are excluded due to a known upstream performance regression.
- Without context propagation, cross-service trace correlation requires application-level `traceparent` propagation (e.g., via OTel SDK).
