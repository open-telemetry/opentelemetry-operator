# To build one auto-instrumentation image for dotnet, please:
#  - Download your dotnet auto-instrumentation artefacts to `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
#  - Following environment variables are injected to the application container to enable the auto-instrumentation.
#    CORECLR_ENABLE_PROFILING=1
#    CORECLR_PROFILER={918728DD-259F-4A6A-AC2B-B85E1B658318}
#    CORECLR_PROFILER_PATH=%InstallationLocation%/linux-x64/OpenTelemetry.AutoInstrumentation.Native.so # for glibc based images
#    CORECLR_PROFILER_PATH=%InstallationLocation%/linux-musl-x64/OpenTelemetry.AutoInstrumentation.Native.so # for musl based images
#    DOTNET_ADDITIONAL_DEPS=%InstallationLocation%/AdditionalDeps
#    DOTNET_SHARED_STORE=%InstallationLocation%/store
#    DOTNET_STARTUP_HOOKS=%InstallationLocation%/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll 
#    OTEL_DOTNET_AUTO_HOME=%InstallationLocation%
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be available in the image.

FROM busybox AS downloader

ARG version

WORKDIR /autoinstrumentation

ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v$version/opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip .
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v$version/opentelemetry-dotnet-instrumentation-linux-musl-x64.zip .

RUN unzip opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip &&\
    unzip opentelemetry-dotnet-instrumentation-linux-musl-x64.zip "linux-musl-x64/*" -d . &&\
    rm opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip opentelemetry-dotnet-instrumentation-linux-musl-x64.zip &&\
    chmod -R go+r .

FROM busybox

COPY --from=downloader /autoinstrumentation /autoinstrumentation
