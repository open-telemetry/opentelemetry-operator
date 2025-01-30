FROM alpine:latest as agent

ARG version

RUN mkdir /opt/opentelemetry
WORKDIR /opt/opentelemetry

RUN mkdir agent
RUN wget -c https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/download/webserver%2Fv$version/opentelemetry-webserver-sdk-x64-linux.tgz
RUN unzip -p opentelemetry-webserver-sdk-x64-linux.tgz | tar -zx -C agent

RUN chmod 775 -R /opt/opentelemetry/
RUN chmod a+w /opt/opentelemetry/logs

FROM ghcr.io/open-telemetry/opentelemetry-operator/cp:1.0.0 AS cp

FROM scratch

COPY --from=cp /cp /usr/bin/cp

COPY --from=agent /opt/opentelemetry/agent/opentelemetry-webserver-sdk /opt/opentelemetry
