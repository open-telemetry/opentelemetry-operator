#!/bin/bash
set -euo pipefail

# Existing Pods do not rerun admission; recreate the Pod to observe base updates.
for attempt in $(seq 1 30); do
  image=$(kubectl get pod "$POD_NAME" -n "$POD_NAMESPACE" \
    -o jsonpath='{.spec.initContainers[?(@.name=="opentelemetry-auto-instrumentation-java")].image}' 2>/dev/null || true)
  if [ "$image" = "$EXPECTED_JAVA_IMAGE" ]; then
    echo "$POD_NAMESPACE/$POD_NAME uses $EXPECTED_JAVA_IMAGE"
    exit 0
  fi

  if [ "$attempt" -eq 30 ]; then
    echo "$POD_NAMESPACE/$POD_NAME still uses '$image'; expected '$EXPECTED_JAVA_IMAGE'"
    exit 1
  fi

  echo "$POD_NAMESPACE/$POD_NAME uses '$image' (attempt $attempt/30); creating another Pod"
  kubectl delete pod "$POD_NAME" -n "$POD_NAMESPACE" --ignore-not-found --wait=true --timeout=30s
  pod_env=""
  if [ -n "${POD_PRIORITY_VALUE:-}" ]; then
    pod_env=$(printf '    env:\n    - name: POD_PRIORITY\n      value: %s' "$POD_PRIORITY_VALUE")
  fi
  kubectl create -n "$POD_NAMESPACE" -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $POD_NAME
  annotations:
    instrumentation.opentelemetry.io/inject-java: java-local
spec:
  nodeSelector:
    e2e.opentelemetry.io/no-node: "true"
  containers:
  - name: app
    image: busybox:1.36.1
    command: [sh, -c, "sleep 3600"]
$pod_env
EOF
  sleep 2
done
