# To build one auto-instrumentation image for Java, please:
#  - Download your customized `javaagent.jar` to `/javaagent.jar`. This is required as when instrumenting the pod,
#    one init container will be created to copy the jar to your app's container.
#  - Grant the necessary access to the jar. `chmod -R go+r /javaagent.jar`
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be availabe in the image.
FROM busybox as downloader

ARG version

ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$version/opentelemetry-javaagent.jar /javaagent.jar

RUN chmod -R go+r /javaagent.jar

FROM ghcr.io/open-telemetry/opentelemetry-operator/cp:1.0.0 AS cp

FROM scratch

COPY --from=cp /cp /usr/bin/cp

COPY --from=downloader /javaagent.jar /javaagent.jar
