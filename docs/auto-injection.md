# Centralized Auto-Injection Configuration

## Overview

The OpenTelemetry Operator supports centralized auto-injection configuration, allowing administrators to configure automatic instrumentation for multiple services without adding annotations to individual deployments.

## Configuration

Configure auto-injection in the `Instrumentation` resource using the `autoInjection` field:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  exporter:
    endpoint: http://otel-collector:4317
  propagators:
    - tracecontext
    - baggage
  sampler:
    type: parentbased_traceidratio
    argument: "1.0"
  autoInjection:
    enabled: true
    targetServices:
      - example1-java-service                         # Default Java
      - jvm:example2-java-service                     # Prefix for Java
      - go:example1-go-service                        # Prefix for Go with default executable
      - go:example2-go-service:/bin/execute           # Prefix for Go with custom executable
      - node:example-node-service                     # Prefix for NodeJS
      - py:example-python-service                     # Prefix for Python
      - dotnet:example-dotnet-service                 # Prefix for .NET
```

## Service Name Format

Services can be specified with optional language prefixes and executable paths:

### Basic Format
```
servicename
```

### With Language Prefix
```
language:servicename
```

### With Executable Path (Go only)
```
go:servicename:/path/to/executable
```

## Language Prefixes

| Prefix | Language | Example |
|--------|----------|---------|
| (none) | Java (default) | `payment-service` |
| `java` or `jvm` | Java | `jvm:payment-service` |
| `node` | NodeJS | `node:frontend` |
| `py` | Python | `py:analytics` |
| `dotnet` | .NET | `dotnet:api` |
| `go` | Go | `go:worker` or `go:worker:/app/binary` |

### Go Services

For Go services, you can optionally specify the executable path after the service name:

```yaml
targetServices:
  - go:worker                          # Uses default executable detection
  - go:worker:/app/myapp               # Custom executable path
  - go:processor:/usr/local/bin/proc   # Custom executable path
```

## Examples

### Basic Configuration

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: auto-inject
spec:
  exporter:
    endpoint: http://otel-collector:4317
  autoInjection:
    enabled: true
    targetServices:
      - myapp
      - frontend
```

### Multi-Language Configuration

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: multi-lang-inject
spec:
  exporter:
    endpoint: http://otel-collector:4317
  autoInjection:
    enabled: true
    targetServices:
      - payment                    # Java (default)
      - jvm:order-service          # Java (explicit)
      - node:frontend              # NodeJS
      - py:analytics               # Python
      - dotnet:api                 # .NET
```

### Go Service Configuration

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: go-inject
spec:
  exporter:
    endpoint: http://otel-collector:4317
  autoInjection:
    enabled: true
    targetServices:
      - go:worker                    # Default executable
      - go:api:/usr/local/bin/api    # Custom executable
```

### Mixed Configuration

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: mixed-inject
spec:
  exporter:
    endpoint: http://otel-collector:4317
  autoInjection:
    enabled: true
    targetServices:
      - payment-service              # Java (default)
      - jvm:order-service            # Java (explicit)
      - node:frontend                # NodeJS
      - py:analytics                 # Python
      - dotnet:api                   # .NET
      - go:worker:/app/worker        # Go with custom executable
```

## How It Works

1. The operator watches for pod creation events in the same namespace as the Instrumentation resource
2. When a pod is created, it extracts the service name from pod labels (`app` or `app.kubernetes.io/name`)
3. If the service name matches a target in `autoInjection.targetServices`, instrumentation is injected
4. The language prefix (or default Java) determines which instrumentation to inject
5. For Go services, the executable path is extracted and set as `OTEL_GO_AUTO_TARGET_EXE`

## Service Name Detection

The operator detects service names from pod labels in this priority order:

1. `app` label
2. `app.kubernetes.io/name` label
3. Pod name (if no labels found)

Example pod that will match service name "myapp":

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp-pod-12345
  labels:
    app: myapp
spec:
  containers:
  - name: app
    image: myapp:latest
```

## Benefits

- **Centralized Management**: Configure instrumentation for multiple services in one place
- **No Deployment Changes**: No need to add annotations to individual deployments
- **Namespace Scoped**: Instrumentation applies only to pods in the same namespace
- **Language Flexibility**: Support for Java, NodeJS, Python, .NET, and Go
- **Simple Configuration**: Flat list of service names instead of nested structures

## Migration from Annotation-Based Injection

If you're currently using annotation-based injection:

```yaml
# Old approach - annotation on each deployment
annotations:
  instrumentation.opentelemetry.io/inject-java: "true"
```

You can migrate to centralized configuration:

```yaml
# New approach - centralized configuration
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
  namespace: default
spec:
  autoInjection:
    enabled: true
    targetServices:
      - myapp
```

Both approaches can coexist. Annotation-based injection takes precedence over auto-injection.

## Important Notes

- Auto-injection only works for pods in the **same namespace** as the Instrumentation resource
- If you need to instrument services in multiple namespaces, create separate Instrumentation resources in each namespace
- Pods with annotation `instrumentation.opentelemetry.io/auto-injected: "true"` will be skipped to prevent duplicate injection
- Language-specific instrumentation must be enabled in the operator configuration (e.g., `--enable-java-instrumentation=true`)
