# Multi-container pods with single instrumentation

If nothing else is specified, instrumentation is performed on the first container available in the pod spec (from `.spec.containers`, not init containers).
In some cases (for example in the case of the injection of an Istio sidecar) it becomes necessary to specify on which container(s) this injection must be performed.

For this, it is possible to fine-tune the pod(s) on which the injection will be carried out.

For this, we will use the `instrumentation.opentelemetry.io/container-names` annotation for which we will indicate one or more container names (from `.spec.containers.name` or `.spec.initContainers.name`) on which the injection must be made:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment-with-multiple-containers
spec:
  selector:
    matchLabels:
      app: my-pod-with-multiple-containers
  replicas: 1
  template:
    metadata:
      labels:
        app: my-pod-with-multiple-containers
      annotations:
        instrumentation.opentelemetry.io/inject-java: "true"
        instrumentation.opentelemetry.io/container-names: "myapp,myapp2"
    spec:
      containers:
        - name: myapp
          image: myImage1
        - name: myapp2
          image: myImage2
        - name: myapp3
          image: myImage3
```

In the above case, `myapp` and `myapp2` containers will be instrumented, `myapp3` will not.

> 🚨 **NOTE**: Go auto-instrumentation **does not** support multicontainer pods. When injecting Go auto-instrumentation the first container should be the only container you want instrumented.
