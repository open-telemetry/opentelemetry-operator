#!/bin/sh
# Init container script for Nginx auto-instrumentation.
# Runs in the otel-agent-attach-nginx init container (the one with the
# instrumentation image), after the otel-agent-source-container-clone init
# container has staged the user's nginx configuration and the nginx version
# file into the shared volume.
#
# Constants below MUST stay in sync with the corresponding Go constants in
# internal/instrumentation/nginx.go (nginxAgentDirFull,
# nginxAgentConfDirFull).
#
# Inputs:
#   $1 - Filename (basename) of the nginx config to instrument
#        (e.g. nginx.conf), passed positionally so it is never parsed by the
#        shell. Sourced from Instrumentation.spec.nginx.configFile.
#   OTEL_NGINX_AGENT_CONF (env)            - Generated OTel agent configuration
#                                            file contents.
#   OTEL_NGINX_SERVICE_INSTANCE_ID (env)   - Pod name (downward API), used as
#                                            service.instance.id.
set -e
set -x

nginx_config_file="$1"

cp -r /opt/opentelemetry/* /opt/opentelemetry-webserver/agent

nginx_version=$(cat /opt/opentelemetry-webserver/source-conf/version.txt)
nginx_agent_log_dir=$(echo "/opt/opentelemetry-webserver/agent/logs" | sed 's,/,\\/,g')

sed "s,__agent_log_dir__,${nginx_agent_log_dir},g" \
    /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml.template \
    > /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml

# Materialise the OTel agent config file (placeholder pod name is replaced at runtime).
echo "${OTEL_NGINX_AGENT_CONF}" > /opt/opentelemetry-webserver/source-conf/opentelemetry_agent.conf
sed -i "s,<<SID-PLACEHOLDER>>,${OTEL_NGINX_SERVICE_INSTANCE_ID},g" \
    /opt/opentelemetry-webserver/source-conf/opentelemetry_agent.conf

# Inject directives at the top of the user's nginx config:
#   - load_module so nginx loads the OTel module
#   - env OTEL_RESOURCE_ATTRIBUTES so the env var is visible to nginx workers
sed -i "1s,^,load_module /opt/opentelemetry-webserver/agent/WebServerModule/Nginx/${nginx_version}/ngx_http_opentelemetry_module.so;\\n,g" \
    "/opt/opentelemetry-webserver/source-conf/${nginx_config_file}"
sed -i "1s,^,env OTEL_RESOURCE_ATTRIBUTES;\\n,g" \
    "/opt/opentelemetry-webserver/source-conf/${nginx_config_file}"

mv /opt/opentelemetry-webserver/source-conf/opentelemetry_agent.conf \
    /opt/opentelemetry-webserver/source-conf/conf.d
