
############################
# STEP 1 download the webserver agent 
############################
FROM alpine:latest@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4 AS agent

ARG version

RUN mkdir /opt/opentelemetry
WORKDIR /opt/opentelemetry

RUN mkdir agent
RUN wget -c https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/download/webserver%2Fv$version/opentelemetry-webserver-sdk-x64-linux.tgz
RUN unzip -p opentelemetry-webserver-sdk-x64-linux.tgz | tar -zx -C agent

############################
# STEP 2 download the webserver agent 
############################
FROM alpine:latest@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

COPY --from=agent /opt/opentelemetry/agent/opentelemetry-webserver-sdk /opt/opentelemetry

RUN chmod 775 -R /opt/opentelemetry/
RUN chmod a+w /opt/opentelemetry/logs

CMD ["cat", "Just delivering the Opentelemetry Apache/Nginx agent"]