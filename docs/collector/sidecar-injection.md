# Sidecar injection

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
