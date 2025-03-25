# Running the Operator

This page describes operational considerations for running the OpenTelemetry Operator.

## Configuration

The OpenTelemetry Operator can be configured using environment variables:

| Environment Variable | Description | Default |
|----------------------|-------------|---------|
| `OPAMP_VERSION` | The version of the OpAmp | `""` |
| `ENABLE_WEBHOOKS` | Enable or disable the admission webhooks | `true` |
| `OTELCOL_NAMESPACE` | The namespace where the OpenTelemetry Collector is installed | `""` |
| `WATCH_NAMESPACE` | The namespace to watch for OpenTelemetry resources | `""` |