# Sharing Instrumentation configuration

An Instrumentation can use another Instrumentation as a base configuration. This lets teams keep a shared auto-instrumentation image and SDK settings in one resource while setting workload-specific values in their own namespaces.

The workload still selects an Instrumentation with its injection annotation. The operator reads spec.baseRef from that selected resource when the Pod is created. A base resource is not selected as a fallback when an annotation is missing or invalid.

    apiVersion: opentelemetry.io/v1alpha1
    kind: Instrumentation
    metadata:
      name: java-shared
      namespace: observability
    spec:
      java:
        extensions:
          - image: example.com/agents/java-plugin:v2
            dir: /extensions
        image: example.com/agents/java:v2
      exporter:
        endpoint: http://collector.observability:4317
    ---
    apiVersion: opentelemetry.io/v1alpha1
    kind: Instrumentation
    metadata:
      name: team-java
      namespace: team-a
    spec:
      baseRef:
        name: java-shared
        namespace: observability
      exporter:
        endpoint: http://team-collector.team-a:4317
      java:
        env:
          - name: OTEL_INSTRUMENTATION_HTTP_CAPTURE_HEADERS_SERVER_REQUEST
            value: x-request-id

Annotate the workload in team-a with `instrumentation.opentelemetry.io/inject-java: "team-java"`. The base namespace can be omitted when the base resource is in the same namespace as team-java. Each selected resource can reference one base resource; the base cannot reference another resource.

## Precedence

For an ordinary SDK configuration environment variable with the same name, the order is:

1. An explicit environment variable on the Pod container.
2. The selected resource's language-specific env.
3. The selected resource's common env.
4. The base resource's language-specific env.
5. The base resource's common env.
6. Values generated from the effective Instrumentation fields, followed by operator defaults.

The injector also supplies reserved Pod and node metadata variables, such as OTEL_POD_IP and OTEL_NODE_IP. Their existing handling is separate from this order. For Go auto-instrumentation, SDK variables are placed on the agent sidecar rather than the application container.

For a configuration field, a non-empty value in the selected resource overrides the base value. Resource attributes are merged by attribute key, env by variable name, and resource requests and limits by resource name. A non-empty java.extensions list replaces the base list. A new exporter endpoint replaces the base endpoint and TLS settings; setting only exporter TLS keeps the base endpoint when it uses HTTPS. A new sampler type replaces the base sampler and argument. Set sampler.type together with sampler.argument when overriding the argument. Volume settings and security context objects are replaced as units. A selected resource's exporter, sampler, or propagators configuration also takes precedence over equivalent environment variables from the base resource, including signal-specific OTLP endpoint and certificate variables when the exporter is overridden.

The ordinary resource attribute precedence still applies after the resources are combined. In particular, Pod resource annotations and Kubernetes metadata can determine the injected value of an attribute. See [Resource attributes](resource-attributes.md).

The order above compares the same environment variable name. The SDK may give one variable precedence over another. For example, `OTEL_SERVICE_NAME` takes precedence over the `service.name` entry in `OTEL_RESOURCE_ATTRIBUTES`; the operator can also generate `OTEL_SERVICE_NAME` from Pod metadata. A selected `spec.resource.resourceAttributes.service.name` does not always determine the final service name. If you need to override a base `OTEL_SERVICE_NAME`, set `OTEL_SERVICE_NAME` in the selected resource's env or on the Pod. Similarly, a signal-specific OTLP endpoint can take precedence over a generic `OTEL_EXPORTER_OTLP_ENDPOINT`. To replace an inherited export destination, set the selected resource's `spec.exporter.endpoint` rather than only the generic environment variable.

## Operational notes

- The current CRD serves only `v1alpha1`. The planned `v1beta1` API requires an image for each configured language, and its conversion omits language overrides without an image. Use `v1alpha1` for overrides that inherit their language image.
- Base resources are read when a new Pod is admitted. Updating the base changes newly created Pods; it does not change running Pods or trigger a rollout.
- The base must be visible to the operator's watched namespaces. If it is missing or cannot be read, the Pod admission webhook permits the Pod without instrumentation and logs the injection error. Check operator logs; a Kubernetes Event is not guaranteed.
- Secrets and ConfigMaps referenced by exporter TLS settings must exist in the **workload namespace**, even when the base is in another namespace.
- An annotation value of "true" still requires exactly one Instrumentation in the workload namespace. Use the selected resource's name when that namespace has more than one.
- Create referencing resources without existing webhook-populated defaults. Adding baseRef to a resource already created without it leaves persisted default images and resources on that resource; those values may mask the base until removed.
- The current merge treats false and empty lists as unset, so a selected resource cannot turn an inherited true into false or clear an inherited list with []. The non-pointer boolean fields and lists tagged with `omitempty` do not provide a reliable explicit-value signal across API round-trips. Use a separate resource without that inherited setting when clearing is required.
- The operator does not fetch or expand application `envFrom` sources during admission. Kubernetes gives explicit `env` entries precedence over same-name `envFrom` values, so an injected explicit variable can override a value supplied only through `envFrom`. If the selected resource sets resource attributes, inherited OTEL_RESOURCE_ATTRIBUTES text is merged by comma-separated key=value entries. An inherited valueFrom reference or an entry the operator cannot parse causes injection to be skipped.
- When overriding an inherited exporter endpoint through `spec.env` instead of `spec.exporter.endpoint`, inherited exporter TLS settings remain in effect and may still require a Secret or ConfigMap in the workload namespace.
