#!/bin/bash

# Install metrics-server on kind clusters for autoscale tests.  Note: This is not needed for minikube,
# you can just add --addons "metrics-server" to the start command.
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl patch deployment -n kube-system metrics-server --type "json" -p '[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": --kubelet-insecure-tls}]'
kubectl wait --for=condition=available deployment/metrics-server -n kube-system  --timeout=5m
