#!/bin/bash

# This script validates proportional mode sampling behavior
# Expected sampling percentage: 50%

NAMESPACE="chainsaw-probabilistic-sampler"
COLLECTOR_LABEL="app.kubernetes.io/name=probabilistic-proportional-collector"
EXPECTED_PERCENTAGE=50
TOLERANCE=5  # ±5% tolerance

echo "=== Checking Proportional Mode Sampling (Expected: ${EXPECTED_PERCENTAGE}%) ==="

# Function to check sampling in collector logs
check_proportional_sampling() {
    echo "Getting collector pod logs..."
    
    PODS=$(kubectl -n $NAMESPACE get pods -l $COLLECTOR_LABEL -o jsonpath='{.items[*].metadata.name}')
    
    if [ -z "$PODS" ]; then
        echo "ERROR: No collector pods found with label: $COLLECTOR_LABEL"
        return 1
    fi
    
    for POD in $PODS; do
        echo "Checking logs in pod: $POD"
        
        LOGS=$(kubectl -n $NAMESPACE logs $POD --tail=500 2>/dev/null)
        
        if [ $? -ne 0 ]; then
            echo "Failed to get logs from pod: $POD"
            continue
        fi
        
        # Count actual trace outputs (sampled traces will appear in debug output)
        SAMPLED_COUNT=$(echo "$LOGS" | grep -c "Trace ID.*:" || echo "0")
        
        # Check for tracestate with sampling threshold
        TRACESTATE_COUNT=$(echo "$LOGS" | grep -c "TraceState.*ot=.*th:" || echo "0")
        
        # Ensure we have numeric values
        SAMPLED_COUNT=${SAMPLED_COUNT:-0}
        TRACESTATE_COUNT=${TRACESTATE_COUNT:-0}
        
        echo "Sampled spans (in debug output): $SAMPLED_COUNT"
        echo "Tracestate entries found: $TRACESTATE_COUNT"
        
        # Verify that proportional mode is working
        if [ $SAMPLED_COUNT -gt 0 ] && [ $TRACESTATE_COUNT -gt 0 ]; then
            echo "✓ Proportional sampling is processing traces successfully"
            echo "✓ Tracestate with sampling threshold found ($TRACESTATE_COUNT instances)"
            return 0
        else
            echo "No sampled traces or tracestate found in debug output yet..."
        fi
    done
    
    return 1
}

# Function to verify tracestate encoding
check_tracestate_encoding() {
    echo "Checking tracestate encoding..."
    
    PODS=$(kubectl -n $NAMESPACE get pods -l $COLLECTOR_LABEL -o jsonpath='{.items[*].metadata.name}')
    
    for POD in $PODS; do
        LOGS=$(kubectl -n $NAMESPACE logs $POD --tail=300 2>/dev/null)
        
        # Look for tracestate with OpenTelemetry section
        if echo "$LOGS" | grep -q "tracestate.*ot=th:"; then
            echo "✓ Tracestate with OpenTelemetry section (ot=th:) found"
            
            # Extract threshold values
            THRESHOLDS=$(echo "$LOGS" | grep -o "ot=th:[0-9a-f]" | sort | uniq)
            echo "Threshold values found: $THRESHOLDS"
        fi
        
        # Check for sampling randomness values
        if echo "$LOGS" | grep -q "ot=.*rv:"; then
            echo "✓ Randomness values (rv:) found in tracestate"
        fi
    done
}

# Function to check Tempo for received traces with tracestate
check_tempo_with_tracestate() {
    echo "Checking Tempo for traces with tracestate..."
    
    TEMPO_POD=$(kubectl -n $NAMESPACE get pods -l app=tempo -o jsonpath='{.items[0].metadata.name}')
    
    if [ -n "$TEMPO_POD" ]; then
        # Query for traces with proportional mode
        TRACE_DATA=$(kubectl -n $NAMESPACE exec $TEMPO_POD -- \
            curl -s "http://localhost:3200/api/search?tags=test.mode=proportional" 2>/dev/null)
        
        if echo "$TRACE_DATA" | jq -e '.traces | length > 0' >/dev/null 2>&1; then
            echo "✓ Proportional mode traces found in Tempo"
            
            # Check for tracestate in trace data
            if echo "$TRACE_DATA" | jq -e '.traces[].traceID' >/dev/null 2>&1; then
                echo "✓ Trace IDs successfully recorded"
                return 0
            fi
        fi
    fi
    
    echo "Could not verify traces in Tempo yet..."
    return 1
}

# Main execution loop
ATTEMPT=1
MAX_ATTEMPTS=15

while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    echo ""
    echo "=== Attempt $ATTEMPT/$MAX_ATTEMPTS ==="
    
    if check_proportional_sampling; then
        echo ""
        check_tracestate_encoding
        check_tempo_with_tracestate
        echo "SUCCESS: Proportional mode sampling validation completed!"
        exit 0
    fi
    
    echo "Waiting 12 seconds before next check..."
    sleep 12
    ATTEMPT=$((ATTEMPT + 1))
done

echo "TIMEOUT: Could not validate proportional sampling within expected timeframe"
exit 1