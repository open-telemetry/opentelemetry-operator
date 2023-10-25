#!/bin/bash

if [[ "$(kubectl api-resources --api-group=monitoring.coreos.com -o name)" ]]; then
    echo "Prometheus CRDs are there"
else
    kubectl create -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
    kubectl create -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/example/prometheus-operator-crd/monitoring.coreos.com_podmonitors.yaml
fi
