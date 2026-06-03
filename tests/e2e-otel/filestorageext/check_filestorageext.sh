#!/bin/bash

# Define the directories to check
directories=("/filestorageext/compaction" "/filestorageext/data")

# Define the file to check
file="receiver_filelog_"

# Initialize a variable to keep track of the number of files found
files_found=0

# Keep running the loop until all files are found
while [ "$files_found" -ne "${#directories[@]}" ]; do
  # Reset the counter
  files_found=0

  # Loop through the directories
  for dir in "${directories[@]}"; do
    # Use oc exec to run ls in the directory and grep for the file
    if oc exec -n $NAMESPACE replicationcontrollers/app-log-plaintext-rc -- ls "$dir" | grep -q "$file"; then
      echo "File $file found in $dir"
      ((files_found++))
    else
      echo "File $file not found in $dir"
    fi
  done

  # Sleep for a while before the next iteration
  sleep 2
done

echo "All files found!"