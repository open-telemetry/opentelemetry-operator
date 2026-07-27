#!/usr/bin/env bash
# Verifies Spec OTEL_NODE_IP is forwarded to the nginx attach init container
# before OTEL_NGINX_AGENT_CONF, kubelet expands $(OTEL_NODE_IP) into the
# rendered agent config, and the concrete node IP is used (no Service).
set -euo pipefail

POD=$(kubectl get pod -n "$NAMESPACE" -l app=my-nginx -o jsonpath='{.items[0].metadata.name}')
test -n "$POD"

ENV_JSON=$(kubectl get pod -n "$NAMESPACE" "$POD" -o json)
NODE_IP=$(echo "$ENV_JSON" | jq -r '.status.hostIP')
test -n "$NODE_IP"
test "$NODE_IP" != "null"

NAMES=$(echo "$ENV_JSON" | jq -r '.spec.initContainers[] | select(.name=="otel-agent-attach-nginx") | .env[].name')
NODE_IDX=$(echo "$NAMES" | grep -n '^OTEL_NODE_IP$' | head -1 | cut -d: -f1)
AGENT_CONF_IDX=$(echo "$NAMES" | grep -n '^OTEL_NGINX_AGENT_CONF$' | head -1 | cut -d: -f1)
test -n "$NODE_IDX"
test -n "$AGENT_CONF_IDX"
test "$NODE_IDX" -lt "$AGENT_CONF_IDX"

AGENT_CONF=$(echo "$ENV_JSON" | jq -r '.spec.initContainers[] | select(.name=="otel-agent-attach-nginx") | .env[] | select(.name=="OTEL_NGINX_AGENT_CONF") | .value')
echo "$AGENT_CONF" | grep -Fq '$(OTEL_NODE_IP)'

RENDERED=""
for _ in $(seq 1 30); do
  if RENDERED=$(kubectl exec -n "$NAMESPACE" "$POD" -c myapp -- cat /etc/nginx/conf.d/opentelemetry_agent.conf 2>/dev/null); then
    break
  fi
  sleep 2
done
test -n "$RENDERED"
echo "$RENDERED" | grep -Fq "NginxModuleOtelExporterEndpoint http://${NODE_IP}:4317"
if echo "$RENDERED" | grep -Fq '$(OTEL_NODE_IP)'; then
  echo "rendered agent conf still contains unexpanded \$(OTEL_NODE_IP)" >&2
  exit 1
fi

echo "nginx node-IP exporter endpoint interpolation verified (endpoint=http://${NODE_IP}:4317)"
