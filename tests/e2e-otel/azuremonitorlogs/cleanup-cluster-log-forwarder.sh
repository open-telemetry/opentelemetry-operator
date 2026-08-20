#!/bin/bash
set -e

echo "--- Cleaning up ClusterLogForwarder ---"

oc delete clusterlogforwarder instance -n openshift-logging --ignore-not-found=true 2>/dev/null || true

# Remove cluster role bindings granted to logcollector
oc adm policy remove-cluster-role-from-user collect-application-logs system:serviceaccount:openshift-logging:logcollector 2>/dev/null || true

# Remove the logcollector service account
oc -n openshift-logging delete serviceaccount logcollector --ignore-not-found=true 2>/dev/null || true

echo "--- ClusterLogForwarder cleanup completed ---"
