#!/bin/bash
# This script repeatedly checks pod logs from a specific container until all specified strings are found
# simultaneously or a timeout occurs.

# --- Configuration ---
LABEL_SELECTOR=${LABEL_SELECTOR} # Read from env
CONTAINER_NAME=${CONTAINER_NAME} # Read from env
RETRY_TIMEOUT=${RETRY_TIMEOUT:-120} # Read from env, default 120
RETRY_SLEEP=${RETRY_SLEEP:-5}    # Read from env, default 5
SEARCH_STRINGS_INPUT=${SEARCH_STRINGS_ENV} # Read the delimited string from env

# --- Input Validation ---
if [[ -z "$LABEL_SELECTOR" ]]; then
  echo "ERROR: LABEL_SELECTOR environment variable is not set."
  exit 1
fi
if [[ -z "$CONTAINER_NAME" ]]; then
  echo "ERROR: CONTAINER_NAME environment variable is not set."
  exit 1
fi
if [[ -z "$SEARCH_STRINGS_INPUT" ]]; then
  echo "ERROR: SEARCH_STRINGS_ENV environment variable is not set or empty."
  exit 1
fi

# --- Process Search Strings ---
declare -a SEARCH_STRINGS # Declare the array

# Read the multiline env var into the array, IFS sets the delimiter to newline
# Handles strings potentially containing spaces correctly.
# The <(...) construct avoids a subshell, making the array available after the loop.
# Using printf and \0 ensures robustness even if strings contain special characters.
IFS=$'\n' read -r -d '' -a SEARCH_STRINGS < <(printf '%s\0' "$SEARCH_STRINGS_INPUT")

if [[ ${#SEARCH_STRINGS[@]} -eq 0 ]]; then
  echo "ERROR: Failed to parse any search strings from SEARCH_STRINGS_ENV."
  exit 1
fi
# --- End Configuration Processing ---


# --- Script Execution ---
echo "Namespace: $NAMESPACE" # NAMESPACE is implicitly set by Chainsaw
echo "Label Selector: $LABEL_SELECTOR"
echo "Container Name: $CONTAINER_NAME"
echo "Timeout: ${RETRY_TIMEOUT}s, Sleep: ${RETRY_SLEEP}s"
echo "Searching for ${#SEARCH_STRINGS[@]} strings:"
printf "  - '%s'\n" "${SEARCH_STRINGS[@]}" # Print strings for verification, quoting them


# --- Get Initial Pod Name ---
echo "Finding target pod..."
# Ensure kubectl uses the namespace chainsaw provides. Use --request-timeout for robustness.
PODS_JSONPATH_OUTPUT=$(kubectl get pods -n "$NAMESPACE" -l "$LABEL_SELECTOR" --request-timeout=10s -o jsonpath='{.items[0].metadata.name}')
KUBECTL_GET_EXIT_CODE=$?
if [ $KUBECTL_GET_EXIT_CODE -ne 0 ]; then
    echo "ERROR: Failed to run initial kubectl get pods (Exit Code: $KUBECTL_GET_EXIT_CODE). Is kubectl configured correctly for namespace '$NAMESPACE'?"
    exit 1
fi
# Handle case where no pod is found gracefully
if [[ -z "$PODS_JSONPATH_OUTPUT" ]]; then
    echo "ERROR: No pods found with label '$LABEL_SELECTOR' in namespace '$NAMESPACE'"
    exit 1
fi
# Assuming only one pod matches, if multiple could match, logic needs adjustment
POD=$PODS_JSONPATH_OUTPUT
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
        # Attempt to print last known missing strings, might be empty if loop never found anything
        if [[ ${#MISSING_STRINGS_THIS_ATTEMPT[@]} -gt 0 ]]; then
            echo "Last known missing strings:"
            printf "  - '%s'\n" "${MISSING_STRINGS_THIS_ATTEMPT[@]}"
        fi
        # Display last few lines of logs for debugging
        echo "Last 20 lines of logs from $POD/$CONTAINER_NAME:"
        kubectl logs "$POD" -n "$NAMESPACE" -c "$CONTAINER_NAME" --tail=20 || echo "  (failed to retrieve final logs)"
        echo "-----------------------------------------------------"
        exit 1
    fi

    # echo "-----------------------------------------------------" # Reduced verbosity
    echo "Attempting log check on $POD/$CONTAINER_NAME (Elapsed: ${ELAPSED_TIME}s / ${RETRY_TIMEOUT}s)"

    # 2. Get Logs for this attempt from the SPECIFIC container
    # Using --tail=-1 to get all logs. Consider limiting if logs are huge and only recent ones matter.
    LOGS=$(kubectl logs "$POD" -n "$NAMESPACE" -c "$CONTAINER_NAME" --tail=-1 --request-timeout=10s)
    KUBECTL_LOGS_EXIT_CODE=$?

    if [ $KUBECTL_LOGS_EXIT_CODE -ne 0 ]; then
         echo "Warning: Failed to get logs for container '$CONTAINER_NAME' in pod '$POD' (Exit code: $KUBECTL_LOGS_EXIT_CODE). Retrying after sleep..."
         sleep "$RETRY_SLEEP"
         continue # Go to next loop iteration
    fi

    # It's possible logs are valid but empty initially, so don't exit, just report.
    # if [ -z "$LOGS" ]; then
    #      echo "Warning: Logs for container '$CONTAINER_NAME' in pod '$POD' were empty on this attempt. Retrying after sleep..."
    #      sleep "$RETRY_SLEEP"
    #      continue # Go to next loop iteration
    # fi

    # 3. Check for all strings in the current logs
    ALL_FOUND_THIS_ATTEMPT=true
    MISSING_STRINGS_THIS_ATTEMPT=() # Reset missing strings for this attempt
    for STRING in "${SEARCH_STRINGS[@]}"; do
        # Use grep -F (fixed string) and -q (quiet) for efficiency
        if ! echo "$LOGS" | grep -Fq -- "$STRING"; then
            ALL_FOUND_THIS_ATTEMPT=false
            MISSING_STRINGS_THIS_ATTEMPT+=("$STRING")
            # Optimization: uncomment break if you only need to know *if* something is missing,
            # not the full list of missing items each time.
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
        # Report missing strings only if the list changed since last time or on the first attempt
        CURRENT_MISSING_COUNT=${#MISSING_STRINGS_THIS_ATTEMPT[@]}
        # Use -ne for numeric comparison
        if [ "$CURRENT_MISSING_COUNT" -ne "$LAST_MISSING_COUNT" ] || [ "$ELAPSED_TIME" -lt "$RETRY_SLEEP" ]; then # Log on first effective attempt too
             echo "Attempt Failed: Missing ${CURRENT_MISSING_COUNT} string(s):"
             printf "  - '%s'\n" "${MISSING_STRINGS_THIS_ATTEMPT[@]}" # Print missing strings, quoted
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