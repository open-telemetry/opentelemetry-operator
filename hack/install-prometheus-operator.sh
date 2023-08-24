#!/bin/bash

if [[ "$(kubectl api-resources --api-group=monitoring.coreos.com -o name)" ]]; then
    echo "Prometheus CRDs are there"
else
    kubectl create -f https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.66.0/bundle.yaml
fi
