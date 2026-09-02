# OBI Capability & Context Propagation Findings

Findings from live testing on CRC (OpenShift, kernel 5.14.0, RHEL 9) with OBI v0.9.0.

## Test Environment

- **Platform:** CRC (CodeReady Containers), OpenShift
- **Kernel:** 5.14.0-570.103.1.el9_6.x86_64
- **OBI version:** v0.9.0
- **Sample app:** Go HTTP api-gateway → Go HTTP processor + Go gRPC enricher + Python gRPC enricher-py
- **Observability:** OTel Collector → Jaeger

## Capability Matrix (Validated)

All tests used `drop: ALL` with explicit capability adds. OBI config included `context_propagation: all` (disabled by default).

### Results

| Capabilities | HTTP context prop | Go gRPC context prop | Python gRPC context prop |
|-------------|:-:|:-:|:-:|
| Base 6 only | Yes | No | No |
| Base 6 + NET_ADMIN | Yes | No | No |
| Base 6 + NET_ADMIN + SYS_ADMIN | Yes | Yes | Yes |

**Base 6 capabilities** (application mode):
- `BPF` — load/attach eBPF programs
- `PERFMON` — perf events, BPF program loading
- `SYS_PTRACE` — /proc/pid/exe access across container namespaces
- `DAC_READ_SEARCH` — read ELF files across UID boundaries
- `CHECKPOINT_RESTORE` — /proc symlinks for process info
- `NET_RAW` — AF_PACKET raw sockets for socket filter programs

### Key Findings

1. **HTTP/1.1 context propagation works with base 6 caps alone.** OBI intercepts the TCP stream via kprobes/uprobes and injects traceparent headers without needing NET_ADMIN or SYS_ADMIN.

2. **gRPC/HTTP2 context propagation requires SYS_ADMIN on OpenShift.** This applies to ALL runtimes (Go, Python, Java, etc.), not just Go. Without SYS_ADMIN, gRPC spans appear as disconnected single-span traces with separate trace IDs.

3. **NET_ADMIN alone does not enable context propagation on kernel 5.14.** OBI has two context propagation paths for gRPC:
   - **bpf_probe_write_user** (Go-specific): writes traceparent into Go runtime memory via uprobes. Requires `SYS_ADMIN`.
   - **sk_msg / sockhash** (runtime-agnostic): injects HPACK headers at the socket layer. Requires `NET_ADMIN` + kernel >= 6.4.
   
   On kernel 5.14, the sk_msg path is disabled due to a known locking bug in `iter/tcp + sockhash`. OBI logs: `"TCP socket iterator disabled: kernel versions < 6.4 have a locking bug in iter/tcp + sockhash that can cause an RCU stall and kernel panic."`

4. **OpenShift 4.x ships kernel 5.14 (RHEL 9).** Kernel >= 6.4 arrives with RHEL 10 / OpenShift 5.x. Until then, SYS_ADMIN is the only viable path for gRPC context propagation on OpenShift.

5. **context_propagation must be explicitly enabled.** OBI defaults to `context_propagation: disabled`. The operator must set `ebpf.context_propagation: all` (or `headers` / `tcp`) in the OBI config for any context propagation to work.

6. **OTel SDK bypasses all of this.** If a gRPC service has the OTel SDK with gRPC instrumentation, the SDK handles context propagation at the application layer. No SYS_ADMIN, no NET_ADMIN, no kernel version dependency. OBI detects SDK-instrumented services and correlates with their spans.

## Implications for ClusterOBIAgent CRD

### Mode-to-capability mapping

```go
var modeCaps = map[OBIAgentMode][]corev1.Capability{
    OBIAgentModeApplication: {BPF, PERFMON, NET_RAW, SYS_PTRACE, DAC_READ_SEARCH, CHECKPOINT_RESTORE},
    OBIAgentModeNetwork:     {BPF, PERFMON, NET_RAW, NET_ADMIN},
    OBIAgentModeFull:        {BPF, PERFMON, NET_RAW, SYS_PTRACE, DAC_READ_SEARCH, CHECKPOINT_RESTORE, NET_ADMIN},
}
```

- `SYS_ADMIN` stays opt-in via `spec.additionalCapabilities` — it's too powerful to include in any default mode.
- `NET_ADMIN` in network/full mode is for TC-based network flow monitoring, not context propagation (on current OpenShift kernels).

### Documentation requirements

The operator should clearly document:

- gRPC context propagation for uninstrumented services requires `additionalCapabilities: [SYS_ADMIN]`
- Services with the OTel SDK do not need SYS_ADMIN — the SDK handles context propagation in-process
- On OpenShift 4.x, NET_ADMIN does not enable gRPC context propagation (kernel too old for sk_msg path)
- `context_propagation` must be enabled in the OBI config (the operator should set this when SYS_ADMIN is granted)

### Status conditions

The operator could surface useful status information:

```yaml
status:
  conditions:
    - type: ContextPropagation
      status: "Degraded"
      reason: "MissingSysAdmin"
      message: "gRPC context propagation unavailable — SYS_ADMIN not in capabilities. HTTP context propagation is active. Add additionalCapabilities: [SYS_ADMIN] or use the OTel SDK for gRPC services."
```

## OBI Source References

- Capability checking: `opentelemetry-ebpf-instrumentation/pkg/obi/os.go:174-205` (`checkCapabilitiesForSetOptions`)
- Context propagation config: `opentelemetry-ebpf-instrumentation/pkg/config/ebpf_tracer.go:83` (`OTEL_EBPF_BPF_CONTEXT_PROPAGATION`)
- Kernel version check for sockhash: OBI logs `"TCP socket iterator disabled: kernel versions < 6.4"`
- Go context propagation check: OBI logs `"Go context propagation at library level disabled due to missing capability CAP_SYS_ADMIN"`

## Upstream Issues

- [#2159](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/issues/2159) — Selective eBPF program loading (resource optimization)
- [#2234](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/issues/2234) — DynamicSelector with per-signal controls (operator integration API)
- [#2251](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/issues/2251) — Config v2 (config format redesign, actively landing)
- [#1920](https://github.com/open-telemetry/opentelemetry-ebpf-instrumentation/issues/1920) — Pinned eBPF map permissions on capability changes (upgrade concern)
