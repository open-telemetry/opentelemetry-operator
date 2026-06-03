#!/bin/bash
set -euo pipefail

# This script creates AWS STS configuration for CloudWatch Logs testing
# Modeled after tempo-operator STS configuration
# Run the script with ./create-aws-rolearn-secret.sh COLLECTOR_NAME NAMESPACE

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

# Create CloudWatch log group name based on namespace and build
log_group_name="tracing-$namespace-$OPENSHIFT_BUILD_NAMESPACE"

# Create a CloudWatch log group
echo "Creating CloudWatch log group: $log_group_name"
aws logs create-log-group --log-group-name "$log_group_name" --region "$region" || echo "Log group may already exist"

# Set required vars to create AWS IAM policy and role
oidc_provider=$(oc get authentication cluster -o json | jq -r '.spec.serviceAccountIssuer' | sed 's~http[s]*://~~g')
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
cluster_id=$(oc get clusterversion -o jsonpath='{.items[].spec.clusterID}{"\n"}')
trust_rel_file="/tmp/$namespace-cloudwatch-trust.json"
role_name="tracing-cloudwatch-$namespace-$OPENSHIFT_BUILD_NAMESPACE"

# Create a trust relationship file for CloudWatch Logs
cat > "$trust_rel_file" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${aws_account_id}:oidc-provider/${oidc_provider}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${oidc_provider}:sub": "system:serviceaccount:${namespace}:${collector_name}"
        }
      }
    }
  ]
}
EOF

echo "Creating IAM role '$role_name'..."
role_arn=$(aws iam create-role \
             --role-name "$role_name" \
             --assume-role-policy-document "file://$trust_rel_file" \
             --query Role.Arn \
             --output text)

# Create custom CloudWatch Logs policy
policy_name="CloudWatchLogsPolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
policy_file="/tmp/$namespace-cloudwatch-policy.json"

cat > "$policy_file" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:${region}:${aws_account_id}:log-group:${log_group_name}*"
    }
  ]
}
EOF

echo "Creating IAM policy '$policy_name'..."
policy_arn=$(aws iam create-policy \
             --policy-name "$policy_name" \
             --policy-document "file://$policy_file" \
             --query Policy.Arn \
             --output text)

echo "Attaching policy '$policy_name' to role '$role_name'..."
aws iam attach-role-policy \
  --role-name "$role_name" \
  --policy-arn "$policy_arn"

echo "Role created and policy attached successfully!"

echo "Create the secret to be used with OpenTelemetry Collector"
oc -n $namespace create secret generic aws-sts-cloudwatch \
  --from-literal=log_group_name="$log_group_name" \
  --from-literal=region="$region" \
  --from-literal=role_arn="$role_arn"

echo "AWS STS configuration for CloudWatch Logs completed successfully!"
echo "Log Group: $log_group_name"
echo "Role ARN: $role_arn"
echo "Region: $region"