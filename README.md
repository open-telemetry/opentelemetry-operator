[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc] [![Maintainability][code-climate-img]][code-climate] [![codecov][codecov-img]][codecov]
[![Repository on Quay](https://quay.io/repository/opentelemetry/opentelemetry-operator/status "Repository on Quay")](https://quay.io/repository/opentelemetry/opentelemetry-operator)

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

At this point, it has [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) as the only managed component.

## Documentation

* [SplunkOtelAgent Custom Resource Specification](./docs/otelcol_cr_spec.md)

## Getting started

To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/) and run:
```
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Collector (otelcol) instance, like:

```console
$ kubectl apply -f - <<EOF
apiVersion: splunk.com/v1alpha1
kind: SplunkOtelAgent
metadata:
  name: simplest
spec:
  config: |
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: []
          exporters: [logging]
EOF
```

**_WARNING:_** Until the OpenTelemetry Collector format is stable, changes may be required in the above example to remain
compatible with the latest version of the OpenTelemetry Collector image being referenced.

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `logging`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) documentation for a reference of the possible entries.

At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Collector might crash.

### Deployment modes

The `CustomResource` for the `SplunkOtelAgent` exposes a property named `.Spec.Mode`, which can be used to specify whether the collector should run as a `DaemonSet`, `Sidecar`, or `Deployment` (default). Look at the `examples/daemonset.yaml` for reference.

#### Sidecar injection

A sidecar with the OpenTelemetry Collector can be injected into pod-based workloads by setting the pod annotation `sidecar.splunk.com/inject` to either `"true"`, or to the name of a concrete `SplunkOtelAgent` from the same namespace, like in the following example:

```console
$ kubectl apply -f - <<EOF
apiVersion: splunk.com/v1alpha1
kind: SplunkOtelAgent
metadata:
  name: sidecar-for-my-app
spec:
  mode: sidecar
  config: |
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: []
          exporters: [logging]
EOF

$ kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    sidecar.splunk.com/inject: "true"
spec:
  containers:
  - name: myapp
    image: jaegertracing/vertx-create-span:operator-e2e-tests
    ports:
      - containerPort: 8080
        protocol: TCP
EOF
```

When there are multiple `SplunkOtelAgent` resources with a mode set to `Sidecar` in the same namespace, a concrete name should be used. When there's only one `Sidecar` instance in the same namespace, this instance is used when the annotation is set to `"true"`.

The annotation value can come either from the namespace, or from the pod. The most specific annotation wins, in this order:

* the pod annotation is used when it's set to a concrete instance name or to `"false"`
* namespace annotation is used when the pod annotation is either absent or set to `"true"`, and the namespace is set to a concrete instance or to `"false"`

When using a pod-based workload, such as `Deployment` or `Statefulset`, make sure to add the annotation to the `PodTemplate` part. Like:

```console
$ kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
  annotations:
    sidecar.splunk.com/inject: "true" # WRONG
spec:
  selector:
    matchLabels:
      app: my-app
  replicas: 1
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        sidecar.splunk.com/inject: "true" # CORRECT
    spec:
      containers:
      - name: myapp
        image: jaegertracing/vertx-create-span:operator-e2e-tests
        ports:
          - containerPort: 8080
            protocol: TCP
EOF
```

## Compatibility matrix

### OpenTelemetry Operator vs. OpenTelemetry Collector

The OpenTelemetry Operator follows the same versioning as the operand (OpenTelemetry Collector) up to the minor part of the version. For example, the OpenTelemetry Operator v0.18.1 tracks OpenTelemetry Collector 0.18.0. The patch part of the version indicates the patch level of the operator itself, not that of OpenTelemetry Collector. Whenever a new patch version is released for OpenTelemetry Collector, we'll release a new patch version of the operator.

### OpenTelemetry Operator vs. Kubernetes

We strive to be compatible with the widest range of Kubernetes versions as possible, but some changes to Kubernetes itself require us to break compatibility with older Kubernetes versions, be it because of code incompatibilities, or in the name of maintainability.

Our promise is that we'll follow what's common practice in the Kubernetes world and support N-2 versions, based on the release date of the OpenTelemetry Operator.

For instance, when we released v0.27.0, the latest Kubernetes version was v1.21.1. As such, the minimum version of Kubernetes we support for OpenTelemetry Operator v0.27.0 is v1.19 and we tested it with up to 1.21.

The OpenTelemetry Operator *might* work on versions outside of the given range, but when opening new issues, please make sure to test your scenario on a supported version.

| OpenTelemetry Operator | Kubernetes           |
|------------------------|----------------------|
| v0.33.0                | v1.20 to v1.22       |
| v0.32.0 (skipped)      | n/a                  |
| v0.31.0                | v1.19 to v1.21       |
| v0.30.0                | v1.19 to v1.21       |
| v0.29.0                | v1.19 to v1.21       |
| v0.28.0                | v1.19 to v1.21       |
| v0.27.0                | v1.19 to v1.21       |
| v0.26.0                | v1.18 to v1.20       |

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

Approvers ([@open-telemetry/operator-approvers](https://github.com/orgs/open-telemetry/teams/operator-approvers)):

- [@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)

Maintainers ([@open-telemetry/operator-maintainers](https://github.com/orgs/open-telemetry/teams/operator-maintainers)):

- [@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)
- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Red Hat

Learn more about roles in the [community repository](https://github.com/open-telemetry/community/blob/main/community-membership.md).

Thanks to all the people who already contributed!

[![Contributors][contributors-img]][contributors]

## License
  
[Apache 2.0 License](./LICENSE).

[github-workflow]: https://github.com/open-telemetry/opentelemetry-operator/actions
[github-workflow-img]: https://github.com/open-telemetry/opentelemetry-operator/workflows/Continuous%20Integration/badge.svg
[goreport-img]: https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-operator
[goreport]: https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-operator
[godoc-img]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator?status.svg
[godoc]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#SplunkOtelAgent
[code-climate]: https://codeclimate.com/github/open-telemetry/opentelemetry-operator/maintainability
[code-climate-img]: https://api.codeclimate.com/v1/badges/7bb215eea77fc9c24484/maintainability
[codecov]: https://codecov.io/gh/open-telemetry/opentelemetry-operator
[codecov-img]: https://codecov.io/gh/open-telemetry/opentelemetry-operator/branch/main/graph/badge.svg
[contributors]: https://github.com/open-telemetry/opentelemetry-operator/graphs/contributors
[contributors-img]: https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-operator
