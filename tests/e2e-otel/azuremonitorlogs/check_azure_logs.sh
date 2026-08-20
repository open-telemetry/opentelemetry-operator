#!/bin/bash
set -uo pipefail

COLLECTOR_NAME="otel-logs-gateway-collector"

echo "--- Verifying logs in Azure Monitor ---"

# 1. Get workspace customer ID from ConfigMap
WORKSPACE_CUSTOMER_ID=$(oc -n "$NAMESPACE" get configmap azure-monitor-test-config -o jsonpath='{.data.WORKSPACE_CUSTOMER_ID}')
if [ -z "$WORKSPACE_CUSTOMER_ID" ]; then
    echo "ERROR: Could not retrieve WORKSPACE_CUSTOMER_ID from ConfigMap"
    exit 1
fi
echo "  Workspace Customer ID: $WORKSPACE_CUSTOMER_ID"

# 2. Check OTel Collector pod logs for export errors
echo "Checking OTel Collector pod for errors..."
COLLECTOR_POD=$(oc -n "$NAMESPACE" get pods -l app.kubernetes.io/name="$COLLECTOR_NAME" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$COLLECTOR_POD" ]; then
    ERRORS=$(oc -n "$NAMESPACE" logs "$COLLECTOR_POD" 2>/dev/null | grep -c "exporting failed" || true)
    if [ "$ERRORS" -gt 0 ]; then
        echo "WARNING: Found $ERRORS export errors in collector logs"
        oc -n "$NAMESPACE" logs "$COLLECTOR_POD" 2>/dev/null | grep "exporting failed" | tail -5
    else
        echo "  No export errors found in collector logs"
    fi

    EXPORTED=$(oc -n "$NAMESPACE" logs "$COLLECTOR_POD" 2>/dev/null | grep -c "LogsExporter" || true)
    echo "  Log export entries in collector: $EXPORTED"
else
    echo "WARNING: Could not find collector pod"
fi

# 3. Query Azure Monitor for OTelLogs containing SVTLogger
echo "Querying Azure Monitor OTelLogs table..."
MAX_RETRIES=25
RETRY_INTERVAL=30

for i in $(seq 1 $MAX_RETRIES); do
    echo "  Attempt $i/$MAX_RETRIES: Querying for SVTLogger logs..."
    RESULT=$(az monitor log-analytics query \
        --workspace "$WORKSPACE_CUSTOMER_ID" \
        --analytics-query "OTelLogs | where Body has 'SVTLogger' | take 10" \
        --output json 2>/dev/null || echo "[]")

    COUNT=$(echo "$RESULT" | jq '. | length' 2>/dev/null || echo "0")

    if [ "$COUNT" -gt 0 ] && [ "$COUNT" != "null" ]; then
        echo "Found $COUNT log entries containing 'SVTLogger' in Azure Monitor"
        echo "Sample log entry:"
        echo "$RESULT" | jq '.[0]' 2>/dev/null
        echo "--- Azure Monitor log verification PASSED ---"
        exit 0
    fi

    echo "  No logs found yet, waiting ${RETRY_INTERVAL}s... (elapsed: $((i * RETRY_INTERVAL))s)"
    sleep $RETRY_INTERVAL
done

echo "ERROR: No SVTLogger logs found in Azure Monitor after $((MAX_RETRIES * RETRY_INTERVAL))s"

# Re-check collector for export errors
if [ -n "$COLLECTOR_POD" ]; then
    echo "--- Collector export errors ---"
    oc -n "$NAMESPACE" logs "$COLLECTOR_POD" 2>/dev/null | grep -iE "(error|fail|reject|denied|unauthorized|403|401|500)" | tail -20 || echo "  No errors found"
    echo "--- Collector pod logs (last 50 lines) ---"
    oc -n "$NAMESPACE" logs "$COLLECTOR_POD" --tail=50 2>/dev/null || true
fi

exit 1
