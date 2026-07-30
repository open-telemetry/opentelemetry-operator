# Instrumenting Init Containers

Init containers can be instrumented by including their names in the `container-names` annotation. When an init container is targeted for instrumentation, the operator automatically inserts the instrumentation init container **before** the target init container in the pod's init container sequence. This ensures the instrumentation agent files are available when the target init container runs.

Supported instrumentations for init containers:
- Java
- Python
- Node.js
- .NET
- SDK-only injection

**Not supported** for init containers:
- Go (does not support multicontainer pods)
- Apache HTTPD
- Nginx

> **Note**: Kubernetes guarantees that container names are unique across both the `initContainers` and `containers` lists within a pod spec. This allows the operator to unambiguously identify whether a container name refers to an init container or a regular container.

Example with both init container and regular container instrumentation:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment-with-init-container
spec:
  selector:
    matchLabels:
      app: my-app
  replicas: 1
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        instrumentation.opentelemetry.io/inject-python: "true"
        instrumentation.opentelemetry.io/container-names: "my-init-job,myapp"
    spec:
      initContainers:
        - name: my-init-job
          image: my-python-init-image
      containers:
        - name: myapp
          image: my-python-app-image
```

In this example, both `my-init-job` (an init container) and `myapp` (a regular container) will be instrumented with Python auto-instrumentation.
