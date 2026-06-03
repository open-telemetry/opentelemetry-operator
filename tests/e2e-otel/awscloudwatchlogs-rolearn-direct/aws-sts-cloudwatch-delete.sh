#!/bin/bash
set -euo pipefail

# This script cleans up AWS STS configuration for CloudWatch Logs testing with direct role_arn
# This cleanup handles TWO roles (base and target) and their associated policies
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
base_role_name="cw-base-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
target_role_name="cw-target-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
base_policy_name="CWBasePolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
target_policy_name="CWTargetPolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"

echo "Cleaning up AWS resources..."

# Get AWS account ID
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
base_policy_arn="arn:aws:iam::${aws_account_id}:policy/${base_policy_name}"
target_policy_arn="arn:aws:iam::${aws_account_id}:policy/${target_policy_name}"

# ======================
# Clean up BASE ROLE
# ======================
echo "Detaching base policy '$base_policy_name' from base role '$base_role_name'..."
aws iam detach-role-policy \
  --role-name "$base_role_name" \
  --policy-arn "$base_policy_arn" 2>/dev/null || echo "Base policy may not be attached"

# Delete all non-default policy versions for base policy
echo "Deleting non-default base policy versions..."
aws iam list-policy-versions \
  --policy-arn "$base_policy_arn" \
  --query 'Versions[?IsDefaultVersion==`false`].VersionId' \
  --output text 2>/dev/null | tr '\t' '\n' | while read version_id; do
  if [ -n "$version_id" ]; then
    aws iam delete-policy-version \
      --policy-arn "$base_policy_arn" \
      --version-id "$version_id" 2>/dev/null || true
  fi
done

echo "Deleting base IAM policy '$base_policy_name'..."
aws iam delete-policy \
  --policy-arn "$base_policy_arn" 2>/dev/null || echo "Base policy may not exist"

echo "Deleting base IAM role '$base_role_name'..."
aws iam delete-role \
  --role-name "$base_role_name" 2>/dev/null || echo "Base role may not exist"

# ======================
# Clean up TARGET ROLE
# ======================
echo "Detaching target policy '$target_policy_name' from target role '$target_role_name'..."
aws iam detach-role-policy \
  --role-name "$target_role_name" \
  --policy-arn "$target_policy_arn" 2>/dev/null || echo "Target policy may not be attached"

# Delete all non-default policy versions for target policy
echo "Deleting non-default target policy versions..."
aws iam list-policy-versions \
  --policy-arn "$target_policy_arn" \
  --query 'Versions[?IsDefaultVersion==`false`].VersionId' \
  --output text 2>/dev/null | tr '\t' '\n' | while read version_id; do
  if [ -n "$version_id" ]; then
    aws iam delete-policy-version \
      --policy-arn "$target_policy_arn" \
      --version-id "$version_id" 2>/dev/null || true
  fi
done

echo "Deleting target IAM policy '$target_policy_name'..."
aws iam delete-policy \
  --policy-arn "$target_policy_arn" 2>/dev/null || echo "Target policy may not exist"

echo "Deleting target IAM role '$target_role_name'..."
aws iam delete-role \
  --role-name "$target_role_name" 2>/dev/null || echo "Target role may not exist"

# ======================
# Clean up CloudWatch and temp files
# ======================
echo "Deleting CloudWatch log group '$log_group_name'..."
aws logs delete-log-group \
  --log-group-name "$log_group_name" \
  --region "$region" 2>/dev/null || echo "Log group may not exist"

# Clean up temporary files
rm -f "/tmp/$namespace-cloudwatch-base-trust.json"
rm -f "/tmp/$namespace-cloudwatch-base-policy.json"
rm -f "/tmp/$namespace-cloudwatch-target-trust.json"
rm -f "/tmp/$namespace-cloudwatch-target-policy.json"

echo "AWS STS CloudWatch cleanup completed successfully!"