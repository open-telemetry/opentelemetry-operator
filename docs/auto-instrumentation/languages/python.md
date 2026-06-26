# Python auto-instrumentation

Python auto-instrumentation also honors an annotation that will permit it to run it on images with a different C library than glibc.

```bash
instrumentation.opentelemetry.io/inject-python: "true"
instrumentation.opentelemetry.io/otel-python-platform: "glibc" # for Linux glibc based images, this is the default value and can be omitted
instrumentation.opentelemetry.io/otel-python-platform: "musl" # for Linux musl based images
```
