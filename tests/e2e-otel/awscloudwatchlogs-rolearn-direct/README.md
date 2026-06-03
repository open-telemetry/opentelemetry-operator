# OpenTelemetry AWS CloudWatch Logs Exporter with Direct Role ARN Configuration Test

This test demonstrates AWS STS (Security Token Service) **role chaining** for the OpenTelemetry AWS CloudWatch Logs exporter using the **exporter's `role_arn` parameter** for cross-account/role assumption scenarios.

## 🎯 What This Test Does

This test validates and demonstrates:
- ✅ **Two-role architecture** for proper role chaining
- ✅ **Base role** authentication via web identity token (OIDC federation)
- ✅ **Target role** assumption via exporter's `role_arn` parameter
- ✅ Exporter's `role_arn` parameter functionality for role chaining
- ✅ Log delivery to AWS CloudWatch using assumed target role credentials
- ✅ Service account without annotations (credentials via projected token volume)

## 🔐 Authentication Method: Two-Role Chain with Web Identity Token

This test uses a **two-role architecture** to properly test the exporter's `role_arn` parameter:

### Role 1: Base Role (Web Identity Authentication)
1. **No Service Account Annotation** - Service account has NO role ARN annotation
2. **Projected Token Volume** - Kubernetes service account token with `sts.amazonaws.com` audience
3. **Web Identity Token** - JWT token at `/var/run/secrets/eks.amazonaws.com/serviceaccount/token`
4. **Base Role Assumption** - AWS SDK assumes base role via `AssumeRoleWithWebIdentity`
5. **Base Role Permission** - Has permission to assume the target role

### Role 2: Target Role (Exporter's role_arn Parameter)
1. **Exporter Configuration** - `role_arn` parameter specifies target role ARN
2. **Role Chaining** - Exporter assumes target role using base role credentials
3. **Target Role Trust Policy** - Trusts the base role to assume it
4. **CloudWatch Access** - Target role has CloudWatch Logs permissions
5. **Log Delivery** - Logs written to CloudWatch using target role credentials

## 🔄 Key Differences from Standard STS Configuration

### Service Account Configuration
- **Standard STS** (`awscloudwatchlogs-rolearn`): Service account has `eks.amazonaws.com/role-arn` annotation
- **This Test**: Service account has **NO** annotation, uses projected token volume

