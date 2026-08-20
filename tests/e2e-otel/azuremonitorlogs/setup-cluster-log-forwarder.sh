#!/bin/bash
set -e

echo "--- Setting up ClusterLogForwarder ---"

COLLECTOR_SERVICE="otel-logs-gateway-collector.${NAMESPACE}.svc.cluster.local"
echo "  OTel Collector service: $COLLECTOR_SERVICE"

# 1. Create the logcollector service account (required before CLF creation)
echo "Creating logcollector service account..."
oc -n openshift-logging create serviceaccount logcollector 2>/dev/null || true

# 2. Grant log collector permissions
echo "Granting log collector permissions..."
oc adm policy add-cluster-role-to-user collect-application-logs system:serviceaccount:openshift-logging:logcollector

# 3. Apply ClusterLogForwarder
echo "Creating ClusterLogForwarder..."
cat <<EOF | oc apply -f -
apiVersion: observability.openshift.io/v1
kind: ClusterLogForwarder
metadata:
  name: instance
  namespace: openshift-logging
  annotations:
    observability.openshift.io/tech-preview-otlp-output: "enabled"
spec:
  managementState: Managed
  serviceAccount:
    name: logcollector
  filters:
    - name: multiline-exceptions
      type: detectMultilineException
  outputs:
    - name: otel-collector
      type: otlp
      otlp:
        url: http://${COLLECTOR_SERVICE}:4318/v1/logs
  pipelines:
    - name: app-logs-to-azure
      inputRefs:
        - application
      outputRefs:
        - otel-collector
      filterRefs:
        - multiline-exceptions
EOF

# 4. Wait for log collector pods to be ready
echo "Waiting for log collector pods to be ready..."
sleep 10
oc -n openshift-logging wait pod -l app.kubernetes.io/component=collector --for=condition=Ready --timeout=120s 2>/dev/null || \
    oc -n openshift-logging wait pod -l component=collector --for=condition=Ready --timeout=120s 2>/dev/null || \
    echo "WARNING: Could not verify collector pod readiness, continuing..."

echo "--- ClusterLogForwarder setup completed ---"
