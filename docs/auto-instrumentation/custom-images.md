# Use customized or vendor instrumentation

By default, the operator uses upstream auto-instrumentation libraries. Custom auto-instrumentation can be configured by
overriding the `image` fields in a CR.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  java:
    image: your-customized-auto-instrumentation-image:java
  nodejs:
    image: your-customized-auto-instrumentation-image:nodejs
  python:
    image: your-customized-auto-instrumentation-image:python
  dotnet:
    image: your-customized-auto-instrumentation-image:dotnet
  go:
    image: your-customized-auto-instrumentation-image:go
  apacheHttpd:
    image: your-customized-auto-instrumentation-image:apache-httpd
  nginx:
    image: your-customized-auto-instrumentation-image:nginx
```

The Dockerfiles for auto-instrumentation can be found in [autoinstrumentation directory](../../autoinstrumentation).
Follow the instructions in the Dockerfiles on how to build a custom container image.
