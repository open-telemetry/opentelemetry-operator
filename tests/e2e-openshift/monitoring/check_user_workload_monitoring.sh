#!/bin/bash

set -e

check_replicas() {
  replicas=$(oc get $1 $2 -n openshift-user-workload-monitoring -o 'jsonpath={.status.availableReplicas} {.status.readyReplicas} {.status.replicas}')
  for count in $replicas; do
    if [[ $count =~ ^[0-9]+$ ]]; then
      if ((count < 1)); then
        echo "The number of replicas is 0 for $1 $2"
        exit 1
      fi
    else
      echo "Error: Replica count is not a valid number for $1 $2"
      exit 1
    fi
  done
}

check_replicas deployment prometheus-operator
check_replicas statefulset prometheus-user-workload
check_replicas statefulset thanos-ruler-user-workload
