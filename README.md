[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc]

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

The operator manages:
* [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
* auto-instrumentation of the workloads using OpenTelemetry instrumentation libraries

## Documentation

* [API docs](./docs/api.md)

## Helm Charts

You can install Opentelemetry Operator via [Helm Chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-operator) from the opentelemetry-helm-charts repository. More information is available in [here](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-operator).

## Getting started

To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/) and run:
```bash
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Collector (otelcol) instance, like:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:
    processors:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 75
        spike_limit_percentage: 15
      batch:
        send_batch_size: 10000
        timeout: 10s

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: []
          exporters: [logging]
EOF
```

**_WARNING:_** Until the OpenTelemetry Collector format is stable, changes may be required in the above example to remain
compatible with the latest version of the OpenTelemetry Collector image being referenced.

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `logging`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) documentation for a reference of the possible entries.

At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Collector might crash.

The Operator does examine the configuration file to discover configured receivers and their ports. If it finds receivers with ports, it creates a pair of kubernetes services, one headless, exposing those ports within the cluster. The headless service contains a `service.beta.openshift.io/serving-cert-secret-name` annotation that will cause OpenShift to create a secret containing a certificate and key. This secret can be mounted as a volume and the certificate and key used in those receivers' TLS configurations.

### Upgrades

As noted above, the OpenTelemetry Collector format is continuing to evolve.  However, a best-effort attempt is made to upgrade all managed `OpenTelemetryCollector` resources.

In certain scenarios, it may be desirable to prevent the operator from upgrading certain `OpenTelemetryCollector` resources. For example, when a resource is configured with a custom `.Spec.Image`, end users may wish to manage configuration themselves as opposed to having the operator upgrade it.  This can be configured on a resource by resource basis with the exposed property `.Spec.UpgradeStrategy`.

By configuring a resource's `.Spec.UpgradeStrategy` to `none`, the operator will skip the given instance during the upgrade routine.

The default and only other acceptable value for `.Spec.UpgradeStrategy` is `automatic`.


### Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the collector should run as a `DaemonSet`, `Sidecar`, or `Deployment` (default). Look at [this sample](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/daemonset-features/00-install.yaml) for reference.

#### Sidecar injection

A sidecar with the OpenTelemetry Collector can be injected into pod-based workloads by setting the pod annotation `sidecar.opentelemetry.io/inject` to either `"true"`, or to the name of a concrete `OpenTelemetryCollector` from the same namespace, like in the following example:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: sidecar-for-my-app
spec:
  mode: sidecar
  config: |
    receivers:
      jaeger:
        protocols:
          thrift_compact:
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

kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    sidecar.opentelemetry.io/inject: "true"
spec:
  containers:
  - name: myapp
    image: jaegertracing/vertx-create-span:operator-e2e-tests
    ports:
      - containerPort: 8080
        protocol: TCP
EOF
```

When there are multiple `OpenTelemetryCollector` resources with a mode set to `Sidecar` in the same namespace, a concrete name should be used. When there's only one `Sidecar` instance in the same namespace, this instance is used when the annotation is set to `"true"`.

The annotation value can come either from the namespace, or from the pod. The most specific annotation wins, in this order:

* the pod annotation is used when it's set to a concrete instance name or to `"false"`
* namespace annotation is used when the pod annotation is either absent or set to `"true"`, and the namespace is set to a concrete instance or to `"false"`

When using a pod-based workload, such as `Deployment` or `Statefulset`, make sure to add the annotation to the `PodTemplate` part. Like:

```yaml
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  labels:
    app: my-app
  annotations:
    sidecar.opentelemetry.io/inject: "true" # WRONG
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
        sidecar.opentelemetry.io/inject: "true" # CORRECT
    spec:
      containers:
      - name: myapp
        image: jaegertracing/vertx-create-span:operator-e2e-tests
        ports:
          - containerPort: 8080
            protocol: TCP
