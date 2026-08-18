# Multi-container pods with multiple instrumentations

Works only when `enable-multi-instrumentation` flag is `true`.

Annotations defining which language instrumentation will be injected are required. When feature is enabled, specific for Instrumentation language containers annotations are used (these also support init container names for Java, Python, Node.js, .NET, and SDK):

Java:

```bash
instrumentation.opentelemetry.io/java-container-names: "java1,java2"
```

NodeJS:

```bash
instrumentation.opentelemetry.io/nodejs-container-names: "nodejs1,nodejs2"
```

Python:

```bash
instrumentation.opentelemetry.io/python-container-names: "python1,python3"
```

DotNet:

```bash
instrumentation.opentelemetry.io/dotnet-container-names: "dotnet1,dotnet2"
```

Go:

```bash
instrumentation.opentelemetry.io/go-container-names: "go1"
```

ApacheHttpD:

```bash
instrumentation.opentelemetry.io/apache-httpd-container-names: "apache1,apache2"
```

NGINX:

```bash
instrumentation.opentelemetry.io/inject-nginx-container-names: "nginx1,nginx2"
```

SDK:

```bash
instrumentation.opentelemetry.io/sdk-container-names: "app1,app2"
```

If language instrumentation specific container names are not specified, instrumentation is performed on the first regular container available in the pod spec (only if single instrumentation injection is configured).

In some cases containers in the pod are using different technologies. It becomes necessary to specify language instrumentation for container(s) on which this injection must be performed.

For this, we will use language instrumentation specific container names annotation for which we will indicate one or more container names (`.spec.containers.name` or `.spec.initContainers.name`) on which the injection must be made:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment-with-multi-containers-multi-instrumentations
spec:
  selector:
    matchLabels:
      app: my-pod-with-multi-containers-multi-instrumentations
  replicas: 1
  template:
    metadata:
      labels:
        app: my-pod-with-multi-containers-multi-instrumentations
      annotations:
        instrumentation.opentelemetry.io/inject-java: "true"
        instrumentation.opentelemetry.io/java-container-names: "myapp,myapp2"
        instrumentation.opentelemetry.io/inject-python: "true"
        instrumentation.opentelemetry.io/python-container-names: "myapp3"
    spec:
      containers:
        - name: myapp
          image: myImage1
        - name: myapp2
          image: myImage2
        - name: myapp3
          image: myImage3
```

In the above case, `myapp` and `myapp2` containers will be instrumented using Java and `myapp3` using Python instrumentation.

**NOTE**: Go auto-instrumentation **does not** support multicontainer pods. When injecting Go auto-instrumentation the first container should be the only you want to instrument.

**NOTE**: This type of instrumentation **does not** allow to instrument a container with multiple language instrumentations.

**NOTE**: `instrumentation.opentelemetry.io/container-names` annotation is not used for this feature.
