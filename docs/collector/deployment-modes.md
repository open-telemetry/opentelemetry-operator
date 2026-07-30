# Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the Collector should run as a [`DaemonSet`](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/), [`Sidecar`](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods), [`StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) or [`Deployment`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) (default).

See below for examples of each deployment mode:

- [`Deployment`](../../tests/e2e/ingress/00-install.yaml)
- [`DaemonSet`](../../tests/e2e/smoke-collector/smoke-collector-daemonset/00-install.yaml)
- [`StatefulSet`](../../tests/e2e/smoke-collector/smoke-collector-statefulset/00-install.yaml)
- [`Sidecar`](../../tests/e2e-sidecar/smoke-sidecar/00-install.yaml)
