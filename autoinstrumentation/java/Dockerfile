# To build one auto-instrumentation image for Java, please:
#  - Download your customized `javaagent.jar` to `/javaagent.jar`. This is required as when instrumenting the pod,
#    one init container will be created to copy the jar to your app's container.
#  - Grant the necessary access to the jar. `chmod -R go+r /javaagent.jar`
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be availabe in the image.

# TODO: Remove dev image reference
FROM jvsplk/opentelemetry-operator-utils:v0.0.1 AS utils
ARG version
ARG release_url=https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v${version}/opentelemetry-javaagent.jar
WORKDIR /utils/
ADD $release_url  javaagent.jar
RUN chmod 644 -R javaagent.jar

FROM scratch AS final
COPY --from=utils /utils/ /
ENV PATH="${PATH}:/bin"