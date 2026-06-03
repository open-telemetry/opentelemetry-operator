#!/bin/bash
set -euo pipefail

# This script creates AWS STS configuration for CloudWatch Logs testing with direct role_arn
# This test uses TWO roles to properly test the exporter's role_arn parameter:
#   1. Base role: Authenticated via web identity token
#   2. Target role: Assumed by the exporter using role_arn parameter
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

# Set required vars to create AWS IAM policies and roles
oidc_provider=$(oc get authentication cluster -o json | jq -r '.spec.serviceAccountIssuer' | sed 's~http[s]*://~~g')
aws_account_id=$(aws sts get-caller-identity --query 'Account' --output text)
cluster_id=$(oc get clusterversion -o jsonpath='{.items[].spec.clusterID}{"\n"}')

# Role names (shortened to stay under AWS 64 char limit)
base_role_name="cw-base-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
target_role_name="cw-target-$namespace-$OPENSHIFT_BUILD_NAMESPACE"

# ======================
# Create BASE ROLE (for web identity authentication)
# ======================
echo "Creating base IAM role '$base_role_name' for web identity authentication..."
base_trust_file="/tmp/$namespace-cloudwatch-base-trust.json"
cat > "$base_trust_file" <<EOF
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

base_role_arn=$(aws iam create-role \
             --role-name "$base_role_name" \
             --assume-role-policy-document "file://$base_trust_file" \
             --query Role.Arn \
             --output text)

echo "Base role created: $base_role_arn"

# Attach minimal policy to base role (just STS permissions)
base_policy_name="CWBasePolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
base_policy_file="/tmp/$namespace-cloudwatch-base-policy.json"
cat > "$base_policy_file" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "arn:aws:iam::${aws_account_id}:role/${target_role_name}"
    }
  ]
}
EOF

echo "Creating base role policy '$base_policy_name'..."
base_policy_arn=$(aws iam create-policy \
             --policy-name "$base_policy_name" \
             --policy-document "file://$base_policy_file" \
             --query Policy.Arn \
             --output text)

echo "Attaching policy to base role..."
aws iam attach-role-policy \
  --role-name "$base_role_name" \
  --policy-arn "$base_policy_arn"

# Wait for IAM propagation before creating target role
echo "Waiting for IAM propagation..."
sleep 5

# ======================
# Create TARGET ROLE (to be assumed by exporter's role_arn)
# ======================
echo "Creating target IAM role '$target_role_name' for exporter to assume..."
target_trust_file="/tmp/$namespace-cloudwatch-target-trust.json"
cat > "$target_trust_file" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::${aws_account_id}:role/${base_role_name}"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

target_role_arn=$(aws iam create-role \
             --role-name "$target_role_name" \
             --assume-role-policy-document "file://$target_trust_file" \
             --query Role.Arn \
             --output text)

echo "Target role created: $target_role_arn"

# Attach CloudWatch Logs policy to TARGET role
target_policy_name="CWTargetPolicy-$namespace-$OPENSHIFT_BUILD_NAMESPACE"
target_policy_file="/tmp/$namespace-cloudwatch-target-policy.json"

cat > "$target_policy_file" <<EOF
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

echo "Creating CloudWatch Logs policy '$target_policy_name'..."
target_policy_arn=$(aws iam create-policy \
             --policy-name "$target_policy_name" \
             --policy-document "file://$target_policy_file" \
             --query Policy.Arn \
             --output text)

echo "Attaching CloudWatch policy to target role..."
aws iam attach-role-policy \
  --role-name "$target_role_name" \
  --policy-arn "$target_policy_arn"

echo "Roles created and policies attached successfully!"

echo "Create the secret to be used with OpenTelemetry Collector"
oc -n $namespace create secret generic aws-sts-cloudwatch \
  --from-literal=log_group_name="$log_group_name" \
  --from-literal=region="$region" \
  --from-literal=base_role_arn="$base_role_arn" \
  --from-literal=target_role_arn="$target_role_arn"

echo "AWS STS configuration for CloudWatch Logs completed successfully!"
echo "Log Group: $log_group_name"
echo "Base Role ARN (for web identity): $base_role_arn"
echo "Target Role ARN (for exporter): $target_role_arn"
echo "Region: $region"