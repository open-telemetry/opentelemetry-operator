#!/bin/bash

TOKEN=$(oc -n openshift-logging create token otel-collector-deployment)
LOKI_URL=$(oc -n openshift-logging get route logging-loki -o json | jq '.spec.host' -r)

while true; do
  LOG_OUTPUT=$(logcli -o raw --tls-skip-verify \
  --bearer-token="${TOKEN}" \
  --addr "https://${LOKI_URL}/api/logs/v1/application" query '{log_type="application"}')

  if echo "$LOG_OUTPUT" | jq -e '
    . as $root |
    select(
      .body == "the message" and
      .severity == "Info" and
      .attributes.app == "server" and
      .resources."k8s.container.name" == "telemetrygen" and
      .resources."k8s.namespace.name" == "chainsaw-incllogs"
    )
  ' > /dev/null; then
    echo "Logs found:"
    break
  else
    echo "Logs not found. Continuing to check..."
    sleep 5
  fi
done
