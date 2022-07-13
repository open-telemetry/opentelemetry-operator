# To build one auto-instrumentation image for dotnet, please:
#  - Download your dotnet auto-instrumentation artefacts to `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
FROM busybox

ARG version

WORKDIR /autoinstrumentation

ADD https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/download/v$version/opentelemetry-dotnet-instrumentation-linux-glibc.zip .

RUN unzip opentelemetry-dotnet-instrumentation-linux-glibc.zip

RUN chmod -R go+r .