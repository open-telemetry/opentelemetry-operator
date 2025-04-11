#!/bin/bash
# This script repeatedly checks pod logs from a specific container until all specified strings are found
# simultaneously or a timeout occurs.

# --- Configuration ---
LABEL_SELECTOR="app=my-java-other-ns"
CONTAINER_NAME="otc-container"    # <--- Specify the container to get logs from
RETRY_TIMEOUT=120 # Total seconds to keep retrying (e.g., 2 minutes)
RETRY_SLEEP=5    # Seconds to wait between retries

# Define the search strings - EACH ELEMENT MUST BE QUOTED
SEARCH_STRINGS=(
  "k8s.container.name: Str(myapp)"
  "k8s.deployment.name: Str(my-java-other-ns)"
  "os.type: Str(linux)"
  "process.executable.path: Str(/opt/java/openjdk/bin/java)"
  "service.name: Str(my-java-other-ns)"
  "telemetry.sdk.language: Str(java)"
  "telemetry.sdk.name: Str(opentelemetry)"
  "Trace ID"
  "Parent ID"
  "Name           : DemoApplication.home"
  "Name: http.server.duration"
  "Description: The duration of the inbound HTTP request"
  "Unit: ms"
  "DataType: Histogram"
  "AggregationTemporality: Cumulative"
  "Name: process.runtime.jvm.memory.usage"
  "Description: Measure of memory used"
)
# --- End Configuration ---

echo "Namespace: $NAMESPACE"
echo "Label Selector: $LABEL_SELECTOR"
echo "Container Name: $CONTAINER_NAME" # Added container name log
echo "Timeout: ${RETRY_TIMEOUT}s, Sleep: ${RETRY_SLEEP}s"
echo "Searching for ${#SEARCH_STRINGS[@]} strings..."

# --- Get Initial Pod Name ---
echo "Finding target pod..."
PODS_JSONPATH_OUTPUT=$(kubectl -n "$NAMESPACE" get pods -l "$LABEL_SELECTOR" -o jsonpath='{.items[*].metadata.name}')
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to run initial kubectl get pods. Is kubectl configured correctly for namespace '$NAMESPACE'?"
    exit 1
fi
read -r -a PODS <<< "$PODS_JSONPATH_OUTPUT"
if [ ${#PODS[@]} -eq 0 ]; then
    echo "ERROR: No pods found with label '$LABEL_SELECTOR' in namespace '$NAMESPACE'"
    exit 1
fi
POD=${PODS[0]}
echo "Target Pod: $POD"
# --- End Get Initial Pod Name ---

# --- Main Retry Loop ---
START_TIME=$(date +%s)
LAST_MISSING_COUNT=${#SEARCH_STRINGS[@]} # Track missing count to reduce log noise

while true; do
    CURRENT_TIME=$(date +%s)
    ELAPSED_TIME=$(( CURRENT_TIME - START_TIME ))

    # 1. Check for Timeout
    if [ "$ELAPSED_TIME" -ge "$RETRY_TIMEOUT" ]; then
        echo "-----------------------------------------------------"
        echo "ERROR: Timeout ($RETRY_TIMEOUT seconds) reached. Not all required strings were found in container '$CONTAINER_NAME' of pod '$POD'."
        echo "-----------------------------------------------------"
        exit 1
    fi

    echo "-----------------------------------------------------"
    echo "Attempting log check on container '$CONTAINER_NAME' (Elapsed: ${ELAPSED_TIME}s / ${RETRY_TIMEOUT}s)"

    # 2. Get Logs for this attempt from the SPECIFIC container
    # Fetches all logs using --tail=-1 each time
    LOGS=$(kubectl -n "$NAMESPACE" logs "$POD" -c "$CONTAINER_NAME" --tail=-1) # <-- Added -c "$CONTAINER_NAME"
    KUBECTL_LOGS_EXIT_CODE=$?

    if [ $KUBECTL_LOGS_EXIT_CODE -ne 0 ]; then
         echo "Warning: Failed to get logs for container '$CONTAINER_NAME' in pod '$POD' (Exit code: $KUBECTL_LOGS_EXIT_CODE). Retrying after sleep..."
         sleep "$RETRY_SLEEP"
         continue # Go to next loop iteration
    fi

    if [ -z "$LOGS" ]; then
         echo "Warning: Logs for container '$CONTAINER_NAME' in pod '$POD' were empty on this attempt. Retrying after sleep..."
         sleep "$RETRY_SLEEP"
         continue # Go to next loop iteration
    fi

    # 3. Check for all strings in the current logs
    ALL_FOUND_THIS_ATTEMPT=true
    MISSING_STRINGS_THIS_ATTEMPT=()
    for STRING in "${SEARCH_STRINGS[@]}"; do
        # Use grep -F (fixed string) and -q (quiet)
        if ! echo "$LOGS" | grep -Fq -- "$STRING"; then
            ALL_FOUND_THIS_ATTEMPT=false
            MISSING_STRINGS_THIS_ATTEMPT+=("$STRING")
            # Optimization: uncomment break if you don't need the full list of missing strings per attempt
            # break
        fi
    done

    # 4. Evaluate outcome of this attempt
    if [ "$ALL_FOUND_THIS_ATTEMPT" = true ]; then
        echo "-----------------------------------------------------"
        echo "Success: All ${#SEARCH_STRINGS[@]} strings found simultaneously in container '$CONTAINER_NAME' of pod '$POD'."
        echo "-----------------------------------------------------"
        exit 0 # Successful exit!
    else
        # Report missing strings only if the list changed since last time or first failure
        CURRENT_MISSING_COUNT=${#MISSING_STRINGS_THIS_ATTEMPT[@]}
        if [ "$CURRENT_MISSING_COUNT" -ne "$LAST_MISSING_COUNT" ]; then
             echo "Attempt Failed: Missing ${CURRENT_MISSING_COUNT} string(s):"
             printf "  - %s\n" "${MISSING_STRINGS_THIS_ATTEMPT[@]}" # Print missing strings
             LAST_MISSING_COUNT=$CURRENT_MISSING_COUNT
        else
             # Avoid repeating the full list if the same strings are missing
             echo "Attempt Failed: Still missing ${CURRENT_MISSING_COUNT} string(s). Waiting..."
        fi
        echo "Retrying in $RETRY_SLEEP seconds..."
        sleep "$RETRY_SLEEP"
    fi

done
# --- End Main Retry Loop ---

# Fallback exit status (should not be reached normally)
exit 1