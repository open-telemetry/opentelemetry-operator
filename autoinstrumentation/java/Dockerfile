# To build one auto-instrumentation image for Java, please:
#  - Download your customized `javaagent.jar` to `/javaagent.jar`. This is required as when instrumenting the pod,
#    one init container will be created to copy the jar to your app's container.
#  - Grant the necessary access to the jar. `chmod -R go+r /javaagent.jar`
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be available in the image.
ARG TARGETARCH
ARG injector_version=0.9.2

FROM busybox@sha256:fd8d9aa63ba2f0982b5304e1ee8d3b90a210bc1ffb5314d980eb6962f1a9715d AS base

ARG version

ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$version/opentelemetry-javaagent.jar /javaagent.jar

RUN chmod -R go+r /javaagent.jar

# The opentelemetry-injector shared object activates the agent via LD_PRELOAD
# when the operator.autoinstrumentation.injector feature gate is enabled.
# It is only published for amd64 and arm64, so the other architectures this
# image is built for (s390x, ppc64le) ship without it.
FROM base AS injector-amd64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_amd64.so /libotelinject.so
RUN chmod go+r /libotelinject.so

FROM base AS injector-arm64
ARG injector_version
ADD https://github.com/open-telemetry/opentelemetry-injector/releases/download/v$injector_version/libotelinject_arm64.so /libotelinject.so
RUN chmod go+r /libotelinject.so

FROM base AS injector-s390x

FROM base AS injector-ppc64le

FROM injector-$TARGETARCH
