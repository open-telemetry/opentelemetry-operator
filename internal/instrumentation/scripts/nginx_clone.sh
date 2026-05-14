#!/bin/sh
# Clone-container init script for Nginx auto-instrumentation.
# Runs as a clone of the application container so it can read the original
# nginx binary and configuration files. Copies the configuration directory
# into a shared volume and records the nginx version for the agent init
# container to pick up.
#
# Constants below MUST stay in sync with the corresponding Go constants in
# internal/instrumentation/nginx.go (nginxAgentConfDirFull).
#
# Inputs:
#   $1 - Directory containing the nginx configuration (e.g. /etc/nginx),
#        passed positionally so it is never parsed by the shell. Sourced from
#        the parent directory of Instrumentation.spec.nginx.configFile.
set -e

nginx_conf_dir="$1"

cp -r "$nginx_conf_dir"/* /opt/opentelemetry-webserver/source-conf

# nginx prints the version banner on stderr; merge to stdout, strip the
# "nginx version: " prefix, and persist for the agent init container.
nginx_version="$( { nginx -v ; } 2>&1 )"
echo "${nginx_version##*/}" > /opt/opentelemetry-webserver/source-conf/version.txt
