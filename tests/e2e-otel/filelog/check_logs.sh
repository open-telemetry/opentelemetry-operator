
#!/bin/sh
# This script checks the OpenTelemetry collector pod for the presence of Logs.
# It continuously checks until logs are found or the script is terminated by timeout.

# Define the label selector
LABEL_SELECTOR="app.kubernetes.io/component=opentelemetry-collector"

# Define the search strings for container operator format (POSIX-compliant)
SEARCH_STRING_1="log.file.path"
SEARCH_STRING_2="SVTLogger"
SEARCH_STRING_3="Body: Str(.*SVTLogger.*app-log-plaintext-"
SEARCH_STRING_4="k8s.container.name: Str(app-log-plaintext)"
SEARCH_STRING_5="k8s.namespace.name: Str(chainsaw-filelog)"
SEARCH_STRING_COUNT=5

# Function to get search string by index
get_search_string() {
    case $1 in
        1) echo "$SEARCH_STRING_1" ;;
        2) echo "$SEARCH_STRING_2" ;;
        3) echo "$SEARCH_STRING_3" ;;
        4) echo "$SEARCH_STRING_4" ;;
        5) echo "$SEARCH_STRING_5" ;;
        *) echo "" ;;
    esac
}

# Function to check logs in all collector pods
check_logs_in_collectors() {
    echo "Checking logs in all collector instances..."
    
    # Get the list of pods with the specified label
    PODS=$(kubectl -n $NAMESPACE get pods -l $LABEL_SELECTOR -o jsonpath='{.items[*].metadata.name}')
    
    # Check if PODS is empty
    if [ -z "$PODS" ]; then
        echo "No collector pods found with label: $LABEL_SELECTOR"
        return 1
    fi
    
    echo "Found collector pods: $PODS"
    
    # Check each pod until we find all search strings
    for POD in $PODS; do
        echo "Checking logs in pod: $POD"
        
        # Get logs from the pod
        LOGS=$(kubectl -n $NAMESPACE --tail=200 logs $POD 2>/dev/null)
        
        if [ $? -ne 0 ]; then
            echo "Failed to get logs from pod: $POD"
            continue
        fi
        
        # Check if all search strings are present in this pod's logs
        all_found=true
        i=1
        while [ $i -le $SEARCH_STRING_COUNT ]; do
            STRING=$(get_search_string $i)
            if echo "$LOGS" | grep -q -- "$STRING"; then
                echo "✓ \"$STRING\" found in $POD"
            else
                all_found=false
                break
            fi
            i=$((i + 1))
        done
        
        # If all strings found in this pod, we're done
        if [ "$all_found" = "true" ]; then
            echo "SUCCESS: All required log strings found in collector instance $POD!"
            return 0
        fi
    done
    
    echo "Not all log strings found in any collector instance."
    return 1
}

# Main loop - continuously check until success or timeout
echo "Starting continuous log checking for app-log-plaintext.log in collector instances..."
echo "This will continue until logs are found or chainsaw timeout is reached..."

ATTEMPT=1
while true; do
    echo ""
    echo "=== Attempt $ATTEMPT ==="
    
    if check_logs_in_collectors; then
        echo "Found logs for app-log-plaintext.log in collector instances."
        exit 0
    fi
    
    echo "Logs not yet available. Waiting 10 seconds before next check..."
    sleep 10
    ATTEMPT=$((ATTEMPT + 1))
done
