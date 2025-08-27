# CLI Flags Reference

This document lists all available CLI flags for the OpenTelemetry Operator. These configuration flags can also be parsed as JSON using the same keys mentioned below.

## Flag Table by Categories

### Core Configuration

| Name                     | Type   | Description                                                                                       |
|--------------------------|--------|---------------------------------------------------------------------------------------------------|
| **metrics-addr**         | string | The address the metric endpoint binds to.                                                         |
| **health-probe-addr**    | string | The address the probe endpoint binds to.                                                          |
| **pprof-addr**           | string | The address to expose the pprof server. Default is an empty string, which disables the pprof server.   |
| **enable-leader-election** | bool | Enable leader election for controller manager. Ensures there is only one active controller manager. |
| **webhook-port**         | int    | The port the webhook endpoint binds to.                                                           |
| **enable-webhooks**      | bool   | Enable webhooks for the controllers.                                                              |

---

### Auto-Instrumentation Enablement

| Name                                | Type | Description                                                               |
|-------------------------------------|------|---------------------------------------------------------------------------|
| **enable-multi-instrumentation**    | bool | Controls whether the operator supports multi-instrumentation.             |
| **enable-apache-httpd-instrumentation** | bool | Controls whether the operator supports Apache HTTPD auto-instrumentation. |
| **enable-dotnet-instrumentation**   | bool | Controls whether the operator supports dotnet auto-instrumentation.       |
| **enable-go-instrumentation**       | bool | Controls whether the operator supports Go auto-instrumentation.           |
| **enable-python-instrumentation**   | bool | Controls whether the operator supports Python auto-instrumentation.       |
| **enable-nginx-instrumentation**    | bool | Controls whether the operator supports Nginx auto-instrumentation.        |
| **enable-nodejs-instrumentation**   | bool | Controls whether the operator supports Node.js auto-instrumentation.      |
| **enable-java-instrumentation**     | bool | Controls whether the operator supports Java auto-instrumentation.         |

---

### Image Configuration

| Name                                 | Type   | Description                                                                                           |
|--------------------------------------|--------|-------------------------------------------------------------------------------------------------------|
| **collector-image**                  | string | Default OpenTelemetry collector image (used if not specified in the CustomResource).                  |
| **target-allocator-image**           | string | Default OpenTelemetry target allocator image (used if not specified in the CustomResource).           |
| **operator-opamp-bridge-image**      | string | Default OpenTelemetry Operator OpAMP Bridge image (used if not specified in the CustomResource).      |
| **auto-instrumentation-java-image**  | string | Default OpenTelemetry Java instrumentation image (used if not specified in the CustomResource).       |
| **auto-instrumentation-nodejs-image**| string | Default OpenTelemetry NodeJS instrumentation image (used if not specified in the CustomResource).     |
| **auto-instrumentation-python-image**| string | Default OpenTelemetry Python instrumentation image (used if not specified in the CustomResource).     |
| **auto-instrumentation-dotnet-image**| string | Default OpenTelemetry DotNet instrumentation image (used if not specified in the CustomResource).     |
| **auto-instrumentation-go-image**    | string | Default OpenTelemetry Go instrumentation image (used if not specified in the CustomResource).         |
| **auto-instrumentation-apache-httpd-image** | string | Default OpenTelemetry Apache HTTPD instrumentation image (used if not specified in the CustomResource).|
| **auto-instrumentation-nginx-image** | string | Default OpenTelemetry Nginx instrumentation image (used if not specified in the CustomResource).      |

---

### Monitoring & Observability

| Name                        | Type | Description                                                                 |
|-----------------------------|------|-----------------------------------------------------------------------------|
| **enable-cr-metrics**       | bool | Controls whether exposing the CR metrics is enabled.                        |
| **create-sm-operator-metrics** | bool | Create a ServiceMonitor for the operator metrics.                           |
| **openshift-create-dashboard** | bool | Create an OpenShift dashboard for monitoring OpenTelemetryCollector instances.|

---

### Security & TLS

| Name                | Type     | Description                                                                                   |
|---------------------|----------|-----------------------------------------------------------------------------------------------|
| **tls-min-version** | string   | Minimum TLS version supported (must match constants from [Go crypto/tls](https://golang.org/pkg/crypto/tls/#pkg-constants)). |
| **tls-cipher-suites** | string[] | Comma-separated list of cipher suites. Values from [Go crypto/tls](https://golang.org/pkg/crypto/tls/#pkg-constants). Defaults to Go's cipher suites. |

---

### Logging Configuration

| Name              | Type   | Description                                              |
|-------------------|--------|----------------------------------------------------------|
| **zap-message-key** | string | The message key used in the customized Log Encoder.     |
| **zap-level-key**   | string | The level key used in the customized Log Encoder.       |
| **zap-time-key**    | string | The time key used in the customized Log Encoder.        |
| **zap-level-format**| string | The level format used in the customized Log Encoder.    |

---

### Filtering & Other

| Name                       | Type     | Description                                                                                           |
|----------------------------|----------|-------------------------------------------------------------------------------------------------------|
| **labels-filter**          | string[] | Labels to filter away from propagating onto deploys. Supports `*` wildcards. Example: `--labels-filter=.*filter.out`. |
| **annotations-filter**     | string[] | Annotations to filter away from propagating onto deploys. Supports `*` wildcards. Example: `--annotations-filter=.*filter.out`. |
| **fips-disabled-components** | string | Disabled collector components on FIPS-enabled platforms. Example: `receiver.foo, receiver.bar, exporter.baz`. |
| **ignore-missing-collector-crds** | bool | Ignore missing OpenTelemetryCollector CRDs in the cluster.                                             |
| **create-rbac-permissions** | bool    | Automatically create RBAC permissions needed by processors (**deprecated**).                           |
