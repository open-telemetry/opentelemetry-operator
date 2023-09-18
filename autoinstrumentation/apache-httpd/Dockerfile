
############################
# STEP 1 download the webserver agent 
############################
FROM alpine:latest as agent

ARG version

RUN mkdir /opt/opentelemetry
WORKDIR /opt/opentelemetry

RUN wget https://github.com/open-telemetry/opentelemetry-cpp-contrib/releases/download/webserver%2Fv$version/opentelemetry-webserver-sdk-x64-linux.tgz
RUN mkdir agent
RUN tar -xvf opentelemetry-webserver-sdk-x64-linux.tgz -C agent

############################
# STEP 2 download the webserver agent 
############################
FROM alpine:latest

COPY --from=agent /opt/opentelemetry/agent/opentelemetry-webserver-sdk /opt/opentelemetry

RUN chmod 775 -R /opt/opentelemetry/
RUN chmod a+w /opt/opentelemetry/logs

CMD ["cat", "Just delivering the Opentelemetry Apache/Nginx agent"]