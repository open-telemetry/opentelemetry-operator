# Instrumentation v1beta1

This document outlines the next version of the Instrumentation CRD - `v1beta1`.

## Motivation

The current `v1alpha1` Instrumentation CRD has been widely adopted in production environments despite its `v1alpha1` version. Promoting to `v1beta1` signals API stability and provides an opportunity to address accumulated design issues. The `v1beta1` version should be a breaking change from `v1alpha1` that:

1. Aligns with OpenTelemetry's [declarative configuration](https://github.com/open-telemetry/opentelemetry-configuration) initiative
2. Fixes structural inconsistencies in the current API

## Objectives

1. Support strongly-typed [declarative configuration](https://github.com/open-telemetry/opentelemetry-configuration) in `spec.declarativeConfig` alongside existing env-var-based configuration in `spec.envConfig` ([#4093](https://github.com/open-telemetry/opentelemetry-operator/issues/4093))
2. Add explicit OTLP exporter protocol field to avoid ambiguity between HTTP and gRPC endpoints ([#3658](https://github.com/open-telemetry/opentelemetry-operator/issues/3658))
3. Normalize per-language resource fields — remove deprecated [`volumeLimitSize`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/apis/v1alpha1/instrumentation_types.go#L178) and unify inconsistent JSON tags ([`json:"resources"`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/apis/v1alpha1/instrumentation_types.go#L188) vs [`json:"resourceRequirements"`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/apis/v1alpha1/instrumentation_types.go#L228))
4. Consolidate `Resource` and `Defaults` into a single top-level `spec.resource` field for operator-level resource attribute configuration

## Non-Goals (for initial v1beta1)

* Label selectors for targeting workloads ([#2744](https://github.com/open-telemetry/opentelemetry-operator/issues/2744), [#821](https://github.com/open-telemetry/opentelemetry-operator/issues/821)) - additive feature, can be added later without breaking changes
* Webhook architecture separation ([#5010](https://github.com/open-telemetry/opentelemetry-operator/issues/5010), [#4115](https://github.com/open-telemetry/opentelemetry-operator/issues/4115)) - operational concern, not CRD spec
* Windows node support ([#642](https://github.com/open-telemetry/opentelemetry-operator/issues/642)) - can be added without breaking changes
* New language support - can be added incrementally

## Proposed Changes

### 1. SDK Declarative Configuration

**Issues:** [#4093](https://github.com/open-telemetry/opentelemetry-operator/issues/4093), [#4607](https://github.com/open-telemetry/opentelemetry-operator/issues/4607)

OpenTelemetry is standardizing on [file-based declarative configuration](https://opentelemetry.io/docs/specs/otel/configuration/) as the preferred way to configure SDKs. The v1beta1 CRD supports two mutually exclusive configuration approaches:

- **`spec.declarativeConfig`** — strongly-typed Go structs matching the [OTel SDK configuration schema](https://github.com/open-telemetry/opentelemetry-configuration). The operator serializes this to a YAML file, mounts it into the workload, and sets `OTEL_CONFIG_FILE`.
- **`spec.envConfig`** — the existing env-var-based configuration (`exporter`, `sampler`, `propagators`, `resource`), moved under a dedicated field. This preserves the current v1alpha1 behavior.

Setting both `declarativeConfig` and `envConfig` is invalid and rejected by the webhook.

#### Declarative config example

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: Instrumentation
metadata:
  name: declarative-example
spec:
  declarativeConfig:
    file_format: "0.4"
    resource:
      attributes:
        - name: service.namespace
          value: production
    tracer_provider:
      sampler:
        parent_based:
          root:
            trace_id_ratio_based:
              ratio: 0.25
      processors:
        - batch: {}
      exporters:
        - otlp:
            endpoint: http://collector:4318
            protocol: http/protobuf
```

#### Environment variable substitution in declarative config

The declarative config supports [environment variable substitution](https://opentelemetry.io/docs/specs/otel/configuration/file-configuration/#environment-variable-substitution) using the `${VAR}` syntax. This is useful for injecting secrets like API tokens without hardcoding them in the CR. Environment variables can be set via `spec.env` or per-language `env` fields.

This also addresses a limitation in v1alpha1 where Kubernetes `$(VAR)` substitution fails due to env var ordering ([#3022](https://github.com/open-telemetry/opentelemetry-operator/issues/3022)). Since `${VAR}` substitution in declarative config happens at SDK runtime rather than Kubernetes pod creation time, it works regardless of the order in which env vars are defined.

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: Instrumentation
metadata:
  name: declarative-with-secret
spec:
  env:
    - name: OTEL_EXPORTER_API_KEY
      valueFrom:
        secretKeyRef:
          name: otel-secrets
          key: api-key
  declarativeConfig:
    file_format: "0.4"
    tracer_provider:
      processors:
        - batch: {}
      exporters:
        - otlp:
            endpoint: https://otlp.example.com:4318
            protocol: http/protobuf
            headers:
              - name: x-api-key
                value: ${OTEL_EXPORTER_API_KEY}
```

#### Env-var config example (current behavior)

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: Instrumentation
metadata:
  name: env-example
spec:
  envConfig:
    exporter:
      endpoint: http://collector:4318
      protocol: http/protobuf
    sampler:
      type: parentbased_traceidratio
      argument: "0.25"
    propagators:
      - tracecontext
      - baggage
  resource:
    attributes:
      service.namespace: production
```

### 2. Explicit Exporter Protocol

**Issues:** [#3658](https://github.com/open-telemetry/opentelemetry-operator/issues/3658)

The v1alpha1 `spec.exporter` struct has a single `endpoint` field with no indication of whether it expects HTTP or gRPC. This is a common source of confusion because different SDK auto-instrumentation images default to different protocols:

| Language | Default OTLP Protocol | Default Port | Operator Override |
|----------|----------------------|--------------|-------------------|
| Java | `http/protobuf` | 4318 | No (SDK default since Java agent 2.x) |
| NodeJS | `http/protobuf` | 4318 | No (SDK default) |
| Python | `grpc` | 4317 | Yes — operator forces `http/protobuf` (port 4318) |
| DotNet | `http/protobuf` | 4318 | No (auto-instrumentation default; differs from .NET SDK which defaults to `grpc`) |
| Go | `http/protobuf` | 4318 | No (auto-instrumentation default; Go SDK itself defaults to `grpc`) |
| Apache HTTPD | `grpc` | 4317 | No (otel-webserver-module only supports gRPC; [proposal to add HTTP](https://github.com/open-telemetry/opentelemetry-cpp-contrib/issues/614)) |
| Nginx | `grpc` | 4317 | No (otel-webserver-module only supports gRPC; [proposal to add HTTP](https://github.com/open-telemetry/opentelemetry-cpp-contrib/issues/614)) |

The v1beta1 adds an explicit `protocol` field to the `Exporter` struct. When set, the operator injects `OTEL_EXPORTER_OTLP_PROTOCOL` alongside the endpoint. Valid values are `grpc`, `http/protobuf`, and `http/json`.

```go
type Exporter struct {
    Endpoint string `json:"endpoint,omitempty"`
    Protocol string `json:"protocol,omitempty"`
    TLS      *TLS   `json:"tls,omitempty"`
}
```

### 3. Normalize Per-Language Resource Fields

The current v1alpha1 has inconsistent JSON tags across language structs:
- Java uses [`"resources"`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/apis/v1alpha1/instrumentation_types.go#L188) while NodeJS/Python/DotNet/Go use [`"resourceRequirements"`](https://github.com/open-telemetry/opentelemetry-operator/blob/main/apis/v1alpha1/instrumentation_types.go#L228)
- The deprecated `volumeLimitSize` field exists on all languages

The v1beta1 normalizes all per-language structs to use a common base:

```go
// CommonLanguageSpec contains fields shared by all language-specific configurations.
type CommonLanguageSpec struct {
    Image               string                                 `json:"image,omitempty"`
    VolumeClaimTemplate corev1.PersistentVolumeClaimTemplate   `json:"volumeClaimTemplate,omitempty"`
    Env                 []corev1.EnvVar                        `json:"env,omitempty"`
    Resources           corev1.ResourceRequirements            `json:"resources,omitempty"`
}
```

Changes from v1alpha1:
- **Removed:** `VolumeSizeLimit` (`volumeLimitSize`) - deprecated in v1alpha1, use `volumeClaimTemplate` instead
- **Renamed:** `resourceRequirements` -> `resources` (consistent across all languages)

Language-specific extensions remain (Java `extensions`, Go `securityContext`, ApacheHttpd `version`/`configPath`/`attrs`, Nginx `configFile`/`attrs`).

### 4. Top-Level Resource Configuration

**Issues:** [#3775](https://github.com/open-telemetry/opentelemetry-operator/issues/3775)

In v1alpha1, `Resource` (user-defined attributes, `addK8sUIDAttributes`) and `Defaults` (`useLabelsForResourceAttributes`) are separate top-level fields, but both control how the operator populates resource attributes. In v1beta1 these are consolidated into a single top-level `spec.resource` field.

This field is independent of the SDK configuration mode (`declarativeConfig` vs `envConfig`) because it controls **operator-level injection behavior**, not SDK configuration. The operator injects K8s metadata and service identity attributes following the [OTel Semantic Conventions for K8s attributes](https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/).

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: Instrumentation
metadata:
  name: resource-example
spec:
  resource:
    # User-defined resource attributes injected into workloads
    attributes:
      deployment.environment.name: production
    # K8s resource attributes (k8s.pod.name, k8s.namespace.name, k8s.deployment.name, etc.)
    # See: https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
    k8sMetadata:
      # Set to false to disable K8s resource attribute injection. Defaults to true.
      enabled: true
      # Include K8s UID attributes (k8s.deployment.uid, k8s.replicaset.uid, etc.)
      includeUIDs: true
    # Service identity attributes (service.name, service.version, service.namespace, service.instance.id)
    # Derived from K8s metadata following OTel semantic conventions precedence.
    # See: https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
    serviceMetadata:
      # Set to false to disable automatic service attribute derivation. Defaults to true.
      enabled: true
  envConfig:
    exporter:
      endpoint: http://collector:4318
```

#### Injection behavior per config mode

When using `declarativeConfig`, the operator mounts a YAML config file and sets `OTEL_CONFIG_FILE` to point to it. In this mode, [all other OTel environment variables are ignored by the SDK](https://opentelemetry.io/docs/languages/sdk-configuration/declarative-configuration/) unless explicitly referenced via `${VAR}` substitution syntax in the config file. This means `OTEL_RESOURCE_ATTRIBUTES` would not work alongside declarative config.

The operator handles this differently depending on the active config mode:

- `envConfig` mode — the operator sets `OTEL_RESOURCE_ATTRIBUTES` env var with the computed attributes (current v1alpha1 behavior).
- `declarativeConfig` mode — the operator merges the computed attributes directly into the `resource.attributes` list in the serialized YAML config file before mounting it into the workload. No `OTEL_RESOURCE_ATTRIBUTES` env var is needed.

In both cases, user-defined attributes from `spec.resource.attributes` are included with the lowest precedence, followed by K8s metadata, then pod annotations (`resource.opentelemetry.io/*`).

## CRD Spec

Full proposed v1beta1 `InstrumentationSpec`:

```go
type InstrumentationSpec struct {
    // DeclarativeConfig defines the OTel SDK configuration as strongly-typed fields
    // matching the OTel declarative configuration schema.
    // The operator serializes this to a YAML file and mounts it into the workload.
    // Mutually exclusive with EnvConfig.
    // +optional
    DeclarativeConfig *DeclarativeConfig `json:"declarativeConfig,omitempty"`

    // EnvConfig defines the SDK configuration via environment variables.
    // This is the same configuration model as v1alpha1 (exporter, sampler, propagators, resource).
    // Mutually exclusive with DeclarativeConfig.
    // +optional
    EnvConfig *EnvConfig `json:"envConfig,omitempty"`

    // Resource defines operator-level resource attribute configuration.
    // These settings control how the operator populates resource attributes
    // and apply regardless of whether declarativeConfig or envConfig is used.
    // +optional
    Resource Resource `json:"resource,omitempty"`

    // Env defines common env vars.
    // Precedence: original container env > language-specific env > common env > SDK config.
    // +optional
    Env []corev1.EnvVar `json:"env,omitempty"`

    // Java defines configuration for Java auto-instrumentation.
    // +optional
    Java Java `json:"java,omitempty"`

    // NodeJS defines configuration for NodeJS auto-instrumentation.
    // +optional
    NodeJS NodeJS `json:"nodejs,omitempty"`

    // Python defines configuration for Python auto-instrumentation.
    // +optional
    Python Python `json:"python,omitempty"`

    // DotNet defines configuration for DotNet auto-instrumentation.
    // +optional
    DotNet DotNet `json:"dotnet,omitempty"`

    // Go defines configuration for Go auto-instrumentation.
    // +optional
    Go Go `json:"go,omitempty"`

    // ApacheHttpd defines configuration for Apache HTTPD auto-instrumentation.
    // +optional
    ApacheHttpd ApacheHttpd `json:"apacheHttpd,omitempty"`

    // Nginx defines configuration for Nginx auto-instrumentation.
    // +optional
    Nginx Nginx `json:"nginx,omitempty"`

    // ImagePullPolicy defines the image pull policy for init containers.
    // +optional
    ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

    // InitContainerSecurityContext applied to auto-instrumentation init containers.
    // +optional
    InitContainerSecurityContext *corev1.SecurityContext `json:"initContainerSecurityContext,omitempty"`
}

// DeclarativeConfig mirrors the OTel SDK configuration schema as strongly-typed Go structs.
// See https://github.com/open-telemetry/opentelemetry-configuration for the full schema.
// The exact struct definitions will be generated from or aligned with the upstream schema.
type DeclarativeConfig struct {
    // FileFormat is the OTel configuration schema version (e.g. "0.4").
    FileFormat string `json:"file_format"`

    // Disabled controls whether the SDK is disabled.
    // +optional
    Disabled *bool `json:"disabled,omitempty"`

    // Resource defines resource attributes configuration.
    // +optional
    Resource *ResourceConfig `json:"resource,omitempty"`

    // Propagator defines context propagation configuration.
    // +optional
    Propagator *PropagatorConfig `json:"propagator,omitempty"`

    // TracerProvider defines tracer provider configuration (samplers, processors, exporters).
    // +optional
    TracerProvider *TracerProviderConfig `json:"tracer_provider,omitempty"`

    // MeterProvider defines meter provider configuration (readers, views).
    // +optional
    MeterProvider *MeterProviderConfig `json:"meter_provider,omitempty"`

    // LoggerProvider defines logger provider configuration (processors, exporters).
    // +optional
    LoggerProvider *LoggerProviderConfig `json:"logger_provider,omitempty"`
}

// EnvConfig defines the env-var-based SDK configuration (same as v1alpha1 top-level fields).
type EnvConfig struct {
    // Exporter defines exporter configuration.
    // +optional
    Exporter Exporter `json:"exporter,omitempty"`

    // Propagators defines inter-process context propagation configuration.
    // +optional
    Propagators []Propagator `json:"propagators,omitempty"`

    // Sampler defines sampling configuration.
    // +optional
    Sampler Sampler `json:"sampler,omitempty"`
}

// Resource defines operator-level resource attribute configuration.
// These fields control how the operator populates resource attributes and
// are independent of the SDK configuration mode (declarativeConfig vs envConfig).
// See: https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
type Resource struct {
    // Attributes defines resource attributes to inject into the workload.
    // +optional
    Attributes map[string]string `json:"attributes,omitempty"`

    // K8sMetadata controls K8s resource attribute injection (k8s.pod.name, k8s.namespace.name, etc.).
    // +optional
    K8sMetadata *K8sMetadataConfig `json:"k8sMetadata,omitempty"`

    // ServiceMetadata controls service identity attribute derivation (service.name, service.version, etc.).
    // +optional
    ServiceMetadata *ServiceMetadataConfig `json:"serviceMetadata,omitempty"`
}

// K8sMetadataConfig defines how Kubernetes resource attributes are injected.
// Controls attributes like k8s.pod.name, k8s.namespace.name, k8s.deployment.name, k8s.node.name, etc.
type K8sMetadataConfig struct {
    // Enabled controls whether K8s resource attributes are automatically injected.
    // When false, no k8s.* attributes are added. Defaults to true.
    // +optional
    Enabled *bool `json:"enabled,omitempty"`

    // IncludeUIDs defines whether K8s UID attributes should be collected
    // (e.g. k8s.deployment.uid, k8s.replicaset.uid). Only applies when Enabled is true.
    // +optional
    IncludeUIDs bool `json:"includeUIDs,omitempty"`
}

// ServiceMetadataConfig defines how service identity attributes are derived from K8s metadata.
// Controls attributes: service.name, service.version, service.namespace, service.instance.id.
// Follows OTel semantic conventions precedence: https://opentelemetry.io/docs/specs/semconv/non-normative/k8s-attributes/
type ServiceMetadataConfig struct {
    // Enabled controls whether service identity attributes are automatically derived.
    // When false, no service.* attributes are added by the operator. Defaults to true.
    // +optional
    Enabled *bool `json:"enabled,omitempty"`
}
```

The exact child types for `DeclarativeConfig` (`ResourceConfig`, `TracerProviderConfig`, etc.) will be defined to match the [OTel configuration schema](https://github.com/open-telemetry/opentelemetry-configuration).

The `EnvConfig` types (`Exporter`, `Sampler`, `Propagator`) are the same as v1alpha1, just moved under `spec.envConfig`. The `Exporter` type gains a new `protocol` field. The `Resource` type is promoted to the top level of `InstrumentationSpec` since it controls operator-level injection behavior that applies to both config modes.

## Breaking Changes from v1alpha1

| Change | v1alpha1 | v1beta1 | Migration |
|--------|----------|---------|-----------|
| SDK configuration | `exporter`, `sampler`, `propagators`, `resource` at top level | `exporter`, `sampler`, `propagators` moved under `spec.envConfig`, or use new `spec.declarativeConfig` | Wrap existing fields under `envConfig`, or migrate to declarative config |
| Resource attributes | `spec.resource.resourceAttributes`, `spec.resource.addK8sUIDAttributes`, `spec.defaults.useLabelsForResourceAttributes` | Consolidated into `spec.resource.attributes`, `spec.resource.k8sMetadata`, and `spec.resource.serviceMetadata` | Rename `resourceAttributes` to `attributes`, use `k8sMetadata` for k8s.* attributes, use `serviceMetadata` for service.* derivation |
| Per-language resources JSON tag | Mixed (`resources` / `resourceRequirements`) | `resources` (all) | Rename in YAML for NodeJS, Python, DotNet, Go, ApacheHttpd, Nginx |
| Volume size limit | `volumeLimitSize` (deprecated) | Removed | Use `volumeClaimTemplate` |

## Migration Strategy

1. **Conversion webhook**: Implement a conversion webhook that translates between v1alpha1 and v1beta1, handling field renames and removals automatically.
2. **Dual-version support**: Serve both v1alpha1 and v1beta1 simultaneously with v1beta1 as the storage version.
3. **Deprecation timeline**: v1alpha1 is deprecated when v1beta1 ships.

## Related Issues

| Category | Issues |
|----------|--------|
| Label selectors (future) | [#2744](https://github.com/open-telemetry/opentelemetry-operator/issues/2744), [#821](https://github.com/open-telemetry/opentelemetry-operator/issues/821), [#4445](https://github.com/open-telemetry/opentelemetry-operator/issues/4445) |
| Declarative config | [#4093](https://github.com/open-telemetry/opentelemetry-operator/issues/4093), [#4607](https://github.com/open-telemetry/opentelemetry-operator/issues/4607) |
| Exporter improvements | [#3658](https://github.com/open-telemetry/opentelemetry-operator/issues/3658), [#3390](https://github.com/open-telemetry/opentelemetry-operator/issues/3390), [#2180](https://github.com/open-telemetry/opentelemetry-operator/issues/2180) |
| Env handling | [#3022](https://github.com/open-telemetry/opentelemetry-operator/issues/3022), [#3775](https://github.com/open-telemetry/opentelemetry-operator/issues/3775), [#4559](https://github.com/open-telemetry/opentelemetry-operator/issues/4559) |
| Security context | [#2272](https://github.com/open-telemetry/opentelemetry-operator/issues/2272), [#2053](https://github.com/open-telemetry/opentelemetry-operator/issues/2053) |
| API stability | [#5060](https://github.com/open-telemetry/opentelemetry-operator/issues/5060) |
| Resource attributes | [#3775](https://github.com/open-telemetry/opentelemetry-operator/issues/3775), [#938](https://github.com/open-telemetry/opentelemetry-operator/issues/938) |
| Rollout on change | [#553](https://github.com/open-telemetry/opentelemetry-operator/issues/553) |
