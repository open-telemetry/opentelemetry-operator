[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc]

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

The operator manages:
* [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
* [auto-instrumentation](https://opentelemetry.io/docs/concepts/instrumentation/automatic/) of the workloads using OpenTelemetry instrumentation libraries

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

> 🚨 **NOTE:** At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Collector might crash.

The Operator does examine the configuration file to discover configured receivers and their ports. If it finds receivers with ports, it creates a pair of kubernetes services, one headless, exposing those ports within the cluster. The headless service contains a `service.beta.openshift.io/serving-cert-secret-name` annotation that will cause OpenShift to create a secret containing a certificate and key. This secret can be mounted as a volume and the certificate and key used in those receivers' TLS configurations.

### Upgrades

As noted above, the OpenTelemetry Collector format is continuing to evolve.  However, a best-effort attempt is made to upgrade all managed `OpenTelemetryCollector` resources.

In certain scenarios, it may be desirable to prevent the operator from upgrading certain `OpenTelemetryCollector` resources. For example, when a resource is configured with a custom `.Spec.Image`, end users may wish to manage configuration themselves as opposed to having the operator upgrade it.  This can be configured on a resource by resource basis with the exposed property `.Spec.UpgradeStrategy`.

By configuring a resource's `.Spec.UpgradeStrategy` to `none`, the operator will skip the given instance during the upgrade routine.

The default and only other acceptable value for `.Spec.UpgradeStrategy` is `automatic`.


### Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the Collector should run as a [`DaemonSet`](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/), [`Sidecar`](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods), [`StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) or [`Deployment`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) (default). 

See below for examples of each deployment mode:
- [`Deployment`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/ingress/00-install.yaml)
- [`DaemonSet`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/daemonset-features/01-install.yaml)
- [`StatefulSet`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/smoke-statefulset/00-install.yaml)
- [`Sidecar`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/instrumentation-python/00-install-collector.yaml)

#### Sidecar injection

A sidecar with the OpenTelemetry Collector can be injected into pod-based workloads by setting the pod annotation `sidecar.opentelemetry.io/inject` to either `"true"`, or to the name of a concrete `OpenTelemetryCollector`, like in the following example:

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

The possible values for the annotation can be:

* "true" - inject `OpenTelemetryCollector` resource from the namespace.
* "sidecar-for-my-app" - name of `OpenTelemetryCollector` CR instance in the current namespace.
* "my-other-namespace/my-instrumentation" - name and namespace of `OpenTelemetryCollector` CR instance in another namespace.
* "false" - do not inject

When using a pod-based workload, such as `Deployment` or `StatefulSet`, make sure to add the annotation to the `PodTemplate` part. Like:

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

The operator can inject and configure OpenTelemetry auto-instrumentation libraries. Currently Apache HTTPD, DotNet, Go, Java, NodeJS and Python are supported.

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
  python:
    env:
      # Required if endpoint is set to 4317.
      # Python autoinstrumentation uses http/proto by default
      # so data must be sent to 4318 instead of 4317.
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: http://otel-collector:4318
  dotnet:
    env:
      # Required if endpoint is set to 4317.
      # Dotnet autoinstrumentation uses http/proto by default
      # See https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/blob/888e2cd216c77d12e56b54ee91dafbc4e7452a52/docs/config.md#otlp
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: http://otel-collector:4318
EOF
```

The values for `propagators` are added to the `OTEL_PROPAGATORS` environment variable.
Valid values for `propagators` are defined by the [OpenTelemetry Specification for OTEL_PROPAGATORS](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_propagators).

The value for `sampler.type` is added to the `OTEL_TRACES_SAMPLER` envrionment variable.
Valid values for `sampler.type` are defined by the [OpenTelemetry Specification for OTEL_TRACES_SAMPLER](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_traces_sampler).
The value for `sampler.argument` is added to the `OTEL_TRACES_SAMPLER_ARG` environment variable. Valid values for `sampler.argument` will depend on the chosen sampler. See the [OpenTelemetry Specification for OTEL_TRACES_SAMPLER_ARG](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_traces_sampler_arg) for more details.  

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

Go:

Go auto-instrumentation also honors an annotation that will be used to set the [OTEL_GO_AUTO_TARGET_EXE env var](https://github.com/open-telemetry/opentelemetry-go-instrumentation/blob/main/docs/how-it-works.md).
This env var can also be set via the Instrumentation resource, with the annotation taking precedence.
Since Go auto-instrumentation requires `OTEL_GO_AUTO_TARGET_EXE` to be set, you must supply a valid
executable path via the annotation or the Instrumentation resource. Failure to set this value causes instrumentation injection to abort, leaving the original pod unchanged.
```bash
instrumentation.opentelemetry.io/inject-go: "true"
instrumentation.opentelemetry.io/otel-go-auto-target-exe: "/path/to/container/executable"
```

Go auto-instrumentation also requires elevated permissions. The below permissions are set automatically and are required.

```yaml
securityContext:
    capabilities:
     add:
     - SYS_PTRACE
    privileged: true
    runAsUser: 0
```

Apache HTTPD:
```bash
instrumentation.opentelemetry.io/inject-apache-httpd: "true"
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

> 🚨 **NOTE**: Go auto-instrumentation **does not** support multicontainer pods. When injecting Go auto-instrumentation the first pod should be the only pod you want instrumented.

#### Use customized or vendor instrumentation

By default, the operator uses upstream auto-instrumentation libraries. Custom auto-instrumentation can be configured by
overriding the `image` fields in a CR.

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
  go:
    image: your-customized-auto-instrumentation-image:go
  apacheHttpd:
    image: your-customized-auto-instrumentation-image:apache-httpd
```

The Dockerfiles for auto-instrumentation can be found in [autoinstrumentation directory](./autoinstrumentation).
Follow the instructions in the Dockerfiles on how to build a custom container image.

#### Using Apache HTTPD autoinstrumentation

For `Apache HTTPD` autoinstrumentation, by default, instrumentation assumes httpd version 2.4 and httpd configuration directory `/usr/local/apache2/conf` as it is in the official `Apache HTTPD` image (f.e. docker.io/httpd:latest). If you need to use version 2.2, or your HTTPD configuration directory is different, and or you need to adjust agent attributes, customize the instrumentation specification per following example:
```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
  apache:
    image: your-customized-auto-instrumentation-image:apache-httpd
    version: 2.2
    configPath: /your-custom-config-path
    attrs:
    - name: ApacheModuleOtelMaxQueueSize
      value: "4096"
    - name: ...
      value: ...
```
List of all available attributes can be found at [otel-webserver-module](https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module)

#### Inject OpenTelemetry SDK environment variables only

You can configure the OpenTelemetry SDK for applications which can't currently be autoinstrumented by using `inject-sdk` in place of `inject-python` or `inject-java`, for example. This will inject environment variables like `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER`, and `OTEL_EXPORTER_OTLP_ENDPOINT`, that you can configure in the `Instrumentation`, but will not actually provide the SDK.

```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```

#### Controlling Instrumentation Capabilities

The operator allows specifying, via the feature gates,  which languages the Instrumentation resource may instrument.
These feature gates must be passed to the operator via the `--feature-gates` flag.
The flag allows for a comma-delimited list of feature gate identifiers.
Prefix a gate with '-' to disable support for the corresponding language.
Prefixing a gate with '+' or no prefix will enable support for the corresponding language.
If a language is enabled by default its gate only needs to be supplied when disabling the gate.

| Language      | Gate                                        | Default Value |
|---------------|---------------------------------------------|---------------|
| Java          | `operator.autoinstrumentation.java`         | enabled       |
| NodeJS        | `operator.autoinstrumentation.nodejs`       | enabled       |
| Python        | `operator.autoinstrumentation.python`       | enabled       |
| DotNet        | `operator.autoinstrumentation.dotnet`       | enabled       |
| ApacheHttpD   | `operator.autoinstrumentation.apache-httpd` | enabled       |
| Go            | `operator.autoinstrumentation.go`           | disabled      |

Language not specified in the table are always supported and cannot be disabled.

### Target Allocator

The OpenTelemetry Operator comes with an optional component, the [Target Allocator](/cmd/otel-allocator/README.md) (TA). When creating an OpenTelemetryCollector Custom Resource (CR) and setting the TA as enabled, the Operator will create a new deployment and service to serve specific `http_sd_config` directives for each Collector pod as part of that CR. It will also change the Prometheus receiver configuration in the CR, so that it uses the [http_sd_config](https://prometheus.io/docs/prometheus/latest/http_sd/) from the TA. The following example shows how to get started with the Target Allocator:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: collector-with-ta
spec:
  mode: statefulset
  targetAllocator:
    enabled: true
  config: |
    receivers:
      prometheus:
        config:
          scrape_configs:
          - job_name: 'otel-collector'
            scrape_interval: 10s
            static_configs:
            - targets: [ '0.0.0.0:8888' ]
            metric_relabel_configs:
            - action: labeldrop
              regex: (id|name)
              replacement: $$1
            - action: labelmap
              regex: label_(.+)
              replacement: $$1 

    exporters:
      logging:

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          processors: []
          exporters: [logging]
```
The usage of `$$` in the replacement keys in the example above is based on the information provided in the Prometheus receiver [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/prometheusreceiver/README.md) documentation, which states:
`Note: Since the collector configuration supports env variable substitution $ characters in your prometheus configuration are interpreted as environment variables. If you want to use $ characters in your prometheus configuration, you must escape them using $$.`

Behind the scenes, the OpenTelemetry Operator will convert the Collector’s configuration after the reconciliation into the following:

```yaml
    receivers:
      prometheus:
        config:
          scrape_configs:
          - job_name: otel-collector
            scrape_interval: 10s
            http_sd_configs:
            - url: http://collector-with-ta-targetallocator:80/jobs/otel-collector/targets?collector_id=$POD_NAME
            metric_relabel_configs:
            - action: labeldrop
              regex: (id|name)
              replacement: $$1
            - action: labelmap
              regex: label_(.+)
              replacement: $$1 

    exporters:
      logging:

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          processors: []
          exporters: [logging]
```

Note how the Operator removes any existing service discovery configurations (e.g., `static_configs`, `file_sd_configs`, etc.) from the `scrape_configs` section and adds an `http_sd_configs` configuration pointing to a Target Allocator instance it provisioned.

The OpenTelemetry Operator will also convert the Target Allocator's Prometheus configuration after the reconciliation into the following:

```yaml
    config:
      scrape_configs:
      - job_name: otel-collector
        scrape_interval: 10s
        static_configs:
        - targets: [ '0.0.0.0:8888' ]
        metric_relabel_configs:
        - action: labeldrop
          regex: (id|name)
          replacement: $1
        - action: labelmap
          regex: label_(.+)
          replacement: $1 
```
Note that in this case, the Operator replaces "$$" with a single "$" in the replacement keys. This is because the collector supports environment variable substitution, whereas the TA (Target Allocator) does not. Therefore, to ensure compatibility, the TA configuration should only contain a single "$" symbol.

More info on the TargetAllocator can be found [here](cmd/otel-allocator/README.md).

#### Target Allocator config rewriting

Prometheus receiver now has explicit support for acquiring scrape targets from the target allocator. As such, it is now possible to have the
Operator add the necessary target allocator configuration automatically. This feature currently requires the `operator.collector.rewritetargetallocator` feature flag to be enabled. With the flag enabled, the configuration from the previous section would be rendered as:

```yaml
    receivers:
      prometheus:
        config:
          global:
            scrape_interval: 1m
            scrape_timeout: 10s
            evaluation_interval: 1m
        target_allocator:
          endpoint: http://collector-with-ta-targetallocator:80
          interval: 30s
          collector_id: $POD_NAME

    exporters:
      logging:

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          processors: []
          exporters: [logging]
```

This also allows for a more straightforward collector configuration for target discovery using prometheus-operator CRDs. See below for a minimal example:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: collector-with-ta-prometheus-cr
spec:
  mode: statefulset
  targetAllocator:
    enabled: true
    serviceAccount: everything-prometheus-operator-needs
    prometheusCR:
      enabled: true
  config: |
    receivers:
      prometheus:
        config:

    exporters:
      logging:

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          processors: []
          exporters: [logging]
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

| OpenTelemetry Operator | Kubernetes           | Cert-Manager        |
|------------------------|----------------------|---------------------|
| v0.81.0                | v1.19 to v1.27       | v1                  |
| v0.80.0                | v1.19 to v1.27       | v1                  |
| v0.79.0                | v1.19 to v1.27       | v1                  |
| v0.78.0                | v1.19 to v1.27       | v1                  |
| v0.77.0                | v1.19 to v1.26       | v1                  |
| v0.76.1                | v1.19 to v1.26       | v1                  |
| v0.75.0                | v1.19 to v1.26       | v1                  |
| v0.74.0                | v1.19 to v1.26       | v1                  |
| v0.73.0                | v1.19 to v1.26       | v1                  |
| v0.72.0                | v1.19 to v1.26       | v1                  |
| v0.71.0                | v1.19 to v1.25       | v1                  |
| v0.70.0                | v1.19 to v1.25       | v1                  |
| v0.69.0                | v1.19 to v1.25       | v1                  |
| v0.68.0                | v1.19 to v1.25       | v1                  |
| v0.67.0                | v1.19 to v1.25       | v1                  |
| v0.66.0                | v1.19 to v1.25       | v1                  |
| v0.64.1                | v1.19 to v1.25       | v1                  |
| v0.63.1                | v1.19 to v1.25       | v1                  |
| v0.62.1                | v1.19 to v1.25       | v1                  |
| v0.61.0                | v1.19 to v1.25       | v1                  |
| v0.60.0                | v1.19 to v1.25       | v1                  |
| v0.59.0                | v1.19 to v1.24       | v1                  |
| v0.58.0                | v1.19 to v1.24       | v1                  |

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

In addition to the [core responsibilities](https://github.com/open-telemetry/community/blob/main/community-membership.md) the operator project requires approvers and maintainers to be responsible for releasing the project. See [RELEASE.md](./RELEASE.md) for more information and release schedule.

Approvers ([@open-telemetry/operator-approvers](https://github.com/orgs/open-telemetry/teams/operator-approvers)):

- [Benedikt Bongartz](https://github.com/frzifus), Red Hat
- [Tyler Helmuth](https://github.com/TylerHelmuth), Honeycomb
- [Yuri Oliveira Sa](https://github.com/yuriolisa), Red Hat

Emeritus Approvers:

- [Anthony Mirabella](https://github.com/Aneurysm9), AWS
- [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
- [Jay Camp](https://github.com/jrcamp), Splunk
- [James Bebbington](https://github.com/james-bebbington), Google
- [Owais Lone](https://github.com/owais), Splunk
- [Pablo Baeyens](https://github.com/mx-psi), DataDog

Target Allocator Maintainers ([@open-telemetry/operator-ta-maintainers](https://github.com/orgs/open-telemetry/teams/operator-ta-maintainers)):

- [Anthony Mirabella](https://github.com/Aneurysm9), AWS
- [Kristina Pathak](https://github.com/kristinapathak), Lightstep
- [Sebastian Poxhofer](https://github.com/secustor)

Maintainers ([@open-telemetry/operator-maintainers](https://github.com/orgs/open-telemetry/teams/operator-maintainers)):

- [Jacob Aronoff](https://github.com/jaronoff97), Lightstep
- [Pavol Loffay](https://github.com/pavolloffay), Red Hat
- [Vineeth Pothulapati](https://github.com/VineethReddy02), Timescale

Emeritus Maintainers

- [Alex Boten](https://github.com/codeboten), Lightstep
- [Bogdan Drutu](https://github.com/BogdanDrutu), Splunk
- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Grafana Labs
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
