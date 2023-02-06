#!/bin/bash

if [[ "$(kubectl api-resources --api-group=operator.openshift.io -o name)" ]]; then
    echo "Connected to an OpenShift cluster. OpenShift routes installation is not needed"
else
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/router_rbac.yaml
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/route_crd.yaml
    kubectl apply -f https://raw.githubusercontent.com/openshift/router/release-4.12/deploy/router.yaml
    kubectl wait --for=condition=available deployment/ingress-router -n openshift-ingress --timeout=5m
fi
