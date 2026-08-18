# Auto-instrumentation

The operator can inject and configure OpenTelemetry auto-instrumentation libraries. Currently, Apache HTTPD, DotNet, Go, Java, Nginx, NodeJS and Python are supported.

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
Deployment, StatefulSet, and other resources.

The possible values for the annotation can be

- `"true"` - inject and `Instrumentation` resource from the namespace.
- `"my-instrumentation"` - name of `Instrumentation` CR instance in the current namespace.
- `"my-other-namespace/my-instrumentation"` - name and namespace of `Instrumentation` CR instance in another namespace.
- `"false"` - do not inject

## Per-language guides

- [Java](languages/java.md)
- [Node.js](languages/nodejs.md)
- [Python](languages/python.md)
- [.NET](languages/dotnet.md)
- [Go](languages/go.md)
- [Apache HTTPD](languages/apache-httpd.md)
- [Nginx](languages/nginx.md)
- [SDK environment variables only](languages/sdk-only.md)

## Topics

- [Multi-container pods (single instrumentation)](multi-container.md)
- [Instrumenting init containers](init-containers.md)
- [Multi-container pods with multiple instrumentations](multi-instrumentation.md)
- [Using customized or vendor instrumentation images](custom-images.md)
- [Configuring resource attributes](resource-attributes.md)

See also the [API reference](../api/instrumentations.md).
