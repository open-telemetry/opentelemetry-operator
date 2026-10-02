# Java auto-instrumentation

```bash
instrumentation.opentelemetry.io/inject-java: "true"
```

## Java agent extensions

Use `spec.java.extensions` in the `Instrumentation` resource to add extension JARs
to the Java agent. Each entry specifies an `image` containing the JARs and a `dir`
pointing to their directory inside that image. The directory is not a path in the
application container.

For example, if the image contains `/extensions/my-extension.jar`, configure:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: java-instrumentation
spec:
  java:
    extensions:
      - image: registry.example.com/java-agent-extensions:1.0
        dir: /extensions
```

The Operator creates an init container for each extension image and runs `cp -r`
to copy the contents of `dir` into the shared Java instrumentation volume. The
extension image must provide the `cp` command. The Operator adds
`-Dotel.javaagent.extensions=<instrumentation-directory>/extensions` to the
application container's `JAVA_TOOL_OPTIONS` so the Java agent loads the JARs.

All extension images copy into the same directory. Use distinct JAR filenames to
avoid overwriting extensions from another image.
