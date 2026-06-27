# Apache HTTPD auto-instrumentation

```bash
instrumentation.opentelemetry.io/inject-apache-httpd: "true"
```

For `Apache HTTPD` autoinstrumentation, by default, instrumentation assumes httpd version 2.4 and httpd configuration directory `/usr/local/apache2/conf` as it is in the official `Apache HTTPD` image (f.e. docker.io/httpd:latest). If you need to use version 2.2, or your HTTPD configuration directory is different, and or you need to adjust agent attributes, customize the instrumentation specification per following example:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  apacheHttpd:
    image: your-customized-auto-instrumentation-image:apache-httpd
    version: "2.2"
    configPath: /your-custom-config-path
    attrs:
      - name: ApacheModuleOtelMaxQueueSize
        value: "4096"
      - name: ...
        value: ...
```

List of all available attributes can be found at [otel-webserver-module](https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module)
