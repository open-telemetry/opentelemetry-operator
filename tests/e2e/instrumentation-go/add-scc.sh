#!/bin/bash

if [[ "$(kubectl api-resources --api-group=operator.openshift.io -o name)" ]]; then
    kubectl apply -f scc.yaml
    oc adm policy add-scc-to-user otel-go-instrumentation -z otel-instrumentation-go -n $NAMESPACE
fi
