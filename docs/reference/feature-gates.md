# Controlling Instrumentation Capabilities

The operator allows specifying, via the flags, which languages the Instrumentation resource may instrument.
If a language is enabled by default its gate only needs to be supplied when disabling the gate.
Language support can be disabled by passing the flag with a value of `false`.

| Language    | Gate                                  | Default Value |
| ----------- | ------------------------------------- | ------------- |
| Java        | `enable-java-instrumentation`         | `true`        |
| NodeJS      | `enable-nodejs-instrumentation`       | `true`        |
| Python      | `enable-python-instrumentation`       | `true`        |
| DotNet      | `enable-dotnet-instrumentation`       | `true`        |
| ApacheHttpD | `enable-apache-httpd-instrumentation` | `true`        |
| Go          | `enable-go-instrumentation`           | `false`       |
| Nginx       | `enable-nginx-instrumentation`        | `false`       |


OpenTelemetry Operator allows to instrument multiple containers using multiple language specific instrumentations.
These features can be enabled using the `enable-multi-instrumentation` flag. By default flag is `false`.

For more information about multi-instrumentation feature capabilities please see [Multi-container pods with multiple instrumentations](../auto-instrumentation/multi-instrumentation.md).
