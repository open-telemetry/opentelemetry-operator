#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${1:?namespace required}"

# Get the namespace's UID range start — this is the value the operator should
# have defaulted fsGroup to.
range_annotation=$(kubectl get namespace "$NAMESPACE" \
  -o jsonpath='{.metadata.annotations.openshift\.io/sa\.scc\.supplemental-groups}')
if [[ -z "$range_annotation" ]]; then
  range_annotation=$(kubectl get namespace "$NAMESPACE" \
    -o jsonpath='{.metadata.annotations.openshift\.io/sa\.scc\.uid-range}')
fi

if [[ -z "$range_annotation" ]]; then
  echo "FAIL: namespace $NAMESPACE has no UID range annotation"
  exit 1
fi

# Parse the range start (format: "start/size" or "start-end")
expected_fsgroup=$(echo "$range_annotation" | sed -E 's@[/-].*@@')

# Get the actual fsGroup from the StatefulSet pod spec
actual_fsgroup=$(kubectl get statefulset fsgroup-test-collector -n "$NAMESPACE" \
  -o jsonpath='{.spec.template.spec.securityContext.fsGroup}')

if [[ "$actual_fsgroup" != "$expected_fsgroup" ]]; then
  echo "FAIL: expected fsGroup=$expected_fsgroup, got fsGroup=$actual_fsgroup"
  exit 1
fi

echo "PASS: fsGroup=$actual_fsgroup matches namespace range start"
