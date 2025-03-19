### Injector + Instrumentations image
#
# The injector image container both the LD_PRELOAD object, as well as
# all the supported instrumentations (as it needs to copy them into
# the emptyDir volume)

# Java agent
FROM busybox AS java-agent

ARG javaagentversion

ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v$javaagentversion/opentelemetry-javaagent.jar /javaagent.jar

RUN chmod -R go+r /javaagent.jar

# build injector
FROM ubuntu:24.04 AS build-injector

RUN apt-get update && apt-get install --no-install-recommends build-essential -y

COPY ./src /injector/src
WORKDIR /injector
RUN gcc \
  -shared \
  -nostdlib \
  -fPIC \
  -Wl,--version-script=src/injector.exports.map \
  /injector/src/injector.c \
  -o /injector/injector.so

# injector_build_end - do not remove this line (see images/instrumentation/injector/test/scripts/test-all.sh)

# build final image
FROM busybox
WORKDIR /

COPY copy-instrumentation.sh /

COPY --from=build-injector /injector/injector.so /instrumentation/injector.so
COPY --from=java-agent /javaagent.jar /instrumentation/jvm/javaagent.jar

CMD ["/copy-instrumentation.sh"]