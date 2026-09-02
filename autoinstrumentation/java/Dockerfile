# To build one auto-instrumentation image for Java, please:
#  - Download your customized `javaagent.jar` to `/javaagent.jar`. This is required as when instrumenting the pod,
#    one init container will be created to copy the jar to your app's container.
#  - Grant the necessary access to the jar. `chmod -R go+r /javaagent.jar`
#  - For auto-instrumentation by container injection, the Linux command cp is
#    used and must be available in the image.
FROM busybox@sha256:dc2d74b28e4cf8984fa52af1f39bc7c3d9c73760b41a74d629f5d11b1ab28616

ARG version

ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$version/opentelemetry-javaagent.jar /javaagent.jar

RUN chmod -R go+r /javaagent.jar
