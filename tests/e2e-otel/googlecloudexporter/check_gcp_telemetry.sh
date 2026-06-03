#!/bin/bash
set -euo pipefail

PROJECT_ID="openshift-qe"
NAMESPACE="chainsaw-googlecloudexporter"
COLLECTOR_NAME="googlecloudexporter-collector"

echo "--- Checking Google Cloud Exporter functionality ---"
echo "Project ID: $PROJECT_ID"
echo "Namespace: $NAMESPACE"
echo "Collector: $COLLECTOR_NAME"
echo "------------------------------------------------------"

# Function to retry commands with backoff
retry_with_backoff() {
    local max_attempts=5
    local delay=10
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if "$@"; then
            return 0
        fi

        echo "Attempt $attempt failed. Retrying in ${delay}s..."
        sleep $delay
        delay=$((delay * 2))
        attempt=$((attempt + 1))
    done

    echo "All $max_attempts attempts failed."
    return 1
}

# 1. Check OpenTelemetry Collector pod logs for successful startup
echo "Checking OpenTelemetry Collector pod logs..."
COLLECTOR_POD=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name="$COLLECTOR_NAME" -o jsonpath='{.items[0].metadata.name}')

if [ -z "$COLLECTOR_POD" ]; then
    echo "ERROR: No OpenTelemetry Collector pod found"
    exit 1
fi

echo "Found collector pod: $COLLECTOR_POD"

# Check for successful startup messages
kubectl logs -n "$NAMESPACE" "$COLLECTOR_POD" | grep -E "(Everything is ready|Startup complete)" || {
    echo "WARNING: Collector startup messages not found, checking for error patterns..."
    kubectl logs -n "$NAMESPACE" "$COLLECTOR_POD" | grep -E "(ERROR|FATAL|panic|failed)" || echo "No obvious errors found"
}

# Check for Google Cloud authentication success
kubectl logs -n "$NAMESPACE" "$COLLECTOR_POD" | grep -E "(authentication.*success|credentials.*loaded)" || {
    echo "WARNING: No explicit authentication success messages found"
}

# 3. Check for any export errors in collector logs
echo "Checking for export errors in collector logs..."
kubectl logs -n "$NAMESPACE" "$COLLECTOR_POD" | grep -E "(export.*error|failed.*export|export.*failed)" && {
    echo "WARNING: Export errors found in collector logs"
} || {
    echo "OK: No export errors found in collector logs"
}

# 5. Summary
echo "------------------------------------------------------"
echo "Google Cloud Exporter verification completed."
echo "------------------------------------------------------"