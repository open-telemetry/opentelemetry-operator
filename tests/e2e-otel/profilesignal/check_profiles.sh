#!/bin/sh
# This script verifies the OpenTelemetry collector profiles pipeline is configured correctly
# and that the collector starts without errors.

# Define the label selector for the profiles collector
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"

# Function to check profiles configuration
check_profiles_in_collector() {
    echo "Checking profile signal support in collector..."

    # Get the collector pod
    PODS=$(kubectl -n $NAMESPACE get pods -l $LABEL_SELECTOR -o jsonpath='{.items[*].metadata.name}')

    # Check if PODS is empty
    if [ -z "$PODS" ]; then
        echo "✗ No collector pods found with label: $LABEL_SELECTOR"
        return 1
    fi

    echo "✓ Found collector pods: $PODS"

    # Check each pod
    for POD in $PODS; do
        echo ""
        echo "Checking pod: $POD"

        # Verify the feature gate is enabled by checking pod args
        echo "Verifying feature gate is enabled..."
        ARGS=$(kubectl -n $NAMESPACE get pod $POD -o jsonpath='{.spec.containers[0].args}' 2>/dev/null)

        if echo "$ARGS" | grep -q "service.profilesSupport"; then
            echo "✓ Feature gate 'service.profilesSupport' is enabled"
        else
            echo "✗ Feature gate 'service.profilesSupport' not found in pod args"
            return 1
        fi

        # Check if the pod is ready
        POD_READY=$(kubectl -n $NAMESPACE get pod $POD -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null)

        if [ "$POD_READY" != "True" ]; then
            echo "⚠ Pod $POD is not ready yet. Current status: $POD_READY"
            continue
        fi

        echo "✓ Pod $POD is ready"

        # Get logs from the pod to check for errors
        echo "Checking logs for errors..."
        LOGS=$(kubectl -n $NAMESPACE logs $POD --tail=200 2>/dev/null)

        if [ $? -ne 0 ]; then
            echo "✗ Failed to get logs from pod: $POD"
            return 1
        fi

        # Check for common error patterns
        if echo "$LOGS" | grep -i "error.*profile" | grep -v "no error"; then
            echo "✗ Found profile-related errors in logs:"
            echo "$LOGS" | grep -i "error.*profile" | grep -v "no error" | head -5
            return 1
        fi

        if echo "$LOGS" | grep -i "fatal"; then
            echo "✗ Found fatal errors in logs:"
            echo "$LOGS" | grep -i "fatal" | head -5
            return 1
        fi

        if echo "$LOGS" | grep -i "panic"; then
            echo "✗ Found panic in logs:"
            echo "$LOGS" | grep -i "panic" | head -5
            return 1
        fi

        echo "✓ No errors found in collector logs"

        # Check for profiles pipeline in logs
        if echo "$LOGS" | grep -q "profiles"; then
            echo "✓ Profiles pipeline found in collector logs"
        else
            echo "⚠ Profiles pipeline not yet visible in logs (this may be normal during startup)"
        fi

        # Verify the collector's configuration
        echo "Verifying collector configuration..."
        CONFIG=$(kubectl -n $NAMESPACE get opentelemetrycollector otel-profiles-collector -o yaml 2>/dev/null)

        if [ $? -ne 0 ]; then
            echo "✗ Failed to get OpenTelemetryCollector resource"
            return 1
        fi

        # Check for profiles pipeline in config
        if echo "$CONFIG" | grep -q "profiles:"; then
            echo "✓ Profiles pipeline configured in OpenTelemetryCollector resource"
        else
            echo "✗ Profiles pipeline not found in configuration"
            return 1
        fi

        # Verify OTLP receiver is configured
        if echo "$CONFIG" | grep -q "otlp:"; then
            echo "✓ OTLP receiver configured"
        else
            echo "✗ OTLP receiver not configured"
            return 1
        fi

        # Verify debug exporter is configured
        if echo "$CONFIG" | grep -q "debug:"; then
            echo "✓ Debug exporter configured"
        else
            echo "✗ Debug exporter not configured"
            return 1
        fi

        echo ""
        echo "SUCCESS: Profile signal support verified!"
        echo "Summary:"
        echo "  ✓ Feature gate 'service.profilesSupport' enabled"
        echo "  ✓ Profiles pipeline configured with OTLP receiver"
        echo "  ✓ Debug exporter configured"
        echo "  ✓ Collector pod is ready and running"
        echo "  ✓ No errors found in collector logs"
        return 0
    done

    echo "✗ Profile signal verification incomplete"
    return 1
}

# Main loop - check until success or timeout
echo "Starting profile signal verification..."
echo "This will check until verification is complete or chainsaw timeout is reached..."

ATTEMPT=1
while true; do
    echo ""
    echo "=== Attempt $ATTEMPT ==="

    if check_profiles_in_collector; then
        echo ""
        echo "Profile signal test verification completed successfully."
        exit 0
    fi

    echo "Verification not complete. Waiting 10 seconds before next check..."
    sleep 10
    ATTEMPT=$((ATTEMPT + 1))
done
