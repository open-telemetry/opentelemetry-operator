#!/bin/bash
if [[ "$(kubectl api-resources --api-group=operator.openshift.io -o name)" ]]; then
    echo "Adding service account to the OpenTelemetry Collector"
    kubectl patch otelcol daemonset --type=merge -p '{"spec":{"serviceAccount":"otel-collector-daemonset"}}' -n $NAMESPACE
fi
