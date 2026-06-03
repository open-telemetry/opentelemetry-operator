# OpenTelemetry AWS CloudWatch Logs Exporter with STS Role Assumption Test

This test demonstrates AWS STS (Security Token Service) role assumption for the OpenTelemetry AWS CloudWatch Logs exporter using Kubernetes service account annotations.

## 🎯 What This Test Does

This test validates and demonstrates:
- ✅ STS role assumption using Kubernetes service account annotations
- ✅ AWS Web Identity Token authentication flow
- ✅ OpenTelemetry Collector deployment with STS configuration
- ✅ Log delivery to AWS CloudWatch using assumed role credentials
- ✅ Automatic credential management via AWS SDK

## 🔐 Authentication Method: STS Web Identity Token

This test uses the **AWS STS AssumeRoleWithWebIdentity** flow:

1. **Service Account Annotation** - Kubernetes service account annotated with IAM role ARN
2. **Web Identity Token** - Kubernetes provides JWT token at `/var/run/secrets/eks.amazonaws.com/serviceaccount/token`
3. **Automatic Role Assumption** - AWS SDK automatically exchanges token for temporary credentials
4. **CloudWatch Access** - Temporary credentials used to write logs to CloudWatch

## 📋 Test Resources

### Key Files in This Directory

1. **[`chainsaw-test.yaml`](./chainsaw-test.yaml)** - Chainsaw test orchestration
2. **[`otel-collector-rolearn.yaml`](./otel-collector-rolearn.yaml)** - Collector with STS configuration
3. **[`create-aws-rolearn-secret.sh`](./create-aws-rolearn-secret.sh)** - AWS IAM role and secret creation
4. **[`check_logs_rolearn.sh`](./check_logs_rolearn.sh)** - Main verification script
5. **[`check_role_arn_config.sh`](./check_role_arn_config.sh)** - STS configuration validation
6. **[`aws-sts-cloudwatch-delete.sh`](./aws-sts-cloudwatch-delete.sh)** - AWS resource cleanup
7. **[`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)** - Log generator application

## 🚀 How to Run This Test

```bash
# Set kubeconfig (required for OpenShift/Kubernetes cluster access)
export KUBECONFIG=~/path/to/kubeconfig

# Run the STS role assumption test
chainsaw test --test-dir tests/e2e-otel/awscloudwatchlogs-rolearn/

# Run with specific namespace (optional)
chainsaw test --test-dir tests/e2e-otel/awscloudwatchlogs-rolearn/ --namespace my-test-ns
```

## 🔧 Test Configuration

### Service Account with STS Annotation
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otelcol-cloudwatch
  namespace: chainsaw-awssts-cloudwatch
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::ACCOUNT:role/ROLE_NAME"
```

### Environment Variables for STS
```yaml
env:
  - name: AWS_REGION
    valueFrom:
      secretKeyRef:
        name: aws-sts-cloudwatch
        key: region
  - name: AWS_ROLE_ARN
    valueFrom:
      secretKeyRef:
        name: aws-sts-cloudwatch
        key: role_arn
  - name: AWS_WEB_IDENTITY_TOKEN_FILE
    value: /var/run/secrets/eks.amazonaws.com/serviceaccount/token
```

### CloudWatch Logs Exporter Configuration
```yaml
exporters:
  awscloudwatchlogs:
    log_group_name: "${env:LOG_GROUP_NAME}"
    log_stream_name: "tracing-otelcol-stream"
    raw_log: true
    region: "${env:AWS_REGION}"
    endpoint: "https://logs.us-east-2.amazonaws.com"
    log_retention: 1
    tags: { 'tracing-otel': 'true', 'test-type': 'aws-sts' }
```

## 🔍 What the Test Validates

### 1. AWS IAM Setup
- ✅ CloudWatch log group creation
- ✅ IAM role creation with proper trust policy
- ✅ IAM policy attachment for CloudWatch Logs access
- ✅ Kubernetes secret creation with role ARN

### 2. STS Configuration
- ✅ Service account annotation with role ARN
- ✅ Environment variables for STS authentication
- ✅ Web identity token file mount
- ✅ Collector deployment with STS service account

### 3. Authentication Flow
- ✅ AWS SDK automatic role assumption
- ✅ Temporary credential generation
- ✅ CloudWatch API access with assumed role
- ✅ Log delivery to CloudWatch

### 4. End-to-End Verification
- ✅ OpenTelemetry Collector pod startup
- ✅ Log export to AWS CloudWatch
- ✅ CloudWatch log group and stream validation
- ✅ Log content verification

## 📊 Expected Test Results

```bash
=== AWS CloudWatch Logs STS Test Verification ===
Log Group: tracing-chainsaw-awssts-cloudwatch-ciotelcwl
Region: us-east-2
Role ARN: arn:aws:iam::301721915996:role/tracing-cloudwatch-chainsaw-awssts-cloudwatch-ciotelcwl

✓ Collector pod is running successfully
✓ STS environment variables found in pod
✓ No AWS-related errors found in collector logs
✓ Log group found in CloudWatch
✓ Log streams found in CloudWatch
✓ Log events found in CloudWatch

=== Test Verification Complete ===
This test successfully demonstrates:
✓ STS role assumption configuration
✓ Collector startup with STS environment variables
✓ CloudWatch Logs exporter using assumed role credentials

STS integration enables secure, temporary access to AWS services
without embedding long-term credentials in the collector configuration.
```

## 🔄 STS Authentication Flow

```
1. Kubernetes Service Account
   ↓ (annotated with role ARN)
2. Web Identity Token (JWT)
   ↓ (mounted at token file path)
3. AWS SDK STS AssumeRoleWithWebIdentity
   ↓ (automatic token exchange)
4. Temporary AWS Credentials
   ↓ (access key, secret key, session token)
5. CloudWatch Logs API Access
   ↓ (using temporary credentials)
6. Log Delivery to CloudWatch
```

## 🛡️ Security Benefits

1. **No Long-term Credentials** - Uses temporary credentials only
2. **Automatic Rotation** - AWS SDK handles credential refresh
3. **Principle of Least Privilege** - IAM role with minimal permissions
4. **Audit Trail** - All API calls logged with assumed role identity
5. **Cross-Account Support** - Can assume roles in different AWS accounts

## 🔧 Prerequisites

### AWS Environment
- AWS account with IAM permissions
- CloudWatch Logs service access
- STS (Security Token Service) enabled

### Kubernetes/OpenShift Cluster
- Service account token projection support
- Network access to AWS APIs
- Proper RBAC for OpenTelemetry Operator

### Test Environment Variables
- `OPENSHIFT_BUILD_NAMESPACE` - Used for resource naming
- `KUBECONFIG` - Kubernetes cluster access

## 🧹 Cleanup

The test automatically cleans up AWS resources in the cleanup phase:

```bash
# Manual cleanup if needed
./aws-sts-cloudwatch-delete.sh otelcol-cloudwatch chainsaw-awssts-cloudwatch
```

This removes:
- IAM role and policy
- CloudWatch log group
- Kubernetes secret

## 🐛 Troubleshooting

### Common Issues

1. **Pod Not Found Error**
   - Check pod label selector in verification scripts
   - Ensure correct namespace is used

2. **STS Permission Denied**
   - Verify IAM role trust policy allows web identity
   - Check service account annotation format

3. **CloudWatch Access Denied**
   - Verify IAM policy includes CloudWatch Logs permissions
   - Check log group name and region configuration

4. **Token File Not Found**
   - Ensure service account token projection is enabled
   - Verify token file path in environment variables

### Debug Commands

```bash
# Check collector pod status
oc get pods -n chainsaw-awssts-cloudwatch -l app.kubernetes.io/name=otelcol-cloudwatch-collector

# View collector logs
oc logs -n chainsaw-awssts-cloudwatch deployment/otelcol-cloudwatch-collector

# Check service account annotation
oc get serviceaccount otelcol-cloudwatch -n chainsaw-awssts-cloudwatch -o yaml

# Verify secret contents
oc get secret aws-sts-cloudwatch -n chainsaw-awssts-cloudwatch -o yaml
```

## 🔗 Related Documentation

- [AWS STS AssumeRoleWithWebIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html)
- [OpenTelemetry AWS CloudWatch Logs Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/awscloudwatchlogsexporter)
- [Kubernetes Service Account Token Projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#serviceaccount-token-volume-projection)

## 🏷️ Test Categories

- **Component**: AWS CloudWatch Logs Exporter  
- **Feature**: STS Role Assumption
- **Status**: ✅ Fully Implemented and Working
- **Environment**: OpenShift/Kubernetes + AWS
- **Authentication**: AWS STS Web Identity Token