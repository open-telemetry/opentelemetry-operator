# Manager configuration reference

The OpenTelemetry Operator manager accepts configuration through four mechanisms, applied in this order (later sources override earlier ones):

1. **Built-in defaults** (`internal/config.New()`)
2. **`--config-file`** — YAML file (see [examples](#yaml-config-file))
3. **Environment variables**
4. **Command-line flags**

The `operator` and `webhook-server` subcommands share this configuration model. Additional controller-runtime logging flags (`--zap-log-level`, `--zap-time-encoding`, and related `--zap-*` flags) are registered separately and are not part of the table below.

Operator-wide feature gates use `--feature-gates` / `FEATURE_GATES` and are documented separately from auto-instrumentation language gates in [feature-gates.md](feature-gates.md).

## YAML config file

Pass a file with `--config-file`:

```yaml
watch-namespace: observability
enable-leader-election: true
metrics-addr: ":8443"
tls:
  minversion: VersionTLS13
  configuresoperands: true
zap:
  level-format: lowercase
```

Nested blocks use lowercase keys matching the struct tags in `internal/config/config.go`. See `internal/config/testdata/` for full examples including default instrumentation when CRDs are absent.

## General

| YAML key | CLI flag | Environment variable | Default | Description |
|----------|----------|----------------------|---------|-------------|
| — | `--config-file` | — | — | Path to YAML configuration file (Cobra flag, not part of `Config`) |
| `watch-namespace` | `--watch-namespace` | `WATCH_NAMESPACE` | `""` | Comma-separated namespaces to watch; empty watches all |
| `enable-leader-election` | `--enable-leader-election` | `ENABLE_LEADER_ELECTION` | `false` | Enable controller-manager leader election |
| `enable-webhooks` | `--enable-webhooks` | `ENABLE_WEBHOOKS` | `true` | Enable admission webhooks |
| `webhook-port` | `--webhook-port` | `WEBHOOK_PORT` | `9443` | Webhook server port |
| `ignore-missing-collector-crds` | `--ignore-missing-collector-crds` | `IGNORE_MISSING_COLLECTOR_CRDS` | `false` | Continue if OpenTelemetryCollector CRD is missing |
| `openshift-webhook-replicas` | `--openshift-webhook-replicas` | `OPENSHIFT_WEBHOOK_REPLICAS` | `2` | Standalone webhook replicas on OpenShift OLM (`0` disables) |

## Metrics, health, and profiling

| YAML key | CLI flag | Environment variable | Default | Description |
|----------|----------|----------------------|---------|-------------|
| `metrics-addr` | `--metrics-addr` | `METRICS_ADDR` | `:8443` | Metrics endpoint listen address |
| `metrics-secure` | `--metrics-secure` | `METRICS_SECURE` | `true` | Serve metrics over HTTPS with auth |
| `metrics-tls-cert-file` | `--metrics-tls-cert-file` | `METRICS_TLS_CERT_FILE` | `""` | Metrics TLS certificate |
| `metrics-tls-key-file` | `--metrics-tls-key-file` | `METRICS_TLS_KEY_FILE` | `""` | Metrics TLS private key |
| `enable-cr-metrics` | `--enable-cr-metrics` | `ENABLE_CR_METRICS` | `false` | Expose custom resource state metrics |
| `create-service-monitor-operator-metrics` | `--create-sm-operator-metrics` | `CREATE_SM_OPERATOR_METRICS` | `false` | Create ServiceMonitor for operator metrics |
| `health-probe-addr` | `--health-probe-addr` | `HEALTH_PROBE_ADDR` | `:8081` | Health/readiness probe address |
| `pprof-addr` | `--pprof-addr` | `PPROF_ADDR` | `""` | pprof listen address; empty disables pprof |

## TLS (`tls:` block in YAML)

| YAML key | CLI flag | Environment variable | Default | Description |
|----------|----------|----------------------|---------|-------------|
| `tls.useclusterprofile` | `--tls-cluster-profile` | `TLS_CLUSTER_PROFILE` | `false` | Read TLS profile from OpenShift APIServer CR |
| `tls.configureoperands` | `--tls-configure-operands` | `TLS_CONFIGURE_OPERANDS` | `false` | Apply TLS min version/ciphers to operands |
| `tls.minversion` | `--tls-min-version` | `TLS_MIN_VERSION` | `VersionTLS12` | Minimum TLS version for operands |
| `tls.ciphersuites` | `--tls-cipher-suites` | `TLS_CIPHER_SUITES` | — | Comma-separated cipher suites |

## Logging encoder (`zap:` block in YAML)

These keys customize the structured log encoder. Controller-runtime log level and encoding use separate `--zap-log-level` / `--zap-time-encoding` flags.

| YAML key | CLI flag | Environment variable | Default |
|----------|----------|----------------------|---------|
| `zap.message-key` | `--zap-message-key` | `ZAP_MESSAGE_KEY` | `message` |
| `zap.level-key` | `--zap-level-key` | `ZAP_LEVEL_KEY` | `level` |
| `zap.time-key` | `--zap-time-key` | `ZAP_TIME_KEY` | `timestamp` |
| `zap.level-format` | `--zap-level-format` | `ZAP_LEVEL_FORMAT` | `uppercase` |

## Filters

| YAML key | CLI flag | Environment variable | Default |
|----------|----------|----------------------|---------|
| `labels-filter` | `--labels-filter` (repeatable) | `LABELS_FILTER` (comma-separated) | `[]` |
| `annotations-filter` | `--annotations-filter` (repeatable) | `ANNOTATIONS_FILTER` (comma-separated) | `[kubectl.kubernetes.io/last-applied-configuration]` |

## Component images

Default container images are version-tagged at build time. In OLM deployments, `RELATED_IMAGE_*` environment variables override the CLI defaults.

| YAML key | CLI flag | OLM environment variable |
|----------|----------|--------------------------|
| `collector-image` | `--collector-image` | `RELATED_IMAGE_COLLECTOR` |
| `targetallocator-image` | `--target-allocator-image` | `RELATED_IMAGE_TARGET_ALLOCATOR` |
| `operatoropampbridge-image` | `--operator-opamp-bridge-image` | `RELATED_IMAGE_OPERATOR_OPAMP_BRIDGE` |
| `auto-instrumentation-java-image` | `--auto-instrumentation-java-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_JAVA` |
| `auto-instrumentation-node-js-image` | `--auto-instrumentation-nodejs-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_NODEJS` |
| `auto-instrumentation-python-image` | `--auto-instrumentation-python-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON` |
| `auto-instrumentation-dot-net-image` | `--auto-instrumentation-dotnet-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_DOTNET` |
| `auto-instrumentation-go-image` | `--auto-instrumentation-go-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_GO` |
| `auto-instrumentation-apache-httpd-image` | `--auto-instrumentation-apache-httpd-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_APACHE_HTTPD` |
| `auto-instrumentation-nginx-image` | `--auto-instrumentation-nginx-image` | `RELATED_IMAGE_AUTO_INSTRUMENTATION_NGINX` |

## Auto-instrumentation toggles

| YAML key | CLI flag | Environment variable | Default |
|----------|----------|----------------------|---------|
| `enable-multi-instrumentation` | `--enable-multi-instrumentation` | `ENABLE_MULTI_INSTRUMENTATION` | `true` |
| `enable-java-auto-instrumentation` | `--enable-java-instrumentation` | `ENABLE_JAVA_AUTO_INSTRUMENTATION` | `true` |
| `enable-node-js-auto-instrumentation` | `--enable-nodejs-instrumentation` | `ENABLE_NODEJS_AUTO_INSTRUMENTATION` | `true` |
| `enable-python-auto-instrumentation` | `--enable-python-instrumentation` | `ENABLE_PYTHON_AUTO_INSTRUMENTATION` | `true` |
| `enable-dot-net-auto-instrumentation` | `--enable-dotnet-instrumentation` | `ENABLE_DOTNET_AUTO_INSTRUMENTATION` | `true` |
| `enable-go-auto-instrumentation` | `--enable-go-instrumentation` | `ENABLE_GO_AUTO_INSTRUMENTATION` | `false` |
| `enable-nginx-auto-instrumentation` | `--enable-nginx-instrumentation` | `ENABLE_NGINX_AUTO_INSTRUMENTATION` | `false` |
| `enable-apache-httpd-instrumentation` | `--enable-apache-httpd-instrumentation` | `ENABLE_APACHE_HTTPD_AUTO_INSTRUMENTATION` | `true` |

## OpenShift and FIPS

| YAML key | CLI flag | Environment variable | Default |
|----------|----------|----------------------|---------|
| `openshift-create-dashboard` | `--openshift-create-dashboard` | `OPENSHIFT_CREATE_DASHBOARD` | `false` |
| `fips-disabled-components` | `--fips-disabled-components` | `FIPS_DISABLED_COMPONENTS` | `uppercase` |

## Feature gates

| YAML key | CLI flag | Environment variable |
|----------|----------|----------------------|
| `feature-gates` | `--feature-gates` | `FEATURE_GATES` |

Format: comma-separated list; prefix with `-` to disable (for example `gate1,-gate2`).

## File-only keys

These fields are read from YAML only and have no CLI or environment variable equivalent:

| YAML key | Description |
|----------|-------------|
| `enable-instrumentation-crds` | When `false`, use embedded default instrumentation instead of CRDs |
| `instrumentations` | Default `Instrumentation` spec when CRDs are absent |
| `collector-configmap-entry` | ConfigMap key for collector config (default: `collector.yaml`) |
| `target-allocator-configmap-entry` | ConfigMap key for Target Allocator config |
| `operator-op-amp-bridge-configmap-entry` | ConfigMap key for OpAMP bridge config |
| `create-rbac-permissions` | Deprecated; use CLI `--create-rbac-permissions` only |

HTTP(S) proxy settings (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`) are captured from the operator pod environment and propagated to managed workloads; they are not part of the YAML schema.

## Source of truth

Configuration structs and defaults: [`internal/config/config.go`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/internal/config/config.go).
