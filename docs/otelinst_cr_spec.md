# Instrumentation Custom Resource Specification

The below `Instrumentation` custom resource contains all the specification that can be configured.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: example-instrumentation
spec:
  // +optional Exporter defines exporter configuration.
  exporter:
    // +optional Endpoint is address of the collector with OTLP endpoint.
    endpoint: 

  // +optional Java defines configuration for java auto-instrumentation.
  java: 
    // +optional Image is a container image with javaagent JAR.
    image:
```