### Role Architecture
- **Standard STS**: Single role (via service account annotation)
- **This Test**: **Two roles** - base role (web identity) + target role (exporter's `role_arn`)

### Authentication Flow
- **Standard STS**: Direct web identity → role → CloudWatch
- **This Test**: Web identity → base role → **assume target role** → CloudWatch

### Exporter Configuration
```yaml
env:
  - name: AWS_ROLE_ARN
    value: "arn:aws:iam::ACCOUNT:role/BASE_ROLE"  # Base role for web identity
  - name: AWS_WEB_IDENTITY_TOKEN_FILE
    value: /var/run/secrets/eks.amazonaws.com/serviceaccount/token

exporters:
  awscloudwatchlogs:
    log_group_name: "${env:LOG_GROUP_NAME}"
    region: "${env:AWS_REGION}"
    role_arn: "arn:aws:iam::ACCOUNT:role/TARGET_ROLE"  # ← Target role via exporter parameter
    # ... other configuration
```

## 📋 Test Resources

### Key Files in This Directory

1. **[`chainsaw-test.yaml`](./chainsaw-test.yaml)** - Chainsaw test orchestration (namespace: `chainsaw-awssts-cloudwatch-direct`)
2. **[`otel-collector-rolearn.yaml`](./otel-collector-rolearn.yaml)** - Collector with direct role_arn configuration
3. **[`create-aws-rolearn-secret.sh`](./create-aws-rolearn-secret.sh)** - AWS IAM role and secret creation
4. **[`check_logs_rolearn.sh`](./check_logs_rolearn.sh)** - Main verification script for direct role_arn
5. **[`check_role_arn_config.sh`](./check_role_arn_config.sh)** - Direct role_arn configuration validation
6. **[`aws-sts-cloudwatch-delete.sh`](./aws-sts-cloudwatch-delete.sh)** - AWS resource cleanup
7. **[`app-plaintext-logs.yaml`](./app-plaintext-logs.yaml)** - Log generator application

## 🚀 How to Run This Test

```bash
# Set kubeconfig (required for OpenShift/Kubernetes cluster access)
export KUBECONFIG=~/path/to/kubeconfig

# Run the direct role ARN test
chainsaw test --test-dir tests/e2e-otel/awscloudwatchlogs-rolearn-direct/

# Run with specific namespace (optional)
chainsaw test --test-dir tests/e2e-otel/awscloudwatchlogs-rolearn-direct/ --namespace my-test-ns
```

## 🔧 Test Configuration

### Service Account WITHOUT STS Annotation
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otelcol-cloudwatch
  namespace: chainsaw-awssts-cloudwatch-direct
  # NO eks.amazonaws.com/role-arn annotation!
```

### Environment Variables for Two-Role Setup
```yaml
env:
  - name: AWSREGION
    valueFrom:
      secretKeyRef:
        name: aws-sts-cloudwatch
        key: region
  - name: LOGGROUPNAME
    valueFrom:
      secretKeyRef:
        name: aws-sts-cloudwatch
        key: log_group_name
  - name: AWS_ROLE_ARN  # ← BASE ROLE for web identity authentication
    valueFrom:
      secretKeyRef:
        name: aws-sts-cloudwatch
        key: base_role_arn
  - name: AWS_WEB_IDENTITY_TOKEN_FILE
    value: /var/run/secrets/eks.amazonaws.com/serviceaccount/token
```

### Projected Token Volume for STS Audience
```yaml
volumes:
  - name: aws-iam-token
    projected:
      sources:
        - serviceAccountToken:
            audience: sts.amazonaws.com  # ← Required for AWS STS
            expirationSeconds: 86400
            path: token
volumeMounts:
  - name: aws-iam-token
    mountPath: /var/run/secrets/eks.amazonaws.com/serviceaccount
    readOnly: true
```

### CloudWatch Logs Exporter with Target Role ARN
```yaml
exporters:
  awscloudwatchlogs:
    log_group_name: "${env:LOGGROUPNAME}"
    log_stream_name: "tracing-otelcol-stream"
    raw_log: true
    region: "${env:AWSREGION}"
    role_arn: "arn:aws:iam::123456789012:role/invalid-role-for-testing"  # ← TARGET ROLE (starts invalid for phase 1)
    # Test patches this to correct target_role_arn in phase 2
    log_retention: 1
    tags: { 'tracing-otel': 'true', 'test-type': 'aws-sts-direct' }
```

## 🔍 What the Test Validates

### 1. AWS IAM Setup (Two Roles)
- ✅ CloudWatch log group creation
- ✅ **Base IAM role** creation with OIDC trust policy
- ✅ Base role policy to assume target role (`sts:AssumeRole`)
- ✅ **Target IAM role** creation with base role trust policy
- ✅ Target role policy for CloudWatch Logs access
- ✅ Kubernetes secret creation with both role ARNs

### 2. Two-Role Configuration
- ✅ Service account created **WITHOUT** role ARN annotation
- ✅ Projected service account token volume with `sts.amazonaws.com` audience
- ✅ Environment variable for **base role ARN** (web identity)
- ✅ Exporter `role_arn` parameter for **target role ARN** (role chaining)
- ✅ Web identity token file mount

### 3. Role Chaining Authentication Flow
- ✅ Phase 1: Invalid target role → Auth errors (expected)
- ✅ Base role assumption via `AssumeRoleWithWebIdentity`
- ✅ Phase 2: Patch with correct target role ARN
- ✅ Target role assumption via exporter's `role_arn` parameter
- ✅ Temporary credential generation for target role
- ✅ CloudWatch API access with target role credentials
- ✅ Log delivery to CloudWatch

### 4. End-to-End Verification
- ✅ OpenTelemetry Collector pod startup with invalid target role
- ✅ Auth error validation (phase 1)
- ✅ Collector patch with correct target role
- ✅ Successful role chaining (base → target)
- ✅ Log export to AWS CloudWatch using target role
- ✅ CloudWatch log group and stream validation
- ✅ Log content verification

## 📊 Expected Test Results

```bash
=== AWS CloudWatch Logs Direct Role ARN Test Verification ===
Log Group: tracing-chainsaw-awssts-cloudwatch-direct-ciotelcwl
Region: us-east-2
Base Role ARN (web identity): arn:aws:iam::123456789012:role/cw-base-chainsaw-awssts-cloudwatch-direct-ciotelcwl
Target Role ARN (exporter): arn:aws:iam::123456789012:role/cw-target-chainsaw-awssts-cloudwatch-direct-ciotelcwl

PHASE 1: Authentication Error Validation
✓ Collector pod running with invalid target role
✓ Expected auth errors found (invalid role_arn working as expected)
✓ Base role successfully assumed via web identity
✓ Target role assumption fails (expected with invalid ARN)

PHASE 2: Successful Role Chaining
✓ Collector patched with correct target role ARN
✓ Base role assumed via web identity token
✓ Target role assumed via exporter's role_arn parameter
✓ No AWS-related errors found in collector logs
✓ Log group found in CloudWatch
✓ Log streams found in CloudWatch
✓ Log events found in CloudWatch

=== Test Verification Complete ===
This test successfully demonstrates:
✓ Two-role setup with proper role chaining
✓ Base role authentication via web identity token
✓ Target role assumption via exporter's role_arn parameter
✓ CloudWatch Logs delivered successfully using assumed role

This validates the exporter's role_arn parameter for cross-account/role-chaining scenarios.
```

## 🔄 Direct Role ARN Authentication Flow

```
1. Kubernetes Service Account (NO annotation)
   ↓ (no role ARN annotation needed)
2. Direct Role ARN Configuration
   ↓ (specified in exporter config)
3. Web Identity Token (JWT)
   ↓ (mounted at token file path)
4. AWS SDK STS AssumeRoleWithWebIdentity
   ↓ (using direct role_arn parameter)
5. Temporary AWS Credentials
   ↓ (access key, secret key, session token)
6. CloudWatch Logs API Access
   ↓ (using temporary credentials)
7. Log Delivery to CloudWatch
```

## 🛡️ Security Benefits of Direct Role ARN

1. **Explicit Role Specification** - Role is clearly visible in exporter configuration
2. **No Service Account Dependency** - Reduced coupling with service account annotations
3. **Per-Exporter Configuration** - Different exporters can use different roles
4. **Clear Audit Trail** - Role usage is explicit in configuration
5. **Simplified Service Account Management** - No need to manage role annotations

## 💡 Advantages of Direct Role ARN Configuration

### Configuration Clarity
- Role ARN is explicitly visible in the exporter configuration
- No need to check service account annotations to understand which role is used
- Easier to audit and review IAM role usage

### Flexibility
- Different exporters in the same collector can use different roles
- Role can be easily changed without modifying service account
- Supports dynamic role configuration through environment variables

### Simplified Management
- Service accounts don't need role annotations
- Reduced dependency on Kubernetes-specific AWS integrations
- Works consistently across different Kubernetes distributions

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
./aws-sts-cloudwatch-delete.sh otelcol-cloudwatch chainsaw-awssts-cloudwatch-direct
```

This removes:
- **Base IAM role** and its policy (STS assume role permission)
- **Target IAM role** and its policy (CloudWatch permissions)
- CloudWatch log group
- Kubernetes secret (with both role ARNs)

## 🐛 Troubleshooting

### Common Issues

1. **Pod Not Found Error**
   - Check pod label selector in verification scripts
   - Ensure correct namespace is used (`chainsaw-awssts-cloudwatch-direct`)

2. **Role ARN Not Found**
   - Verify **base_role_arn** is in the secret for web identity
   - Verify **target_role_arn** is in the secret for exporter
   - Check environment variable `AWS_ROLE_ARN` uses `base_role_arn`
   - Check exporter's `role_arn` parameter uses `target_role_arn`

3. **STS Permission Denied (Base Role)**
   - Verify base role trust policy allows OIDC federation
   - Check service account token has `sts.amazonaws.com` audience
   - Verify base role has permission to assume target role

4. **STS Permission Denied (Target Role)**
   - Verify target role trust policy allows base role to assume it
   - Check that base role ARN is correct in target role trust policy
   - Wait 5-10 seconds for IAM propagation after role creation

5. **CloudWatch Access Denied**
   - Verify **target role** IAM policy includes CloudWatch Logs permissions
   - Check log group name and region configuration
   - Ensure target role (not base role) has CloudWatch permissions

### Debug Commands

```bash
# Check collector pod status
oc get pods -n chainsaw-awssts-cloudwatch-direct -l app.kubernetes.io/name=otelcol-cloudwatch-collector

# View collector logs
oc logs -n chainsaw-awssts-cloudwatch-direct deployment/otelcol-cloudwatch-collector

# Check service account (should have NO role annotation)
oc get serviceaccount otelcol-cloudwatch -n chainsaw-awssts-cloudwatch-direct -o yaml

# Verify secret contents (should have base_role_arn and target_role_arn)
oc get secret aws-sts-cloudwatch -n chainsaw-awssts-cloudwatch-direct -o jsonpath='{.data.base_role_arn}' | base64 -d
oc get secret aws-sts-cloudwatch -n chainsaw-awssts-cloudwatch-direct -o jsonpath='{.data.target_role_arn}' | base64 -d

# Check collector configuration for role_arn parameter
oc get opentelemetrycollector otelcol-cloudwatch -n chainsaw-awssts-cloudwatch-direct -o yaml | grep -A 5 "role_arn"

# Verify IAM roles exist
aws iam get-role --role-name cw-base-chainsaw-awssts-cloudwatch-direct-ciotelcwl
aws iam get-role --role-name cw-target-chainsaw-awssts-cloudwatch-direct-ciotelcwl

# Check base role can assume target role
aws iam list-attached-role-policies --role-name cw-base-chainsaw-awssts-cloudwatch-direct-ciotelcwl
```

## 🔗 Related Documentation

- [AWS CloudWatch Logs Exporter Role ARN Configuration](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/awscloudwatchlogsexporter)
- [AWS STS AssumeRoleWithWebIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html)
- [OpenTelemetry AWS CloudWatch Logs Exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/awscloudwatchlogsexporter)

## 🏷️ Test Categories

- **Component**: AWS CloudWatch Logs Exporter
- **Feature**: Exporter's `role_arn` Parameter (Role Chaining)
- **Architecture**: Two-Role Setup (Base + Target)
- **Status**: ✅ Fully Implemented and Working
- **Environment**: OpenShift/Kubernetes + AWS
- **Authentication**: AWS STS Web Identity → Base Role → Target Role (via exporter's `role_arn`)
- **Namespace**: `chainsaw-awssts-cloudwatch-direct`
- **Use Case**: Cross-Account Logging, Role Chaining, Exporter role_arn Testing