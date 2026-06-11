#!/bin/bash

if [[ "$(kubectl api-resources --api-group=gateway.networking.k8s.io -o name)" ]]; then
    echo "Gateway API CRDs are already installed"
else
    echo "Installing Gateway API CRDs..."
    kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml
fi
