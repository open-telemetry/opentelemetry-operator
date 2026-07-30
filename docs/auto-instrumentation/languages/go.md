# Go auto-instrumentation

Go auto-instrumentation also honors an annotation that will be used to set the [OTEL_GO_AUTO_TARGET_EXE env var](https://github.com/open-telemetry/opentelemetry-go-instrumentation/blob/main/docs/how-it-works.md).
This env var can also be set via the Instrumentation resource, with the annotation taking precedence.
Since Go auto-instrumentation requires `OTEL_GO_AUTO_TARGET_EXE` to be set, you must supply a valid
executable path via the annotation or the Instrumentation resource. Failure to set this value causes instrumentation injection to abort, leaving the original pod unchanged.

```bash
instrumentation.opentelemetry.io/inject-go: "true"
instrumentation.opentelemetry.io/otel-go-auto-target-exe: "/path/to/container/executable"
```

Go auto-instrumentation also requires elevated permissions. The below permissions are set automatically and are required.

```yaml
securityContext:
  privileged: true
  runAsUser: 0
```
