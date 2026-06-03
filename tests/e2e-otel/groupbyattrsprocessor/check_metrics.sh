#!/bin/bash

oc create serviceaccount e2e-test-metrics-reader -n $NAMESPACE
oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:$NAMESPACE:e2e-test-metrics-reader

TOKEN=$(oc create token e2e-test-metrics-reader -n $NAMESPACE)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

#Check metrics for OpenTelemetry collector instance.
metrics="otelcol_processor_groupbyattrs_num_non_grouped_metrics_total otelcol_processor_groupbyattrs_metric_groups_bucket otelcol_processor_groupbyattrs_metric_groups_count otelcol_processor_groupbyattrs_metric_groups_sum"

for metric in $metrics; do
query="$metric"
count=0

# Keep fetching and checking the metrics until metrics with value is present.
while [[ $count -eq 0 ]]; do
    response=$(curl -k -H "Authorization: Bearer $TOKEN" -H "Content-type: application/json" "https://$THANOS_QUERIER_HOST/api/v1/query?query=$query")
    count=$(echo "$response" | jq -r '.data.result | length')

    if [[ $count -eq 0 ]]; then
    echo "No metric '$metric' with value present. Retrying..."
    sleep 5  # Wait for 5 seconds before retrying
    else
    echo "Metric '$metric' with value is present."
    fi
  done
done

