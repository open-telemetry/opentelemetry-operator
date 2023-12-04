#!/bin/bash

SECRET=$(oc get secret -n openshift-user-workload-monitoring | grep prometheus-user-workload-token | head -n 1 | awk '{print $1}')
TOKEN=$(echo $(oc get secret $SECRET -n openshift-user-workload-monitoring -o json | jq -r '.data.token') | base64 -d)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

#Check metrics for OpenTelemetry collector instance.
metrics="otelcol_process_uptime otelcol_process_runtime_total_sys_memory_bytes otelcol_process_memory_rss otelcol_exporter_sent_spans otelcol_process_cpu_seconds otelcol_process_memory_rss otelcol_process_runtime_heap_alloc_bytes otelcol_process_runtime_total_alloc_bytes otelcol_process_runtime_total_sys_memory_bytes otelcol_process_uptime otelcol_receiver_accepted_spans otelcol_receiver_refused_spans"

for metric in $metrics; do
  query="$metric"

  response=$(curl -k -H "Authorization: Bearer $TOKEN" -H "Content-type: application/json" "https://$THANOS_QUERIER_HOST/api/v1/query?query=$query")

  count=$(echo "$response" | jq -r '.data.result | length')

  if [[ $count -eq 0 ]]; then
    echo "No metric '$metric' with value present. Exiting with status 1."
    exit 1
  else
    echo "Metric '$metric' with value is present."
  fi
done
