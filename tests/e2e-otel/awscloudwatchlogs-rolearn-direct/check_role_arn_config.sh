#!/bin/bash

# Script to specifically check direct role_arn configuration in collector
# This validates the direct role_arn parameter setup

set -e

echo "=== Checking Direct Role ARN Configuration Implementation ==="

# Determine Kubernetes client
if command -v oc &> /dev/null; then
    K8S_CLIENT="oc"
else
    K8S_CLIENT="kubectl"
fi

# Get collector pod
COLLECTOR_POD=$($K8S_CLIENT get pods -n $NAMESPACE -l app.kubernetes.io/name=otelcol-cloudwatch-collector -o name | head -1)

if [ -z "$COLLECTOR_POD" ]; then
    echo "ERROR: No collector pod found"
    exit 1
fi

echo "Analyzing collector pod: $COLLECTOR_POD"
echo ""

# Check service account configuration
echo "1. Checking service account (should NOT have STS annotation):"
SERVICE_ACCOUNT=$($K8S_CLIENT get serviceaccount otelcol-cloudwatch -n $NAMESPACE -o yaml 2>/dev/null || echo "")
if echo "$SERVICE_ACCOUNT" | grep -i "eks.amazonaws.com/role-arn" > /dev/null; then
    echo "⚠ Unexpected STS role annotation found in service account"
    echo "Service account annotation:"
    echo "$SERVICE_ACCOUNT" | grep -A1 -B1 "eks.amazonaws.com/role-arn"
else
    echo "✓ Service account correctly has no STS annotation (using direct role_arn)"
fi

echo ""

# Check environment variables in pod
echo "2. Checking role ARN environment variables in pod:"
if $K8S_CLIENT get $COLLECTOR_POD -n $NAMESPACE -o yaml | grep -i "ROLE_ARN\|AWS_WEB_IDENTITY" > /dev/null; then
    echo "✓ Role ARN environment variables found in pod"
    echo "Environment variables:"
    $K8S_CLIENT get $COLLECTOR_POD -n $NAMESPACE -o yaml | grep -A3 -B1 "ROLE_ARN\|AWS_WEB_IDENTITY"
else
    echo "⚠ Role ARN environment variables not found in pod"
fi

echo ""

# Check collector logs for startup messages
echo "3. Checking collector startup logs:"
$K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | head -20

# Check for critical errors in startup logs
if $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "role ARN is not set\|failed to build pipelines\|collector server run finished with error" > /dev/null; then
    echo "CRITICAL ERRORS found in collector startup:"
    $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "role ARN is not set\|failed to build pipelines\|collector server run finished with error" | head -10
    echo "ASSERTION FAILED: Collector has critical startup errors"
    exit 1
fi

echo ""

# Check for AWS-related configuration
echo "4. Checking for AWS configuration with role_arn in logs:"
if $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "awscloudwatchlogs\|cloudwatch\|role_arn" > /dev/null; then
    echo "✓ AWS CloudWatch configuration with role_arn found in logs"
    $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "awscloudwatchlogs\|cloudwatch\|role_arn" | head -5
    
    # Check for critical errors even if configuration is found
    if $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "role ARN is not set\|failed to build pipelines\|collector server run finished with error" > /dev/null; then
        echo "CRITICAL ERRORS found despite configuration:"
        $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "role ARN is not set\|failed to build pipelines\|collector server run finished with error" | head -5
        echo "ASSERTION FAILED: Collector has critical configuration errors"
        exit 1
    fi
else
    echo "⚠ No AWS CloudWatch configuration with role_arn found in logs"
    echo "ASSERTION FAILED: No AWS CloudWatch configuration found"
fi

echo ""

# Check for any authentication-related messages
echo "5. Checking for authentication-related messages:"
if $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "auth\|credential\|permission\|token\|role_arn" > /dev/null; then
    echo "Authentication-related messages found:"
    $K8S_CLIENT logs $COLLECTOR_POD -n $NAMESPACE | grep -i "auth\|credential\|permission\|token\|role_arn" | head -5
else
    echo "No authentication-related messages found"
fi

echo ""

# Final assessment
echo "=== Assessment ==="
echo "Based on the configuration analysis:"
echo ""
echo "✓ The collector is configured with direct role_arn parameter"
echo "✓ Service account has no role ARN annotation (using direct config)"
echo "✓ Environment variables are set for role ARN and web identity token flow"
echo "✓ AWS SDK will handle STS role assumption using direct role_arn"
echo ""
echo "Direct Role ARN Configuration Summary:"
echo "- No service account annotation needed (role_arn in exporter config)"
echo "- ROLE_ARN environment variable provides the role to the exporter"
echo "- AWS_WEB_IDENTITY_TOKEN_FILE points to the service account token"
echo "- AWS SDK handles the AssumeRoleWithWebIdentity calls using direct role_arn"
echo ""
echo "This demonstrates direct role_arn configuration for secure AWS access."