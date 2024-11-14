#!/bin/bash

TOKEN=$(oc create token prometheus-user-workload -n openshift-user-workload-monitoring)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

# Check metrics for OpenTelemetry collector instance.
metrics="otelcol_process_uptime otelcol_process_runtime_total_sys_memory_bytes otelcol_process_memory_rss otelcol_exporter_sent_spans otelcol_process_cpu_seconds otelcol_process_memory_rss otelcol_process_runtime_heap_alloc_bytes otelcol_process_runtime_total_alloc_bytes otelcol_process_runtime_total_sys_memory_bytes otelcol_process_uptime otelcol_receiver_accepted_spans otelcol_receiver_refused_spans controller_runtime_reconcile_time_seconds_count{controller=\"opentelemetrycollector\"} controller_runtime_reconcile_total{controller=\"opentelemetrycollector\",result=\"success\"} workqueue_work_duration_seconds_count{controller=\"opentelemetrycollector\",name=\"opentelemetrycollector\"}"

for metric in $metrics; do
  query="$metric"
  count=0

  # Keep fetching and checking the metrics until metrics with value is present.
  while [[ $count -eq 0 ]]; do
    response=$(curl -k -H "Authorization: Bearer $TOKEN" --data-urlencode "query=$query" "https://$THANOS_QUERIER_HOST/api/v1/query")
    count=$(echo "$response" | jq -r '.data.result | length' | tr -d '\n' | tr -d ' ')

    if [[ "$count" -eq 0 ]]; then
      echo "No metric '$metric' with value present. Retrying..."
      sleep 5  # Wait for 5 seconds before retrying
    else
      echo "Metric '$metric' with value is present."
    fi
  done
done
