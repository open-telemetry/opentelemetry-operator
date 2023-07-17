#!/bin/bash
SECRET=$(oc get secret -n openshift-user-workload-monitoring | grep prometheus-user-workload-token | head -n 1 | awk '{print $1}')
TOKEN=$(echo $(oc get secret $SECRET -n openshift-user-workload-monitoring -o json | jq -r '.data.token') | base64 -d)
THANOS_QUERIER_HOST=$(oc get route thanos-querier -n openshift-monitoring -o json | jq -r '.spec.host')

response=$(curl -k -H "Authorization: Bearer $TOKEN" -H "Content-type: application/json" "https://$THANOS_QUERIER_HOST/api/v1/query?query=gen")

count=$(echo "$response" | jq -r '.data.result | length')

if [[ $count -eq 0 ]]; then
  echo "No telemetrygen metrics count with value present. Exiting with status 1."
  exit 1
else
  echo "telemetrygen metrics with value is present."
fi
