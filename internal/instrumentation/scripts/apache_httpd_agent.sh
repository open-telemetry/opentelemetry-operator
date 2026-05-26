#!/bin/sh
# Init container script for Apache HTTPD auto-instrumentation.
# Runs in the otel-agent-attach-apache init container (the one with the
# instrumentation image), after the otel-agent-source-container-clone init
# container has staged the user's apache configuration into the shared volume.
#
# Constants below MUST stay in sync with the corresponding Go constants in
# internal/instrumentation/apachehttpd.go (apacheAgentDirFull,
# apacheAgentConfDirFull, apacheAgentConfigFile, apacheConfigFile).
#
# Inputs:
#   $1                                - Apache HTTPD configuration directory
#                                       (e.g. /usr/local/apache2/conf), passed
#                                       positionally so it is never parsed by
#                                       the shell. Sourced from
#                                       Instrumentation.spec.apacheHttpd.configPath.
#   OTEL_APACHE_AGENT_CONF (env)      - Generated OTel agent configuration file
#                                       contents.
#   APACHE_SERVICE_INSTANCE_ID (env)  - Pod name (downward API), used as
#                                       service.instance.id.
set -e

apache_conf_dir="$1"

# Copy the agent binaries into the shared volume.
cp -r /opt/opentelemetry/* /opt/opentelemetry-webserver/agent

# Render the log4cxx logging configuration from the agent template.
agent_log_dir=$(echo "/opt/opentelemetry-webserver/agent/logs" | sed 's,/,\\/,g')
sed "s/__agent_log_dir__/${agent_log_dir}/g" \
    /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml.template \
    > /opt/opentelemetry-webserver/agent/conf/opentelemetry_sdk_log4cxx.xml

# Materialise the OTel agent config file (placeholder pod name is replaced at runtime).
echo "${OTEL_APACHE_AGENT_CONF}" > /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf
sed -i "s/<<SID-PLACEHOLDER>>/${APACHE_SERVICE_INSTANCE_ID}/g" \
    /opt/opentelemetry-webserver/source-conf/opentemetry_agent.conf

# Wire the OTel agent config into the user's httpd.conf via Include directive.
printf '\nInclude %s/opentemetry_agent.conf\n' "$apache_conf_dir" \
    >> /opt/opentelemetry-webserver/source-conf/httpd.conf
