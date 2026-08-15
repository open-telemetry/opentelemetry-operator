# Auto-instrumentation e2e tests (Go)

Full-pipeline auto-instrumentation tests, built on `internal/testing/e2e`: the
operator injects a language agent into a sample app, and the telemetry the app
then produces must arrive at a backend with the right identity.

## How it works

Each test gets its own randomly-named namespace containing:

1. an **otlp-sink** (`tests/test-e2e-apps/otlp-sink`) — an OTLP backend that
   records every export request and serves it back as OTLP-JSON,
2. an **OpenTelemetryCollector** (`deployment-collector`) that forwards all
   received OTLP to the sink,
3. an **Instrumentation** CR and an annotated **sample app** Deployment.

The test then drives the app over the API server's pod proxy and asserts on the
telemetry the sink received — for example, that a `SERVER` span arrived whose
resource carries `service.name=my-java`, `k8s.namespace.name=<test ns>` and
`k8s.pod.name`, or that a `jvm.memory.used` metric arrived. Asserting on the
decoded telemetry checks the injected resource identity end-to-end, rather than
matching a substring somewhere in an exporter's output.

Where a telemetry name depends on the agent's semconv version (e.g.
`process.runtime.jvm.memory.usage` vs `jvm.memory.used`), the expectation
accepts any of the known names, so an agent update does not break the suite.

## What to assert (stability policy)

The operator owns the default instrumentation images, so asserting on what
they emit in this simplest-possible setup is in scope — but assertions must
not break on every agent update because experimental telemetry churned. The
balance:

- **Assert exactly, always: the operator-injected resource identity.**
  `service.name`, `service.namespace`, `service.instance.id`,
  `service.version` and the `k8s.*` attributes are wired by the operator via
  `OTEL_SERVICE_NAME`/`OTEL_RESOURCE_ATTRIBUTES`; the runner derives their
  expected values from live cluster state (`resourceIdentity`) and requires
  them on every asserted span and metric. Regressions here are operator bugs
  by definition.
- **Assert values only through stable semantic conventions.** HTTP
  method/status/path on server spans, metric type and unit — expressed as
  `AttrMatch` alternatives so the same value is asserted through whichever
  attribute name the agent's semconv version uses.
- **Assert presence only for experimental telemetry.** Experimental metric
  names use `NamesAnyOf` with all known names; add the new name to the list
  when an agent renames one, don't chase values.
- **Never assert** on durations, exact counts (beyond non-zero), scope
  versions, `telemetry.sdk.version`/`telemetry.distro.*` values, or agents'
  internal self-monitoring telemetry.

## Diagnostics

Failure output is designed to be sufficient on its own:

- sink assertion timeouts report an inventory of what **was** received
  (span/metric counts, `service.name` values, names seen);
- `e2e.DumpNamespaceOnFailure` logs pod and container statuses, namespace
  events, the trailing logs of every container (app, agent init containers,
  collector, sink) plus the operator's log, and the applied CRs with status;
- injection is verified on the pod spec before waiting for telemetry, so a
  webhook failure surfaces as a precise error instead of a timeout.

## Concurrency

Tests call `t.Parallel()` and share nothing: all resources are namespaced and
each namespace has its own collector and sink. Use `go test -parallel N` (or
`GOTESTSUM` via `make e2e-instrumentation-go`) to tune concurrency.

## Running

```bash
make prepare-e2e            # kind cluster + operator + images
make e2e-instrumentation-go
```

The sample apps and the otlp-sink are used as published from `main`
(`ghcr.io/open-telemetry/opentelemetry-operator/e2e-test-app-*`). After
changing anything under `tests/test-e2e-apps`, build and load local versions
with `make load-image-test-e2e-apps` — CI does this automatically when those
files change (see `tests/test-e2e-apps/README.md`).

The sample app images are named in the testdata manifests; the sink is deployed
by the framework instead, so its image comes from `OTLPSINK_IMG`, which the make
target derives from `TEST_E2E_APPS_IMG_PREFIX`. Running `go test` directly
requires setting it.

## Out of scope

This suite covers scenarios where telemetry has to reach a backend. It does not
cover:

- assertions purely about the injected pod spec — init container variants,
  SDK-only injection, the Python spec env, and multi-instrumentation container
  selection all resolve on the pod spec and need no telemetry;
- mTLS between the app and the collector, the multi-container re-injection
  variants, Go auto-instrumentation, and the no-CRDs case.
