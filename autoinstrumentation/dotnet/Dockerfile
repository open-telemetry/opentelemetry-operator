# To build one auto-instrumentation image for dotnet, please:
#  - Download your dotnet auto-instrumentation artifacts to the `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
#  - Following environment variables are injected to the application container to enable the .NET auto-instrumentation.
#    CORECLR_ENABLE_PROFILING=1
#    CORECLR_PROFILER={918728DD-259F-4A6A-AC2B-B85E1B658318}
#    CORECLR_PROFILER_PATH=%InstallationLocation%/linux/OpenTelemetry.AutoInstrumentation.Native.so # for glibc based images
#    CORECLR_PROFILER_PATH=%InstallationLocation%/linux-musl/OpenTelemetry.AutoInstrumentation.Native.so # for musl based images
#    DOTNET_ADDITIONAL_DEPS=%InstallationLocation%/AdditionalDeps
#    DOTNET_SHARED_STORE=%InstallationLocation%/store
#    DOTNET_STARTUP_HOOKS=%InstallationLocation%/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll
#    OTEL_DOTNET_AUTO_HOME=%InstallationLocation%
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be available in the image.
#  - This image is built for linux/amd64 and linux/arm64. The .NET Runtime Identifier (RID) specific
#    artifact for the target platform is downloaded and exposed under the architecture-agnostic
#    `linux`/`linux-musl` directories, with `linux-x64`/`linux-musl-x64` kept as symlinks for
#    backward compatibility with the CORECLR_PROFILER_PATH values used before multi-arch support
#    was added.

FROM busybox:1.38.0@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d AS downloader

ARG TARGETARCH
ARG version

WORKDIR /autoinstrumentation

RUN case "${TARGETARCH}" in \
      amd64) dotnetArch=x64 ;; \
      arm64) dotnetArch=arm64 ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    wget -q -O glibc.zip "https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v${version}/opentelemetry-dotnet-instrumentation-linux-glibc-${dotnetArch}.zip" && \
    wget -q -O musl.zip "https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v${version}/opentelemetry-dotnet-instrumentation-linux-musl-${dotnetArch}.zip" && \
    unzip glibc.zip && \
    unzip musl.zip "linux-musl-${dotnetArch}/*" -d . && \
    mv "linux-${dotnetArch}" linux && \
    mv "linux-musl-${dotnetArch}" linux-musl && \
    ln -s linux "linux-${dotnetArch}" && \
    ln -s linux-musl "linux-musl-${dotnetArch}" && \
    rm glibc.zip musl.zip && \
    chmod -R go+r .

FROM busybox:1.38.0@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d


COPY --from=downloader /autoinstrumentation /autoinstrumentation
