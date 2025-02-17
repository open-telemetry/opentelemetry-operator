# CRD Changelog

This document explains major changes made in new CRD versions. It is intended to help users migrate and take
advantage of the new features.

## TargetAllocator.opentelemetry.io/v1alpha1

The target allocator is an application that can allocate Prometheus scrape targets to OpenTelemetry Collectors using the prometheus receiver,
allowing transparent horizontal scaling for Prometheus metrics collection. You can learn more in the target allocator's [README](../cmd/otel-allocator/README.md).

Until now, it could be enabled via the `targetAllocator` sub-resource in the OpenTelemetryCollector CR. This was, and continues to be fine for
simpler use cases. Some users needed to customize the target allocator further, and embedding all the required attributes in the
OpenTelemetryCollector CR would've made it unnecessarily large. Instead, we introduced a separate CRD for the target allocator.

The following OpenTelemetryCollector CR:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  targetAllocator:
     enabled: true
     prometheusCR:
        enabled: true
  config:
    receivers:
      prometheus:
         config:
            scrape_configs: []
    processors:
      batch:
        send_batch_size: 1000
        timeout: 10s
    exporters:
      debug: {}
    service:
      pipelines:
        traces:
          receivers: [prometheus]
          processors: [batch]
          exporters: [debug]
```

is now equivalent to the pair:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
  labels:
     opentelemetry.io/target-allocator: simplest-ta
spec:
  config:
    receivers:
      prometheus:
         config:
            scrape_configs: []
    processors:
      batch:
        send_batch_size: 1000
        timeout: 10s
    exporters:
      debug: {}
    service:
      pipelines:
        traces:
          receivers: [prometheus]
          processors: [batch]
          exporters: [debug]
---
apiVersion: opentelemetry.io/v1alpha1
kind: TargetAllocator
metadata:
   name: simplest-ta
spec:
   prometheusCR:
      enabled: true
```

> [!NOTE]  
> The OpenTelemetryCollector is connected to the TargetAllocator by setting the `opentelemetry.io/target-allocator` label on the former.

## OpenTelemetryCollector.opentelemetry.io/v1beta1 

### Migration

There is no need for any immediate user action. The operator will continue to support existing `v1alpha1` resources.

In addition, any newly applied `v1alpha1` resource will be converted to `v1beta1` and stored in the new API version.

The plan is to remove support for `v1alpha1` in a future operator version, so users should migrate promptly. In order to migrate fully to `v1beta1`:

1. Update any manifests you have stored outside the cluster, for example in your infrastructure git repository.
2. Apply them, so they're all stored as `v1beta1`.
3. Update the OpenTelemetryCollector CRD to only store `v1beta1`
   ```bash
   kubectl patch customresourcedefinitions opentelemetrycollectors.opentelemetry.io  \
     --subresource='status' \
     --type='merge' \
     -p '{"status":{"storedVersions":["v1beta1"]}}'
   ```
For a more thorough explanation of how and why this migration works, see the relevant [Kubernetes documentation][crd_migration_guide].

#### Operator Lifecycle Manager

If you're installing the opentelemetry-operator in OpenShift using OLM, be advised that
**only `AllNamespaces` install mode is now supported**, due to the conversion webhook from `v1beta1` to `v1alpha1`.
See [OLM docs](https://olm.operatorframework.io/docs/tasks/install-operator-with-olm/) and
[OLM operator groups docs](https://olm.operatorframework.io/docs/advanced-tasks/operator-scoping-with-operatorgroups/).

### Structured Configuration

The `Config` field containing the Collector configuration is a string in `v1alpha1`. This has some downsides:

- It's easy to make YAML formatting errors in the content.
- The field can have a lot of content, and may not show useful diffs for changes.
- It's more difficult for the operator to reject invalid configurations at admission.

To solve these issues, we've changed the type of this field to a structure aligned with OpenTelemetry Collector configuration
format. For example:

```yaml
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
      debug:

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [debug]
```

becomes:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config:
    receivers:
      otlp:
        protocols:
          grpc: {}
          http: {}
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
```

> [!NOTE]  
> Empty maps, like `debug:` in the above configuration, should have an explicit value of `{}`.

### Standard label selectors for Target Allocator

Configuring the target allocator to use Prometheus CRDs can involve setting label selectors for said CRDs. In the
`v1alpha1` Collector, these were simply maps representing the required labels. In order to allow more complex label
selection rules and align with Kubernetes' recommended way of solving this kind of problem, we've switched to
[standard selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/).

For example, in `v1alpha1` we'd have:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  targetAllocator:
    prometheusCR:
      serviceMonitorSelector:
        key: value
```

And in `v1beta1`:

```yaml
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  targetAllocator:
    prometheusCR:
      serviceMonitorSelector:
        matchLabels:   
          key: value
```

> [!NOTE]  
> A `nil` selector now selects no resources, while an empty selector selects all of them. To get the old default behaviour, it's necessary to set `serviceMonitorSelector: {}`.

### Default Collector image

The OpenTelemetry Collector maintainers recently introduced a [Collector distribution][k8s_distro] specifically aimed at 
Kubernetes workloads.

Our intent is to eventually use this distribution as our default collector image, as opposed to the 
[core distribution][core_distro] we're currently using. After some debate, we've decided NOT to make this change in
`v1beta1`, but rather roll it out more gradually, and with more warning to users. See [this issue][k8s_issue] for more information.


[core_distro]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol
[k8s_distro]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-k8s
[k8s_issue]: https://github.com/open-telemetry/opentelemetry-operator/issues/2835
[crd_migration_guide]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version