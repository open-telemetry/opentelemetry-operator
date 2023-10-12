#!/bin/bash
if [[ "$(kubectl api-resources --api-group=operator.openshift.io -o name)" ]]; then
    echo "Running the test against an OpenShift Cluster"
    echo "Creating an Service Account"
    echo "Creating a Security Context Constrain"
    echo "Setting the Service Account for the Daemonset"
    echo "Adding the new policy to the Service Account"
    kubectl apply -f scc.yaml -n $NAMESPACE
    oc adm policy add-scc-to-user -z otel-collector-daemonset daemonset-with-hostport -n $NAMESPACE
fi
