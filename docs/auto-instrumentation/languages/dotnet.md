# .NET auto-instrumentation

.NET auto-instrumentation also honors an annotation that will be used to set the .NET [Runtime Identifiers](https://learn.microsoft.com/en-us/dotnet/core/rid-catalog)(RIDs).
Currently, only two RIDs are supported: `linux-x64` and `linux-musl-x64`.
By default `linux-x64` is used.

```bash
instrumentation.opentelemetry.io/inject-dotnet: "true"
instrumentation.opentelemetry.io/otel-dotnet-auto-runtime: "linux-x64" # for Linux glibc based images, this is default value and can be omitted
instrumentation.opentelemetry.io/otel-dotnet-auto-runtime: "linux-musl-x64"  # for Linux musl based images
```

> **Note:** The auto-instrumentation image is published for both `linux/amd64` and `linux/arm64`. The runtime annotation above only distinguishes glibc vs musl based application images; the CPU architecture of the actual native profiler binary used is resolved automatically when the image is pulled onto the node, so no separate arm64 annotation values are needed.

> **Note:** For `DotNet` auto-instrumentation, by default, operator sets the `OTEL_DOTNET_AUTO_TRACES_ENABLED_INSTRUMENTATIONS` environment variable which specifies the list of traces source instrumentations you want to enable. The value that is set by default by the operator is all available instrumentations supported by the `openTelemery-dotnet-instrumentation` release consumed in the image, i.e. `AspNet,HttpClient,SqlClient`. This value can be overridden by configuring the environment variable explicitly.
