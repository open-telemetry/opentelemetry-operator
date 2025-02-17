[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc]

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

The operator manages:

- [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
- [auto-instrumentation](https://opentelemetry.io/docs/concepts/instrumentation/automatic/) of the workloads using OpenTelemetry instrumentation libraries

## Documentation

- [Compatibility & Support docs](./docs/compatibility.md)
- [API docs](./docs/api.md)
- [Offical Telemetry Operator page](https://opentelemetry.io/docs/kubernetes/operator/)

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
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 75
        spike_limit_percentage: 15
      batch:
        send_batch_size: 10000
        timeout: 10s

    exporters:
      debug: {}

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [debug]
EOF
```

**_WARNING:_** Until the OpenTelemetry Collector format is stable, changes may be required in the above example to remain
compatible with the latest version of the OpenTelemetry Collector image being referenced.

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `debug`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) documentation for a reference of the possible entries.

> 🚨 **NOTE:** At this point, the Operator does _not_ validate the whole contents of the configuration file: if the configuration is invalid, the instance might still be created but the underlying OpenTelemetry Collector might crash.

> 🚨 **Note:** For private GKE clusters, you will need to either add a firewall rule that allows master nodes access to port `9443/tcp` on worker nodes, or change the existing rule that allows access to port `80/tcp`, `443/tcp` and `10254/tcp` to also allow access to port `9443/tcp`. More information can be found in the [Official GCP Documentation](https://cloud.google.com/load-balancing/docs/tcp/setting-up-tcp#config-hc-firewall). See the [GKE documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules) on adding rules and the [Kubernetes issue](https://github.com/kubernetes/kubernetes/issues/79739) for more detail.

The Operator does examine the configuration file for a few purposes:

- To discover configured receivers and their ports. If it finds receivers with ports, it creates a pair of kubernetes services, one headless, exposing those ports within the cluster. If the port is using environment variable expansion or cannot be parsed, an error will be returned. The headless service contains a `service.beta.openshift.io/serving-cert-secret-name` annotation that will cause OpenShift to create a secret containing a certificate and key. This secret can be mounted as a volume and the certificate and key used in those receivers' TLS configurations.

- To check if Collector observability is enabled (controlled by `spec.observability.metrics.enableMetrics`). In this case, a Service and ServiceMonitor/PodMonitor are created for the Collector instance. As a consequence, if the metrics service address contains an invalid port or uses environment variable expansion for the port, an error will be returned. A workaround for the environment variable case is to set `enableMetrics` to `false` and manually create the previously mentioned objects with the correct port if you need them.
 
### Upgrades

As noted above, the OpenTelemetry Collector format is continuing to evolve. However, a best-effort attempt is made to upgrade all managed `OpenTelemetryCollector` resources.

In certain scenarios, it may be desirable to prevent the operator from upgrading certain `OpenTelemetryCollector` resources. For example, when a resource is configured with a custom `.Spec.Image`, end users may wish to manage configuration themselves as opposed to having the operator upgrade it. This can be configured on a resource by resource basis with the exposed property `.Spec.UpgradeStrategy`.

By configuring a resource's `.Spec.UpgradeStrategy` to `none`, the operator will skip the given instance during the upgrade routine.

The default and only other acceptable value for `.Spec.UpgradeStrategy` is `automatic`.

### Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the Collector should run as a [`DaemonSet`](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/), [`Sidecar`](https://kubernetes.io/docs/concepts/workloads/pods/#workload-resources-for-managing-pods), [`StatefulSet`](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/) or [`Deployment`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) (default).

See below for examples of each deployment mode:

- [`Deployment`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/ingress/00-install.yaml)
- [`DaemonSet`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/daemonset-features/01-install.yaml)
- [`StatefulSet`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/smoke-statefulset/00-install.yaml)
- [`Sidecar`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/tests/e2e/smoke-sidecar/00-install.yaml)

#### Sidecar injection

A sidecar with the OpenTelemetry Collector can be injected into pod-based workloads by setting the pod annotation `sidecar.opentelemetry.io/inject` to either `"true"`, or to the name of a concrete `OpenTelemetryCollector`, like in the following example:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: sidecar-for-my-app
spec:
  mode: sidecar
  config:
    receivers:
      jaeger:
        protocols:
          thrift_compact: {}
    processors:

    exporters:
      debug: {}

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          exporters: [debug]
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

- the pod annotation is used when it's set to a concrete instance name or to `"false"`
- namespace annotation is used when the pod annotation is either absent or set to `"true"`, and the namespace is set to a concrete instance or to `"false"`

The possible values for the annotation can be:

- "true" - inject `OpenTelemetryCollector` resource from the namespace.
- "sidecar-for-my-app" - name of `OpenTelemetryCollector` CR instance in the current namespace.
- "my-other-namespace/my-instrumentation" - name and namespace of `OpenTelemetryCollector` CR instance in another namespace.
- "false" - do not inject

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

### Using imagePullSecrets

The OpenTelemetry Collector defines a ServiceAccount field which could be set to run collector instances with a specific Service and their properties (e.g. imagePullSecrets). Therefore, if you have a constraint to run your collector with a private container registry, you should follow the procedure below:

- Create Service Account.

```bash
kubectl create serviceaccount <service-account-name>
```

- Create an imagePullSecret.

```bash
kubectl create secret docker-registry <secret-name> --docker-server=<registry name> \
        --docker-username=DUMMY_USERNAME --docker-password=DUMMY_DOCKER_PASSWORD \
        --docker-email=DUMMY_DOCKER_EMAIL
```

- Add image pull secret to service account

```bash
kubectl patch serviceaccount <service-account-name> -p '{"imagePullSecrets": [{"name": "<secret-name>"}]}'
```

### OpenTelemetry auto-instrumentation injection

The operator can inject and configure OpenTelemetry auto-instrumentation libraries. Currently Apache HTTPD, DotNet, Go, Java, Nginx, NodeJS and Python are supported.

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
  go:
    env:
      # Required if endpoint is set to 4317.
      # Go autoinstrumentation uses http/proto by default
      # so data must be sent to 4318 instead of 4317.
      - name: OTEL_EXPORTER_OTLP_ENDPOINT
        value: http://otel-collector:4318
EOF
```

The values for `propagators` are added to the `OTEL_PROPAGATORS` environment variable.
Valid values for `propagators` are defined by the [OpenTelemetry Specification for OTEL_PROPAGATORS](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_propagators).

The value for `sampler.type` is added to the `OTEL_TRACES_SAMPLER` environment variable.
Valid values for `sampler.type` are defined by the [OpenTelemetry Specification for OTEL_TRACES_SAMPLER](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_traces_sampler).
The value for `sampler.argument` is added to the `OTEL_TRACES_SAMPLER_ARG` environment variable. Valid values for `sampler.argument` will depend on the chosen sampler. See the [OpenTelemetry Specification for OTEL_TRACES_SAMPLER_ARG](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/#otel_traces_sampler_arg) for more details.

The instrumentation will automatically inject `OTEL_NODE_IP` and `OTEL_POD_IP` environment variables should you need to reference either value in an endpoint.

The above CR can be queried by `kubectl get otelinst`.

Then add an annotation to a pod to enable injection. The annotation can be added to a namespace, so that all pods within
that namespace will get instrumentation, or by adding the annotation to individual PodSpec objects, available as part of
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
Python auto-instrumentation also honors an annotation that will permit it to run it on images with a different C library than glibc.

```bash
instrumentation.opentelemetry.io/inject-python: "true"
instrumentation.opentelemetry.io/otel-python-platform: "glibc" # for Linux glibc based images, this is the default value and can be omitted
instrumentation.opentelemetry.io/otel-python-platform: "musl" # for Linux musl based images
```

.NET:
.NET auto-instrumentation also honors an annotation that will be used to set the .NET [Runtime Identifiers](https://learn.microsoft.com/en-us/dotnet/core/rid-catalog)(RIDs).
Currently, only two RIDs are supported: `linux-x64` and `linux-musl-x64`.
By default `linux-x64` is used.

```bash
instrumentation.opentelemetry.io/inject-dotnet: "true"
instrumentation.opentelemetry.io/otel-dotnet-auto-runtime: "linux-x64" # for Linux glibc based images, this is default value and can be omitted
instrumentation.opentelemetry.io/otel-dotnet-auto-runtime: "linux-musl-x64"  # for Linux musl based images
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
  privileged: true
  runAsUser: 0
```

Apache HTTPD:

```bash
instrumentation.opentelemetry.io/inject-apache-httpd: "true"
```

Nginx:

```bash
instrumentation.opentelemetry.io/inject-nginx: "true"
```

OpenTelemetry SDK environment variables only:

```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```

The possible values for the annotation can be

- `"true"` - inject and `Instrumentation` resource from the namespace.
- `"my-instrumentation"` - name of `Instrumentation` CR instance in the current namespace.
- `"my-other-namespace/my-instrumentation"` - name and namespace of `Instrumentation` CR instance in another namespace.
- `"false"` - do not inject

> **Note:** For `DotNet` auto-instrumentation, by default, operator sets the `OTEL_DOTNET_AUTO_TRACES_ENABLED_INSTRUMENTATIONS` environment variable which specifies the list of traces source instrumentations you want to enable. The value that is set by default by the operator is all available instrumentations supported by the `openTelemery-dotnet-instrumentation` release consumed in the image, i.e. `AspNet,HttpClient,SqlClient`. This value can be overriden by configuring the environment variable explicitly.

#### Multi-container pods with single instrumentation

If nothing else is specified, instrumentation is performed on the first container available in the pod spec.
In some cases (for example in the case of the injection of an Istio sidecar) it becomes necessary to specify on which container(s) this injection must be performed.

For this, it is possible to fine-tune the pod(s) on which the injection will be carried out.

For this, we will use the `instrumentation.opentelemetry.io/container-names` annotation for which we will indicate one or more container names (`.spec.containers.name`) on which the injection must be made:

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

#### Multi-container pods with multiple instrumentations

Works only when `enable-multi-instrumentation` flag is `true`.

Annotations defining which language instrumentation will be injected are required. When feature is enabled, specific for Instrumentation language containers annotations are used:

Java:

```bash
instrumentation.opentelemetry.io/java-container-names: "java1,java2"
```

NodeJS:

```bash
instrumentation.opentelemetry.io/nodejs-container-names: "nodejs1,nodejs2"
```

Python:

```bash
instrumentation.opentelemetry.io/python-container-names: "python1,python3"
```

DotNet:

```bash
instrumentation.opentelemetry.io/dotnet-container-names: "dotnet1,dotnet2"
```

Go:

```bash
instrumentation.opentelemetry.io/go-container-names: "go1"
```

ApacheHttpD:

```bash
instrumentation.opentelemetry.io/apache-httpd-container-names: "apache1,apache2"
```

NGINX:

```bash
instrumentation.opentelemetry.io/inject-nginx-container-names: "nginx1,nginx2"
```

SDK:

```bash
instrumentation.opentelemetry.io/sdk-container-names: "app1,app2"
```

If language instrumentation specific container names are not specified, instrumentation is performed on the first container available in the pod spec (only if single instrumentation injection is configured).

In some cases containers in the pod are using different technologies. It becomes necessary to specify language instrumentation for container(s) on which this injection must be performed.

For this, we will use language instrumentation specific container names annotation for which we will indicate one or more container names (`.spec.containers.name`) on which the injection must be made:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment-with-multi-containers-multi-instrumentations
spec:
  selector:
    matchLabels:
      app: my-pod-with-multi-containers-multi-instrumentations
  replicas: 1
  template:
    metadata:
      labels:
        app: my-pod-with-multi-containers-multi-instrumentations
      annotations:
        instrumentation.opentelemetry.io/inject-java: "true"
        instrumentation.opentelemetry.io/java-container-names: "myapp,myapp2"
        instrumentation.opentelemetry.io/inject-python: "true"
        instrumentation.opentelemetry.io/python-container-names: "myapp3"
    spec:
      containers:
        - name: myapp
          image: myImage1
        - name: myapp2
          image: myImage2
        - name: myapp3
          image: myImage3
```

In the above case, `myapp` and `myapp2` containers will be instrumented using Java and `myapp3` using Python instrumentation.

**NOTE**: Go auto-instrumentation **does not** support multicontainer pods. When injecting Go auto-instrumentation the first container should be the only you want to instrument.

**NOTE**: This type of instrumentation **does not** allow to instrument a container with multiple language instrumentations.

**NOTE**: `instrumentation.opentelemetry.io/container-names` annotation is not used for this feature.

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
  nginx:
    image: your-customized-auto-instrumentation-image:nginx
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
spec:
  apacheHttpd:
    image: your-customized-auto-instrumentation-image:apache-httpd
    version: "2.2"
    configPath: /your-custom-config-path
    attrs:
      - name: ApacheModuleOtelMaxQueueSize
        value: "4096"
      - name: ...
        value: ...
```

List of all available attributes can be found at [otel-webserver-module](https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module)

#### Using Nginx autoinstrumentation

For `Nginx` autoinstrumentation, Nginx versions 1.22.0, 1.23.0, and 1.23.1 are supported at this time. The Nginx configuration file is expected to be `/etc/nginx/nginx.conf` by default, if it's different, see following example on how to change it. Instrumentation at this time also expects, that `conf.d` directory is present in the directory, where configuration file resides and that there is a `include <config-file-dir-path>/conf.d/*.conf;` directive in the `http { ... }` section of Nginx configuration file (like it is in the default configuration file of Nginx). You can also adjust OpenTelemetry SDK attributes. Example:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  nginx:
    image: your-customized-auto-instrumentation-image:nginx # if custom instrumentation image is needed
    configFile: /my/custom-dir/custom-nginx.conf
    attrs:
      - name: NginxModuleOtelMaxQueueSize
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

The operator allows specifying, via the flags, which languages the Instrumentation resource may instrument.
If a language is enabled by default its gate only needs to be supplied when disabling the gate.
Language support can be disabled by passing the flag with a value of `false`.

| Language    | Gate                                  | Default Value |
| ----------- | ------------------------------------- | ------------- |
| Java        | `enable-java-instrumentation`         | `true`        |
| NodeJS      | `enable-nodejs-instrumentation`       | `true`        |
| Python      | `enable-python-instrumentation`       | `true`        |
| DotNet      | `enable-dotnet-instrumentation`       | `true`        |
| ApacheHttpD | `enable-apache-httpd-instrumentation` | `true`        |
| Go          | `enable-go-instrumentation`           | `false`       |
| Nginx       | `enable-nginx-instrumentation`        | `false`       |


OpenTelemetry Operator allows to instrument multiple containers using multiple language specific instrumentations.
These features can be enabled using the `enable-multi-instrumentation` flag. By default flag is `false`.

For more information about multi-instrumentation feature capabilities please see [Multi-container pods with multiple instrumentations](#Multi-container-pods-with-multiple-instrumentations).

### Target Allocator

The OpenTelemetry Operator comes with an optional component, the [Target Allocator](/cmd/otel-allocator/README.md) (TA). When creating an OpenTelemetryCollector Custom Resource (CR) and setting the TA as enabled, the Operator will create a new deployment and service to serve specific `http_sd_config` directives for each Collector pod as part of that CR. It will also rewrite the Prometheus receiver configuration in the CR, so that it uses the deployed target allocator. The following example shows how to get started with the Target Allocator:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: collector-with-ta
spec:
  mode: statefulset
  targetAllocator:
    enabled: true
  config:
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
            - action: labelmap
              regex: label_(.+)
              replacement: $$1

    exporters:
      debug: {}

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          exporters: [debug]
EOF
```

The usage of `$$` in the replacement keys in the example above is based on the information provided in the Prometheus receiver [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/prometheusreceiver/README.md) documentation, which states:
`Note: Since the collector configuration supports env variable substitution $ characters in your prometheus configuration are interpreted as environment variables. If you want to use $ characters in your prometheus configuration, you must escape them using $$.`

Behind the scenes, the OpenTelemetry Operator will convert the Collector’s configuration after the reconciliation into the following:

```yaml
receivers:
  prometheus:
    target_allocator:
      endpoint: http://collector-with-ta-targetallocator:80
      interval: 30s
      collector_id: $POD_NAME

exporters:
  debug:

service:
  pipelines:
    metrics:
      receivers: [prometheus]
      exporters: [debug]
```

The OpenTelemetry Operator will also convert the Target Allocator's Prometheus configuration after the reconciliation into the following:

```yaml
config:
  scrape_configs:
    - job_name: otel-collector
      scrape_interval: 10s
      static_configs:
        - targets: ["0.0.0.0:8888"]
      metric_relabel_configs:
        - action: labeldrop
          regex: (id|name)
        - action: labelmap
          regex: label_(.+)
          replacement: $1
```

Note that in this case, the Operator replaces "$$" with a single "$" in the replacement keys. This is because the collector supports environment variable substitution, whereas the TA (Target Allocator) does not. Therefore, to ensure compatibility, the TA configuration should only contain a single "$" symbol.

More info on the TargetAllocator can be found [here](cmd/otel-allocator/README.md).

#### Using Prometheus Custom Resources for service discovery

The target allocator can use Custom Resources from the prometheus-operator ecosystem, like ServiceMonitors and PodMonitors, for service discovery, performing
a function analogous to that of prometheus-operator itself. This is enabled via the `prometheusCR` section in the Collector CR.

See below for a minimal example:

```yaml
kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1beta1
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
      serviceMonitorSelector: {}
      podMonitorSelector: {}
  config:
    receivers:
      prometheus:
        config: {}

    exporters:
      debug: {}

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          exporters: [debug]
EOF
```

## Configure resource attributes

### Configure resource attributes with annotations

This example shows a pod configuration with OpenTelemetry annotations using the `resource.opentelemetry.io/` prefix. 
These annotations can be used to add resource attributes to data produced by OpenTelemetry instrumentation.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  annotations:
    # this is just an example, you can create any resource attributes you need
    resource.opentelemetry.io/service.name: "my-service"
    resource.opentelemetry.io/service.version: "1.0.0"
    resource.opentelemetry.io/deployment.environment.name: "production"
spec:
  containers:
  - name: main-container
    image: your-image:tag
```

### Configure resource attributes with labels

You can also use common labels to set resource attributes.

The following labels are supported:
- `app.kubernetes.io/name` becomes `service.name`
- `app.kubernetes.io/version` becomes `service.version`
- `app.kubernetes.io/part-of` becomes `service.namespace`

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  labels:
    app.kubernetes.io/name: "my-service"
    app.kubernetes.io/version: "1.0.0"
    app.kubernetes.io/part-of: "shop"
spec:
  containers:
  - name: main-container
    image: your-image:tag
```

This requires an explicit opt-in as follows:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  defaults:
    useLabelsForResourceAttributes: true
```

### Priority for setting resource attributes

The priority for setting resource attributes is as follows (first found wins):

1. Resource attributes set via `OTEL_RESOURCE_ATTRIBUTES` and `OTEL_SERVICE_NAME` environment variables
2. Resource attributes set via annotations (with the `resource.opentelemetry.io/` prefix)
3. Resource attributes set via labels (e.g. `app.kubernetes.io/name`)
   if the `Instrumentation` CR has defaults.useLabelsForResourceAttributes=true (see above)
4. Resource attributes calculated from the pod's metadata (e.g. `k8s.pod.name`)
5. Resource attributes set via the `Instrumentation` CR (in the `spec.resource.resourceAttributes` section)

This priority is applied for each resource attribute separately, so it is possible to set some attributes via
annotations and others via labels.

### How resource attributes are calculated from the pod's metadata

The following resource attributes are calculated from the pod's metadata.

#### How `service.name` is calculated

Choose the first value found: 

- `pod.annotation[resource.opentelemetry.io/service.name]`
- `if (config[useLabelsForResourceAttributes]) pod.label[app.kubernetes.io/name]`
- `k8s.deployment.name`
- `k8s.replicaset.name`
- `k8s.statefulset.name`
- `k8s.daemonset.name`
- `k8s.cronjob.name`
- `k8s.job.name`
- `k8s.pod.name`
- `k8s.container.name`

#### How `service.version` is calculated

Choose the first value found:

- `pod.annotation[resource.opentelemetry.io/service.version]`
- `if (cfg[useLabelsForResourceAttributes]) pod.label[app.kubernetes.io/version]`
- `if (contains(container docker image tag, '/') == false) container docker image tag`

#### How `service.instance.id` is calculated

Choose the first value found:
                                   
- `pod.annotation[resource.opentelemetry.io/service.instance.id]`
- `concat([k8s.namespace.name, k8s.pod.name, k8s.container.name], '.')`

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

In addition to the [core responsibilities](https://github.com/open-telemetry/community/blob/main/community-membership.md) the operator project requires approvers and maintainers to be responsible for releasing the project. See [RELEASE.md](./RELEASE.md) for more information and release schedule.

Approvers ([@open-telemetry/operator-approvers](https://github.com/orgs/open-telemetry/teams/operator-approvers)):

- [Tyler Helmuth](https://github.com/TylerHelmuth), Honeycomb
- [Yuri Oliveira Sa](https://github.com/yuriolisa), OllyGarden
- [Israel Blancas](https://github.com/iblancasa), Red Hat

Emeritus Approvers:

- [Anthony Mirabella](https://github.com/Aneurysm9), AWS
- [Dmitrii Anoshin](https://github.com/dmitryax), Splunk
- [Jay Camp](https://github.com/jrcamp), Splunk
- [James Bebbington](https://github.com/james-bebbington), Google
- [Owais Lone](https://github.com/owais), Splunk
- [Pablo Baeyens](https://github.com/mx-psi), DataDog

Maintainers ([@open-telemetry/operator-maintainers](https://github.com/orgs/open-telemetry/teams/operator-maintainers)):

- [Benedikt Bongartz](https://github.com/frzifus), Red Hat
- [Jacob Aronoff](https://github.com/jaronoff97), Lightstep
- [Mikołaj Świątek](https://github.com/swiatekm), Elastic
- [Pavol Loffay](https://github.com/pavolloffay), Red Hat

Emeritus Maintainers

- [Alex Boten](https://github.com/codeboten), Lightstep
- [Bogdan Drutu](https://github.com/BogdanDrutu), Splunk
- [Juraci Paixão Kröhling](https://github.com/jpkrohling), Grafana Labs
- [Tigran Najaryan](https://github.com/tigrannajaryan), Splunk
- [Vineeth Pothulapati](https://github.com/VineethReddy02), Timescale

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
