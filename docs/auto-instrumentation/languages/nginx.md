# Nginx auto-instrumentation

```bash
instrumentation.opentelemetry.io/inject-nginx: "true"
```

## Using Nginx autoinstrumentation

For `Nginx` autoinstrumentation, Nginx versions 1.22.0, 1.23.0, and 1.23.1 are supported at this time. The Nginx configuration file is expected to be `/etc/nginx/nginx.conf` by default, if it's different, see following example on how to change it. Instrumentation at this time also expects, that `conf.d` directory is present in the directory, where configuration file resides and that there is a `include <config-file-dir-path>/conf.d/*.conf;` directive in the `http { ... }` section of Nginx configuration file (like it is in the default configuration file of Nginx). You can also adjust OpenTelemetry SDK attributes. Example:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: my-instrumentation
spec:
  nginx:
    image: your-customized-auto-instrumentation-image:nginx # if custom instrumentation image is needed
    configFile: /my/custom-dir/custom-nginx.conf
    attrs:
      - name: NginxModuleOtelMaxQueueSize
        value: "4096"
      - name: ...
        value: ...
```

List of all available attributes can be found at [otel-webserver-module](https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module)
