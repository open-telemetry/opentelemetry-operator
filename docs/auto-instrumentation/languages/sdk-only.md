# OpenTelemetry SDK environment variables only

```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```

You can configure the OpenTelemetry SDK for applications which can't currently be autoinstrumented by using `inject-sdk` in place of `inject-python` or `inject-java`, for example. This will inject environment variables like `OTEL_RESOURCE_ATTRIBUTES`, `OTEL_TRACES_SAMPLER`, and `OTEL_EXPORTER_OTLP_ENDPOINT`, that you can configure in the `Instrumentation`, but will not actually provide the SDK.

```bash
instrumentation.opentelemetry.io/inject-sdk: "true"
```
