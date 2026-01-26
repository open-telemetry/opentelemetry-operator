#!/usr/bin/env bash

oc create serviceaccount e2e-test-metrics-reader -n $NAMESPACE
oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:$NAMESPACE:e2e-test-metrics-reader

TOKEN=$(oc create token e2e-test-metrics-reader -n $NAMESPACE)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

while true; do
  response=$(curl -k -H "Authorization: Bearer $TOKEN" -H "Content-type: application/json" "https://$THANOS_QUERIER_HOST/api/v1/query?query=gen")
  count=$(echo "$response" | jq -r '.data.result | length')

  if [[ $count -eq 0 ]]; then
    echo "No telemetrygen metrics count with value present. Fetching again..."
  else
    echo "telemetrygen metrics with value is present."
    break
  fi
done
