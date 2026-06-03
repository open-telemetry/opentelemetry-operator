#!/bin/bash
set -euo pipefail

# This script cleans up AWS STS configuration for CloudWatch Logs testing
# Modeled after tempo-operator STS cleanup
# Run the script with ./aws-sts-cloudwatch-delete.sh COLLECTOR_NAME NAMESPACE

# Check if OPENSHIFT_BUILD_NAMESPACE is unset or empty
if [ -z "${OPENSHIFT_BUILD_NAMESPACE+x}" ]; then
    OPENSHIFT_BUILD_NAMESPACE="ciotelcwl"
    export OPENSHIFT_BUILD_NAMESPACE
fi

echo "OPENSHIFT_BUILD_NAMESPACE is set to: $OPENSHIFT_BUILD_NAMESPACE"

if [ -z "${CLUSTER_PROFILE_DIR+x}" ]; then
    echo "Warning: CLUSTER_PROFILE_DIR is not set, proceeding without it..."
else
    export AWS_SHARED_CREDENTIALS_FILE="$CLUSTER_PROFILE_DIR/.awscred"
    echo "AWS_SHARED_CREDENTIALS_FILE is set to: $AWS_SHARED_CREDENTIALS_FILE"
fi

export AWS_PAGER=""
region=us-east-2

# Use parameters if provided, otherwise use environment/defaults
collector_name="${1:-otelcol-cloudwatch}"
namespace="${2:-${NAMESPACE:-chainsaw-awscloudwatchlogs}}"

# Generate names to match creation script
log_group_name="tracing-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
role_name="tracing-cloudwatch-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
policy_name="CloudWatchLogsPolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"

echo "Cleaning up AWS resources..."

# Get AWS account ID
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
policy_arn="arn:aws:iam::${aws_account_id}:policy/${policy_name}"

# Detach and delete the IAM policy
echo "Detaching policy '$policy_name' from role '$role_name'..."
aws iam detach-role-policy \
  --role-name "$role_name" \
  --policy-arn "$policy_arn" || echo "Policy may not be attached"

echo "Deleting IAM policy '$policy_name'..."
aws iam delete-policy \
  --policy-arn "$policy_arn" || echo "Policy may not exist"

# Delete the IAM role
echo "Deleting IAM role '$role_name'..."
aws iam delete-role \
  --role-name "$role_name" || echo "Role may not exist"

# Delete the CloudWatch log group
echo "Deleting CloudWatch log group '$log_group_name'..."
aws logs delete-log-group \
  --log-group-name "$log_group_name" \
  --region "$region" || echo "Log group may not exist"

# Clean up temporary files
rm -f "/tmp/$namespace-cloudwatch-trust.json"
rm -f "/tmp/$namespace-cloudwatch-policy.json"

echo "AWS STS CloudWatch cleanup completed successfully!"