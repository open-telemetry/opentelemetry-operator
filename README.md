[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc] [![Maintainability][code-climate-img]][code-climate] [![codecov][codecov-img]][codecov]
[![Repository on Quay](https://quay.io/repository/opentelemetry/opentelemetry-operator/status "Repository on Quay")](https://quay.io/repository/opentelemetry/opentelemetry-operator)

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

At this point, it has [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) as the only managed component.

## Getting started

To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/) and run:
```
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Collector (otelcol) instance, like:

```console
$ kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config: |
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:
      queued_retry:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: [queued_retry]
          exporters: [logging]
EOF
```
**_WARNING:_** Until the OpenTelemetry Collector format is stable, changes may be required in the above example to remain
compatible with the latest version of the OpenTelemetry Collector image being referenced.

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `logging`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) documentation for a reference of the possible entries.

At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Collector might crash.

### Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the collector should run as a `DaemonSet` or as a `Deployment` (default). Look at the `examples/daemonset.yaml` for reference.

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

Approvers ([@open-telemetry/operator-approvers](https://github.com/orgs/open-telemetry/teams/operator-approvers)):

- [@open-telemetry/collector-approvers](https://github.com/orgs/open-telemetry/teams/collector-approvers)

Maintainers ([@open-telemetry/operator-maintainers](https://github.com/orgs/open-telemetry/teams/operator-maintainers)):

- [@open-telemetry/collector-maintainers](https://github.com/orgs/open-telemetry/teams/collector-maintainers)
- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Red Hat

Learn more about roles in the [community repository](https://github.com/open-telemetry/community/blob/master/community-membership.md).

Thanks to all the people who already contributed!

[![Contributors][contributors-img]][contributors]

## License
  
[Apache 2.0 License](./LICENSE).

[github-workflow]: https://github.com/open-telemetry/opentelemetry-operator/actions
[github-workflow-img]: https://github.com/open-telemetry/opentelemetry-operator/workflows/Continuous%20Integration/badge.svg
[goreport-img]: https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-operator
[goreport]: https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-operator
[godoc-img]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator?status.svg
[godoc]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#OpenTelemetryCollector
[code-climate]: https://codeclimate.com/github/open-telemetry/opentelemetry-operator/maintainability
[code-climate-img]: https://api.codeclimate.com/v1/badges/7bb215eea77fc9c24484/maintainability
[codecov]: https://codecov.io/gh/open-telemetry/opentelemetry-operator
[codecov-img]: https://codecov.io/gh/open-telemetry/opentelemetry-operator/branch/master/graph/badge.svg
[contributors]: https://github.com/open-telemetry/opentelemetry-operator/graphs/contributors
[contributors-img]: https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-operator
