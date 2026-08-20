#!/bin/bash
# This script verifies the k8s_leader_elector extension is working correctly:
# 1. A Lease object exists in the test namespace
# 2. Leader election log messages are present in collector pods
# 3. Metrics are collected by exactly one (leader) pod
# 4. The non-leader pod does NOT collect metrics (proves exclusivity)

LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"
NAMESPACE=chainsaw-k8sleaderelectorext
LEASE_NAME="otel-k8sclusterreceiver-leader"

FOUND_LEASE=false
FOUND_LEADER_LOG=false
VERIFIED_EXCLUSIVITY=false

while ! $FOUND_LEASE || ! $FOUND_LEADER_LOG || ! $VERIFIED_EXCLUSIVITY; do
    # Check that the Lease object exists
    if ! $FOUND_LEASE && kubectl -n "$NAMESPACE" get lease "$LEASE_NAME" > /dev/null 2>&1; then
        echo "Lease '$LEASE_NAME' exists"
        FOUND_LEASE=true
    fi

    PODS=($(kubectl -n "$NAMESPACE" get pods -l "$LABEL_SELECTOR" -o jsonpath='{.items[*].metadata.name}'))

    LEADER_POD=""
    PODS_WITH_METRICS=0
    PODS_WITHOUT_METRICS=0

    for POD in "${PODS[@]}"; do
        LOGS=$(kubectl -n "$NAMESPACE" logs "$POD" --tail=500 2>/dev/null)

        # Check for leader election activity (klog uses capital "Successfully acquired lease")
        if ! $FOUND_LEADER_LOG && echo "$LOGS" | grep -qi -e "successfully acquired lease" -e "Starting k8sClusterReceiver with leader election"; then
            echo "Leader election log found in $POD"
            FOUND_LEADER_LOG=true
        fi

        # Track which pods have metrics
        if echo "$LOGS" | grep -q "k8s.node.allocatable_cpu"; then
            LEADER_POD="$POD"
            PODS_WITH_METRICS=$((PODS_WITH_METRICS + 1))
        else
            PODS_WITHOUT_METRICS=$((PODS_WITHOUT_METRICS + 1))
        fi
    done

    # Verify exclusivity: exactly one pod has metrics, the rest do not
    if [ "$PODS_WITH_METRICS" -eq 1 ] && [ "$PODS_WITHOUT_METRICS" -gt 0 ]; then
        echo "Metrics 'k8s.node.allocatable_cpu' found only in leader pod $LEADER_POD"
        echo "Non-leader pods confirmed not collecting metrics ($PODS_WITHOUT_METRICS standby)"
        VERIFIED_EXCLUSIVITY=true
    elif [ "$PODS_WITH_METRICS" -gt 1 ]; then
        echo "ERROR: $PODS_WITH_METRICS pods are collecting metrics (leader election not enforcing exclusivity)"
        exit 1
    fi

    if ! $FOUND_LEASE || ! $FOUND_LEADER_LOG || ! $VERIFIED_EXCLUSIVITY; then
        sleep 5
    fi
done

echo ""
echo "=== Leader Election Extension Verification ==="
echo "Lease object created: PASS"
echo "Leader election active: PASS"
echo "Metrics collected by leader only: PASS"
echo "Non-leader exclusivity verified: PASS"
echo "All checks passed."
