# To build one auto-instrumentation image for dotnet, please:
#  - Download your dotnet auto-instrumentation artifacts to the `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
#  - Following environment variables are injected to the application container to enable the .NET auto-instrumentation.
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
FROM busybox:1.38.0@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d AS downloader

ARG version

WORKDIR /autoinstrumentation

ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v$version/opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip .
ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v$version/opentelemetry-dotnet-instrumentation-linux-musl-x64.zip .

RUN unzip opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip &&\
    unzip opentelemetry-dotnet-instrumentation-linux-musl-x64.zip "linux-musl-x64/*" -d . &&\
    rm opentelemetry-dotnet-instrumentation-linux-glibc-x64.zip opentelemetry-dotnet-instrumentation-linux-musl-x64.zip &&\
    chmod -R go+r .

# The opentelemetry-injector expects separate glibc and musl installation
# roots (<prefix>/glibc, <prefix>/musl), while this image ships a single
# merged installation with both native profiler builds next to the shared
# managed assemblies. Map the injector layout onto the merged one with
# symlinked directories.
RUN mkdir .glibc .musl &&\
    for f in *; do ln -s ../$f .glibc/$f && ln -s ../$f .musl/$f; done &&\
    mv .glibc glibc && mv .musl musl

FROM busybox:1.38.0@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d

COPY --from=downloader /autoinstrumentation /autoinstrumentation

# The opentelemetry-injector shared object activates the instrumentation via
# LD_PRELOAD when the operator.autoinstrumentation.injector feature gate is
# enabled. It is placed inside /autoinstrumentation so the init container
# copies it together with the instrumentation itself.
ARG TARGETARCH
# Version of the opentelemetry-injector release to include, without the "v"
# prefix. There is no default; the canonical value lives in
# autoinstrumentation/injector_version.txt and must be passed in as a build-arg:
#   --build-arg injector_version=$(cat ../injector_version.txt)
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_${TARGETARCH}.so /autoinstrumentation/libotelinject.so
RUN chmod go+r /autoinstrumentation/libotelinject.so