EOF
```

When using sidecar mode the OpenTelemetry collector container will have the environment variable `OTEL_RESOURCE_ATTRIBUTES`set with Kubernetes resource attributes, ready to be consumed by the [resourcedetection](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/resourcedetectionprocessor) processor.

### OpenTelemetry auto-instrumentation injection

The operator can inject and configure OpenTelemetry auto-instrumentation libraries. Currently DotNet, Java, NodeJS and Python are supported.

To use auto-instrumentation, configure an `Instrumentation` resource with the configuration for the SDK and instrumentation.

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  exporter:
    endpoint: http://otel-collector:4317
  propagators:
    - tracecontext
    - baggage
    - b3
  sampler:
    type: parentbased_traceidratio
    argument: "0.25"
EOF
```

The above CR can be queried by `kubectl get otelinst`.

Then add an annotation to a pod to enable injection. The annotation can be added to a namespace, so that all pods within
that namespace wil get instrumentation, or by adding the annotation to individual PodSpec objects, available as part of
Deployment, Statefulset, and other resources.

Java:
```bash
instrumentation.opentelemetry.io/inject-java: "true"
```

NodeJS:
```bash
instrumentation.opentelemetry.io/inject-nodejs: "true"
```

Python:
```bash
instrumentation.opentelemetry.io/inject-python: "true"
```

DotNet:
```bash
instrumentation.opentelemetry.io/inject-dotnet: "true"
```

OpenTelemetry SDK environment variables only:
```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```

The possible values for the annotation can be
* `"true"` - inject and `Instrumentation` resource from the namespace.
* `"my-instrumentation"` - name of `Instrumentation` CR instance in the current namespace.
* `"my-other-namespace/my-instrumentation"` - name and namespace of `Instrumentation` CR instance in another namespace.
* `"false"` - do not inject

#### Multi-container pods

If nothing else is specified, instrumentation is performed on the first container available in the pod spec.
In some cases (for example in the case of the injection of an Istio sidecar) it becomes necessary to specify on which container(s) this injection must be performed.

For this, it is possible to fine-tune the pod(s) on which the injection will be carried out.

For this, we will use the `instrumentation.opentelemetry.io/container-names` annotation for which we will indicate one or more pod names (`.spec.containers.name`) on which the injection must be made:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment-with-multiple-containers
spec:
  selector:
    matchLabels:
      app: my-pod-with-multiple-containers
  replicas: 1
  template:
    metadata:
      labels:
        app: my-pod-with-multiple-containers
      annotations:
        instrumentation.opentelemetry.io/inject-java: "true"
        instrumentation.opentelemetry.io/container-names: "myapp,myapp2"
    spec:
      containers:
      - name: myapp
        image: myImage1
      - name: myapp2
        image: myImage2
      - name: myapp3
        image: myImage3
```

In the above case, `myapp` and `myapp2` containers will be instrumented, `myapp3` will not.

#### Use customized or vendor instrumentation

By default, the operator uses upstream auto-instrumentation libraries. Custom auto-instrumentation can be configured by
overriding the image fields in a CR.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  java:
    image: your-customized-auto-instrumentation-image:java
  nodejs:
    image: your-customized-auto-instrumentation-image:nodejs
  python:
    image: your-customized-auto-instrumentation-image:python
  dotnet:
    image: your-customized-auto-instrumentation-image:dotnet
```

The Dockerfiles for auto-instrumentation can be found in [autoinstrumentation directory](./autoinstrumentation).
Follow the instructions in the Dockerfiles on how to build a custom container image.

#### Inject OpenTelemetry SDK environment variables only

You can configure the OpenTelemetry SDK for applications which can't currently be autoinstrumented by using `inject-sdk` in place of (e.g.) `inject-python` or `inject-java`. This will inject environment variables like `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER`, and `OTEL_EXPORTER_OTLP_ENDPOINT`, that you can configure in the `Instrumentation`, but will not actually provide the SDK.

```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```

## Compatibility matrix

### OpenTelemetry Operator vs. OpenTelemetry Collector

The OpenTelemetry Operator follows the same versioning as the operand (OpenTelemetry Collector) up to the minor part of the version. For example, the OpenTelemetry Operator v0.18.1 tracks OpenTelemetry Collector 0.18.0. The patch part of the version indicates the patch level of the operator itself, not that of OpenTelemetry Collector. Whenever a new patch version is released for OpenTelemetry Collector, we'll release a new patch version of the operator.

By default, the OpenTelemetry Operator ensures consistent versioning between itself and the managed `OpenTelemetryCollector` resources.  That is, if the OpenTelemetry Operator is based on version `0.40.0`, it will create resources with an underlying OpenTelemetry Collector at version `0.40.0`.

