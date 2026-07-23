# eBPF instrumentation with the OBI receiver

The [OBI receiver](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/obireceiver) uses eBPF to automatically instrument applications at the kernel level, producing distributed traces without code changes or language-specific agents. For detailed OBI configuration, security requirements, and capability breakdowns, see the [upstream OBI documentation](https://opentelemetry.io/docs/zero-code/obi/).

The collector image must include the OBI receiver. The [contrib distribution](https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib) includes it by default since `0.156.0`.

## Collector CR

OBI requires a DaemonSet with `hostPID: true` and elevated privileges to load eBPF programs. The simplest approach is a privileged container, which works across most environments:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: obi-collector
  namespace: obi-system
# ...
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: obi
rules:
  - apiGroups: ['apps']
    resources: ['replicasets']
    verbs: ['list', 'watch']
  - apiGroups: ['']
    resources: ['pods', 'services', 'nodes']
    verbs: ['list', 'watch']
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: obi
subjects:
  - kind: ServiceAccount
    name: obi-collector
    namespace: obi-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: obi
---
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: obi-collector
  namespace: obi-system
spec:
  mode: daemonset
  image: ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.156.0
  serviceAccount: obi-collector
  hostPID: true # Required to access the processes on the host
  securityContext:
    runAsUser: 0
    privileged: true
  config:
    receivers:
      obi:
        discovery:
          instrument:
            - k8s_namespace: my-application
              k8s_pod_annotations:
                obi.instrument: "true"
        attributes:
          kubernetes:
            enable: "true"
    exporters:
      debug:
        verbosity: detailed
    service:
      pipelines:
        traces:
          receivers: [obi]
          exporters: [debug]
```

## Selecting workloads

The `discovery.instrument` list controls which processes OBI attaches to. The example below filters by namespace and pod annotations — only pods matching all fields are instrumented:

```yaml
receivers:
  obi:
    discovery:
      instrument:
        - k8s_namespace: frontend
          k8s_pod_annotations:
            obi.instrument: "true"
        - k8s_namespace: backend
          k8s_pod_annotations:
            obi.instrument: "true"
```

Add the matching annotation to workload pod templates to opt them in:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: frontend
spec:
  template:
    metadata:
      annotations:
        obi.instrument: "true"
    spec:
      containers:
        - name: my-app
          image: my-app:latest
```

For further information, see the [OBI service discovery docs](https://opentelemetry.io/docs/zero-code/obi/configure/service-discovery/).

## Kubernetes attribute enrichment

Setting `attributes.kubernetes.enable: "true"` adds resource attributes like `k8s.namespace.name`, `k8s.pod.name`, and `service.name` to traces. The collector ServiceAccount needs `list` and `watch` on pods, nodes, services, replicationcontrollers, deployments, replicasets, statefulsets, and daemonsets — see the [OBI Kubernetes setup guide](https://opentelemetry.io/docs/zero-code/obi/setup/kubernetes/) for the full RBAC requirements.
