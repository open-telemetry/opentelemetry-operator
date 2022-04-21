# To build one auto-instrumentation image for dotnet, please:
#  - Download your dotnet auto-instrumentation artefacts to `/autoinstrumentation` directory. This is required as when instrumenting the pod,
#    one init container will be created to copy the files to your app's container.
#  - Grant the necessary access to the files in the `/autoinstrumentation` directory.
FROM busybox

ARG version

WORKDIR /autoinstrumentation

# ADD https://github.com/logicmonitor/lm-telemetry-sdk-dotnet/releases/download/$version/lm-telemetry-dotnet-instrumentation-$version.zip lm-telemetry-dotnet-instrumentation-$version.zip

COPY tracer-home_all-signed.tar.gz .

RUN tar -xvf tracer-home_all-signed.tar.gz

RUN chmod -R go+r bin/