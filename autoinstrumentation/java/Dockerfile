FROM busybox

ARG version

ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$version/opentelemetry-javaagent.jar /autoinstrumentation/javaagent.jar

RUN chmod -R o+r /autoinstrumentation
