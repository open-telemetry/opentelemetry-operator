#!/bin/bash

region="us-east-2"

# Fetch AWS credentials based on CLUSTER_PROFILE_DIR variable
if [ -n "${CLUSTER_PROFILE_DIR}" ]; then
    AWS_ACCESS_KEY_ID=$(grep "aws_access_key_id=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
    AWS_SECRET_ACCESS_KEY=$(grep "aws_secret_access_key=" "${CLUSTER_PROFILE_DIR}/.awscred" | cut -d '=' -f2)
elif oc get secret aws-creds -n kube-system >/dev/null 2>&1; then
    AWS_ACCESS_KEY_ID=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_access_key_id' | base64 -d)
    if [ $? -ne 0 ]; then
        echo "Failed to fetch AWS_ACCESS_KEY_ID"
        exit 1
    fi

    AWS_SECRET_ACCESS_KEY=$(oc get secret aws-creds -n kube-system -o json | jq -r '.data.aws_secret_access_key' | base64 -d)
    if [ $? -ne 0 ]; then
        echo "Failed to fetch AWS_SECRET_ACCESS_KEY"
        exit 1
    fi
elif aws sts get-caller-identity >/dev/null 2>&1; then
    # Check if we're already logged into AWS and can get credentials
    AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id)
    AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key)
    
    if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
        echo "AWS CLI is configured but cannot retrieve access key credentials (may be using IAM roles or other auth method)"
        echo "Please ensure AWS credentials are available as access key ID and secret access key"
        exit 1
    fi
    
    echo "Using AWS credentials from AWS CLI configuration"
else
    echo "No AWS credentials found. Please ensure one of the following:"
    echo "1. CLUSTER_PROFILE_DIR is set with .awscred file"
    echo "2. aws-creds secret exists in kube-system namespace"
    echo "3. AWS CLI is configured with valid credentials"
    exit 1
fi

export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
export AWS_DEFAULT_REGION=${region}

# Check if retentionInDays is 365
RETENTION_DAYS=$(aws logs describe-log-groups --log-group-name-prefix $log_group_name --region $region --query "logGroups[?logGroupName=='$log_group_name'].retentionInDays" --output text)
if [ "$RETENTION_DAYS" == "1" ]; then
    echo "retentionInDays is 1"
else
    echo "retentionInDays is not 1"
    exit 1
fi

# Check if the first log record contains the specified string
MESSAGE=$(aws logs get-log-events --log-group-name $log_group_name --log-stream-name $log_stream_name --region $region --no-paginate --query "events[0].message" --output text)
if [[ "$MESSAGE" == *"SVTLogger - INFO - app-log-plaintext-rc-"* ]]; then
    echo "App logs found in CloudWatchLogs"
else
    echo "No app logs found in CloudWatchLogs"
    exit 1
fi

# Run the AWS CloudWatch command and store the result
result=$(aws cloudwatch list-metrics --namespace Tracing-EMF --endpoint https://monitoring.us-east-2.amazonaws.com --region=us-east-2 --dimensions Name=telemetrygen,Value=metrics)

# Check if the result contains the specific metric
if echo "$result" | grep -q '"Name": "telemetrygen",'; then
    echo "The metric with dimension Name 'telemetrygen' and value 'metrics' exists. Script succeeded."
else
    echo "The metric with dimension Name 'telemetrygen' and value 'metrics' does not exist. Script failed."
    exit 1
fi


# Delete the log group
aws logs delete-log-group --log-group-name $log_group_name --region $region
if [ $? -ne 0 ]; then
    echo "Failed to delete log group"
    exit 1
else
    echo "Log group deleted successfully"
fi