When a custom `Spec.Image` is used with an `OpenTelemetryCollector` resource, the OpenTelemetry Operator will not manage this versioning and upgrading. In this scenario, it is best practice that the OpenTelemetry Operator version should match the underlying core version. Given a `OpenTelemetryCollector` resource with a `Spec.Image` configured to a custom image based on underlying OpenTelemetry Collector at version `0.40.0`, it is recommended that the OpenTelemetry Operator is kept at version `0.40.0`.


### OpenTelemetry Operator vs. Kubernetes vs. Cert Manager

We strive to be compatible with the widest range of Kubernetes versions as possible, but some changes to Kubernetes itself require us to break compatibility with older Kubernetes versions, be it because of code incompatibilities, or in the name of maintainability. Every released operator will support a specific range of Kubernetes versions, to be determined at the latest during the release.

We use `cert-manager` for some features of this operator and the third column shows the versions of the `cert-manager` that are known to work with this operator's versions.

The OpenTelemetry Operator *might* work on versions outside of the given range, but when opening new issues, please make sure to test your scenario on a supported version.

| OpenTelemetry Operator | Kubernetes           | Cert-Manager         |
|------------------------|----------------------|----------------------|
| v0.64.1                | v1.19 to v1.25       | v1                   |
| v0.63.1                | v1.19 to v1.25       | v1                   |
| v0.62.1                | v1.19 to v1.25       | v1                   |
| v0.61.0                | v1.19 to v1.25       | v1                   |
| v0.60.0                | v1.19 to v1.25       | v1                   |
| v0.59.0                | v1.19 to v1.24       | v1                   |
| v0.58.0                | v1.19 to v1.24       | v1                   |
| v0.57.2                | v1.19 to v1.24       | v1                   |
| v0.56.0                | v1.19 to v1.24       | v1                   |
| v0.55.0                | v1.19 to v1.24       | v1                   |
| v0.54.0                | v1.19 to v1.24       | v1                   |
| v0.53.0                | v1.19 to v1.24       | v1                   |
| v0.52.0                | v1.19 to v1.23       | v1                   |
| v0.51.0                | v1.19 to v1.23       | v1alpha2             |
| v0.50.0                | v1.19 to v1.23       | v1alpha2             |
| v0.49.0                | v1.19 to v1.23       | v1alpha2             |
| v0.48.0                | v1.19 to v1.23       | v1alpha2             |
| v0.47.0                | v1.19 to v1.23       | v1alpha2             |
| v0.46.0                | v1.19 to v1.23       | v1alpha2             |
| v0.45.0                | v1.21 to v1.23       | v1alpha2             |
| v0.44.0                | v1.21 to v1.23       | v1alpha2             |



## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

Approvers ([@open-telemetry/operator-approvers](https://github.com/orgs/open-telemetry/teams/operator-approvers)):

- [Benedikt Bongartz](https://github.com/frzifus), Red Hat
- [Yuri Oliveira Sa](https://github.com/yuriolisa), Red Hat

Emeritus Approvers:

- [Anthony Mirabella](https://github.com/Aneurysm9), AWS
- [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
- [Jay Camp](https://github.com/jrcamp), Splunk
- [James Bebbington](https://github.com/james-bebbington), Google
- [Owais Lone](https://github.com/owais), Splunk
- [Pablo Baeyens](https://github.com/mx-psi), DataDog

Maintainers ([@open-telemetry/operator-maintainers](https://github.com/orgs/open-telemetry/teams/operator-maintainers)):

- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Grafana Labs
- [Pavol Loffay](https://github.com/pavolloffay), Red Hat
- [Vineeth Pothulapati](https://github.com/VineethReddy02), Timescale

Emeritus Maintainers

- [Alex Boten](https://github.com/codeboten), Lightstep
- [Bogdan Drutu](https://github.com/BogdanDrutu), Splunk
- [Tigran Najaryan](https://github.com/tigrannajaryan), Splunk

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
[godoc]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#OpenTelemetryCollector
[contributors]: https://github.com/open-telemetry/opentelemetry-operator/graphs/contributors
[contributors-img]: https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-operator
