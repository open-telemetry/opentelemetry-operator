[![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc]

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

At this point, it has [OpenTelemetry Service](https://github.com/open-telemetry/opentelemetry-service) as the only managed component.

## Getting started

To install the operator, run:
```
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/crds/opentelemetry_v1alpha1_opentelemetryservice_crd.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/service_account.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/role.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/role_binding.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Service (otelsvc) instance, like:

```
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryService
metadata:
  name: simplest
spec:
  config: |
    receivers:
      jaeger:

    processors:
      queued-retry:

    exporters:
      logging:

    pipelines:
      traces:
        receivers: [jaeger]
        processors: [queued-retry]
        exporters: [logging]
```

This will create an OpenTelemetry Service instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `jaeger-grpc` to a remote Jaeger collector.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Service instances. Refer to the [OpenTelemetry Service](https://github.com/open-telemetry/opentelemetry-service) documentation for a reference of the possible entries.

At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Service might crash.

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License
  
[Apache 2.0 License](./LICENSE).

[goreport-img]: https://goreportcard.com/badge/github.com/jpkrohling/opentelemetry-operator
[goreport]: https://goreportcard.com/report/github.com/jpkrohling/opentelemetry-operator
[godoc-img]: https://godoc.org/github.com/jpkrohling/opentelemetry-operator?status.svg
[godoc]: https://godoc.org/github.com/jpkrohling/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#OpenTelemetryService
