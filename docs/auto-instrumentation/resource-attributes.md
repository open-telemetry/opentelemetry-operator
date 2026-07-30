# Configure resource attributes

The OpenTelemetry Operator can automatically set resource attributes as defined in the 
[OpenTelemetry Semantic Conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/non-normative/k8s-attributes.md).

## Configure resource attributes with annotations

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

## Configure resource attributes with labels

You can also use common labels to set resource attributes (first entry wins).

The following labels are supported:
- `app.kubernetes.io/instance` becomes `service.name`
- `app.kubernetes.io/name` becomes `service.name`
- `app.kubernetes.io/version` becomes `service.version`

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

## Priority for setting resource attributes

The priority for setting resource attributes is as follows (first found wins):

1. Resource attributes set via `OTEL_RESOURCE_ATTRIBUTES` and `OTEL_SERVICE_NAME` environment variables
2. Resource attributes set via annotations (with the `resource.opentelemetry.io/` prefix)
3. Resource attributes set via labels (e.g. `app.kubernetes.io/name`)
   if the `Instrumentation` CR has defaults.useLabelsForResourceAttributes=true (see above)
4. Resource attributes calculated from the pod's metadata (e.g. `k8s.pod.name`)
5. Resource attributes set via the `Instrumentation` CR (in the `spec.resource.resourceAttributes` section)

This priority is applied for each resource attribute separately, so it is possible to set some attributes via
annotations and others via labels.

## How resource attributes are calculated from the pod's metadata

The following resource attributes are calculated from the pod's metadata.

### How `service.name` is calculated

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

### How `service.version` is calculated

Choose the first value found:

- `pod.annotation[resource.opentelemetry.io/service.version]`
- `if (cfg[useLabelsForResourceAttributes]) pod.label[app.kubernetes.io/version]`
- `if (contains(container docker image tag, '/') == false) container docker image tag`

### How `service.instance.id` is calculated

Choose the first value found:

- `pod.annotation[resource.opentelemetry.io/service.instance.id]`
- `concat([k8s.namespace.name, k8s.pod.name, k8s.container.name], '.')`

### How `service.namespace` is calculated

Choose the first value found:

- `pod.annotation[resource.opentelemetry.io/service.namespace]`
- `k8s.namespace.name`
