#!/bin/bash

# Install metrics-server on kind clusters for autoscale tests.
# Note: This is not needed for minikube,
# you can just add --addons "metrics-server" to the start command.


if [[ "$(kubectl api-resources --api-group=operator.openshift.io -o name)" ]]; then
    echo "Connected to an OpenShift cluster. metrics-server installation is not needed"
elif [[ "$(kubectl get deployment metrics-server -n kube-system 2>&1 )" =~ "NotFound" ]]; then
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
    kubectl patch deployment -n kube-system metrics-server --type "json" -p '[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": --kubelet-insecure-tls}]'
    kubectl wait --for=condition=available deployment/metrics-server -n kube-system  --timeout=5m
else
    echo "metrics-server is installed. Skipping installation"
fi
