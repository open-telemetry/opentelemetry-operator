# Instrumentation base reference

**Status:** *Draft*

**Author:** TBD

**Date:** 2026-09-23

**Issue:** [#5653](https://github.com/open-telemetry/opentelemetry-operator/issues/5653)

## Objective

Let several `Instrumentation` resources share one set of auto-instrumentation settings while each resource keeps its own settings. For example, an operator user should be able to update a Java agent image in one shared resource instead of updating a resource in every workload namespace.

## Summary

Add an optional `spec.baseRef` to `Instrumentation`. The Pod still selects one `Instrumentation` through the existing injection annotation. If that resource has `baseRef`, the operator reads the referenced resource, applies the selected resource's settings over it, and uses the result for that Pod admission request. Neither stored resource is changed by this merge.

The reference may point to the same namespace or another namespace. A referenced resource must not set `baseRef`; this proposal supports one base level. A base update affects Pods created after the update is observed. It does not change existing Pods or start a rollout.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: java-shared
  namespace: observability
spec:
  java:
    image: example.com/java-agent:v2
    extensions:
      - image: example.com/java-plugin:v2
        dir: /extensions
---
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: payments
  namespace: payments
spec:
  baseRef:
    name: java-shared
    namespace: observability
  exporter:
    endpoint: http://payments-collector:4317
```

A Pod in `payments` selects the second resource with `instrumentation.opentelemetry.io/inject-java: "payments"`. The base is not a fallback for an omitted or invalid injection annotation. The existing annotation value `"true"` still requires exactly one `Instrumentation` in the workload namespace. A workload can also select a resource in another namespace directly with `namespace/name`, as it can today; that selects the whole resource and does not merge a local override.

## Goals and non-goals

Goals:

- Share agent images, Java extensions, and SDK settings across namespaced resources.
- Allow a selected resource to override supported settings from the base with predictable precedence.
- Keep current behavior for resources without `baseRef`.
- Apply inheritance during Pod admission without rewriting the selected resource or the base.
- Cover same-namespace and cross-namespace references with unit and end-to-end tests.

Non-goals:

- A chain of references, multiple bases, or a cluster-scoped `Instrumentation` resource.
- Updating running Pods or rolling out workloads when a base changes.
- Arbitrary deep merge semantics for every Kubernetes object, list, and zero value.
- Changing the existing annotation selection rules.
- Reading a Pod's `envFrom` values during admission.

## Use cases for proposal

### Shared Java agent release

A platform team keeps the Java agent image and common plugin in `observability/java-shared`. Teams select their own resources, which reference `java-shared` and set their own exporter endpoints. The platform team changes the image once. Newly created Pods that use these resources receive the new image, unless a selected resource sets its own image.

### Namespace-specific SDK configuration

Two namespaces use the same Java image, propagators, and resource attributes. One namespace sets a different exporter endpoint and environment variable. Its settings win over the corresponding base settings. An explicit environment variable in the application's Pod container still wins over values injected from either resource.

## Struct Design

```go
type InstrumentationSpec struct {
    BaseRef *InstrumentationReference `json:"baseRef,omitempty"`
    // Existing fields remain unchanged.
}

type InstrumentationReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
}
```

`name` is required. An empty `namespace` means the selected resource's namespace. Admission rejects invalid names and direct self-references. The Pod mutation path rejects a base that also has `baseRef`, including references that become invalid after a base is updated. The base must be visible to the operator under its watch configuration.

### Merge rules

| Setting | Proposed rule |
| --- | --- |
| Image and other scalar settings | A non-empty selected value replaces the base value. |
| `spec.resource.resourceAttributes` | Merge by attribute key; the selected key wins. Existing Pod and metadata attribute precedence still applies during injection. |
| Common and language `env` | Merge by variable name. An explicit Pod container variable wins. Within the resources, selected language env, selected common env, selected structured SDK setting, base language env, base common env, and base structured SDK setting are considered in that order. |
| Language resource requests and limits | Merge by resource name; the selected CPU or memory value wins without dropping unrelated base entries. |
| Propagators, Java extensions, and other lists | A non-empty selected list replaces the base list. An omitted or empty selected list inherits it. |
| Exporter | A selected endpoint replaces the base endpoint and TLS settings together. Selected TLS without an endpoint keeps the base endpoint only when that endpoint uses HTTPS. |
| Sampler | A selected type replaces the base type and argument together. An argument without a selected type is invalid. |
| Volume settings and security context | A selected object replaces the corresponding base object. |

The selected resource's structured exporter, sampler, or propagators setting must also take precedence over an equivalent environment variable from the base. For exporter endpoint and TLS settings this includes the signal-specific OTLP variables, which the SDK otherwise gives precedence over generic OTLP variables. Merging Go structs alone would get this case wrong because the existing injector gives environment variables precedence over structured fields. A resource's `metadata`, including its namespace, remains that of the selected resource. The merged spec exists only for injection.

The env order in the table compares matching variable names. It does not override all SDK relationships between different variables. For example, `OTEL_SERVICE_NAME` takes precedence over a `service.name` resource attribute, and the operator may generate a service name from Pod metadata. A child `resourceAttributes.service.name` is therefore not guaranteed to determine the final service name; an explicit child `OTEL_SERVICE_NAME` env is needed to override a base env with that name. A child generic `OTEL_EXPORTER_OTLP_ENDPOINT` env also does not clear inherited exporter TLS settings or override a base signal-specific endpoint. Use the child's structured `spec.exporter.endpoint` to replace the inherited export destination and TLS settings together.

### Defaults and compatibility

The current mutating webhook writes default images, resource requests and limits, and some language settings into a resource. A new resource with `baseRef` must keep omitted fields unset during resource admission; otherwise these stored defaults would hide base values. The base continues to receive its normal defaults. An unset field in the effective configuration follows the existing injection behavior.

Adding `baseRef` to an existing resource does not remove values that were already stored by the webhook. Those image and resource values may continue to override the base. Users can create a new referencing resource or explicitly remove persisted defaults while adding the reference. There is no automatic migration that can reliably tell a user value from a past default. Version conversion must preserve `baseRef` if more than one `Instrumentation` API version is served.

The current CRD serves only `v1alpha1`. The planned `v1beta1` API requires an image in each configured language block, and its conversion omits a language block whose `v1alpha1` image is empty. The conversion preserves `baseRef`, but language overrides that inherit their image do not survive a round-trip through that API. Supporting those overrides in `v1beta1` needs a design coordinated with [#5526](https://github.com/open-telemetry/opentelemetry-operator/pull/5526) before that version is served.

## Rollout Plan

1. Seek Operator SIG feedback on the API version, cross-namespace policy, and merge rules before treating this RFC as accepted. The [RFC process](README.md) calls for an Operator SIG meeting discussion before the RFC is merged.
2. Add the API type, generated CRD schema and deepcopy code, conversion support, admission validation, and defaulting behavior. Keep behavior unchanged when `baseRef` is absent.
3. Resolve the base at Pod admission, merge into an in-memory copy, then run the existing injection path. Preserve the current Pod webhook behavior on a missing or invalid base: admit the Pod without instrumentation and log the error. A Kubernetes Event is not guaranteed.
4. Add table-driven unit tests for defaults, references, each merge rule, conflict precedence, source immutability, and invalid combinations. Add Pod mutation tests showing the selected resource, base, and explicit Pod env precedence.
5. Add chainsaw end-to-end tests for a cross-namespace Java base and a namespace-specific override. Check the injected init container image and Pod environment. Cover a missing base by checking the Operator log and the uninstrumented Pod. Run the repository's chainsaw name check and normal formatting, generation, lint, and precommit gates.
6. Document the migration and new-Pod behavior, add a changelog entry, and release with an example that pins a base image version. Users should roll out workloads when they want all Pods to receive a base update.

## Limitations

- Existing `v1alpha1` non-pointer booleans do not distinguish omitted `false` from an explicit `false`. A selected resource can set such a value to `true`, but cannot turn an inherited `true` into `false` with the current fields.
- The current merge does not treat empty lists as explicit overrides. A non-empty `java.extensions` list replaces the base list; it does not append a team plugin to common plugins. Supporting append, removal, or explicit clear would need a presence-aware API design.
- This proposal does not fetch or expand application `envFrom` sources during Pod admission. Kubernetes gives explicit `container.env` entries precedence over same-name `envFrom` values. A base `OTEL_RESOURCE_ATTRIBUTES` value supplied through `valueFrom` cannot be merged by attribute key with selected resource attributes; this combination fails injection rather than silently choosing a value.
- A base change can cause newly created Pods to differ from running Pods. Teams should use versioned image tags or digests and control workload rollouts for agent and plugin upgrades.
- Secret and ConfigMap names used by exporter TLS settings continue to resolve in the workload namespace, including when the base is elsewhere. A cross-namespace base does not grant a workload access to a Secret in the base namespace.
- A missing base, a base outside the operator's watch scope, or an invalid merge follows the current Pod admission behavior: the Pod is admitted without instrumentation. Operators need to watch the Operator log for the error; a Kubernetes Event is not guaranteed.

## Alternatives considered

- **Use one cross-namespace `Instrumentation` directly.** Workloads can keep some differences in their own Pod environment variables. This does not provide separately managed `Instrumentation` overrides for each namespace.
- **Store common YAML in a ConfigMap.** The ConfigMap schema does not validate `Instrumentation` fields. The operator would need a second parser, validation path, and versioning rules. Pod-native ConfigMap references also cannot supply an agent image or init container configuration.
- **Generate resources with Helm, Kustomize, or another GitOps template.** This can remove source duplication, but it still renders and updates many resources when a shared image changes. It may be sufficient for deployments that already control all resources from one template.
- **Do nothing.** This keeps one simple resource model but leaves teams with repeated resource updates for shared agent releases.

## Open questions for the Operator SIG

1. Should this API enter `v1alpha1` now or be designed with the planned `v1beta1` Instrumentation API?
2. Is the existing ability to select an `Instrumentation` across namespaces sufficient precedent for cross-namespace `baseRef`, or should a base owner explicitly allow references?
3. Are non-empty-list replacement and the `false`/empty-value limits acceptable for an initial version? In particular, should Java extensions support an explicit append mode?
4. Should base resolution errors continue the current admit-without-instrumentation behavior, or should users have a way to require injection?
5. Is skipping defaults for new referencing resources and documenting the existing-resource migration enough, or is a different defaulting model required?
