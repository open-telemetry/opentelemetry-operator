#!/bin/bash
set -euo pipefail

# --- User-configurable GCP & OpenShift Vars ---
PROJECT_ID=$(gcloud config get-value project)
OTEL_NAMESPACE="chainsaw-googlecloudexporter"
OTEL_SA_NAME="chainsaw-googlecloudexporter-sa"
GCP_SA_NAME="otel-googlecloudexporter-sa"

# --- Derived GCP & OpenShift Vars ---
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')

# Get provider ID from GCP
PROVIDER_ID=$(gcloud iam workload-identity-pools providers list \
  --project="$PROJECT_ID" \
  --location="global" \
  --workload-identity-pool="$POOL_ID" \
  --filter="displayName:$POOL_ID" \
  --format="value(name)" | awk -F'/' '{print $NF}')

# NOTE: For OSD-GCP clusters, the above automatic derivation may not work correctly.
# In that case, use these static values instead:
#
# OIDC_ISSUER="https://openshift.com"
# PROVIDER_ID="oidc"
#
# For POOL_ID, get this from the OCM command:
# POOL_ID=$(ocm gcp describe wif-config <wif-config-name> | grep -o 'pool_id: [^"]*' | cut -d' ' -f2)
#
# Example static values for OSD-GCP:
# POOL_ID="2ic2l13qv5jc96j5hljg13j8qg0r8bhm"

CRED_CONFIG_FILE="/tmp/credential-configuration.json"
CONFIGMAP_NAME="gcp-wif-credentials"

echo "--- Starting OpenTelemetry Workload Identity Federation setup ---"
echo "  GCP Project ID: $PROJECT_ID"
echo "  OpenShift Namespace: $OTEL_NAMESPACE"
echo "  OpenShift Service Account: $OTEL_SA_NAME"
echo "  Google Cloud Service Account: $GCP_SA_NAME"
echo "  GCP Project Number: $PROJECT_NUMBER"
echo "  OIDC Issuer: $OIDC_ISSUER"
echo "  Workload Identity Pool ID: $POOL_ID"
echo "  Workload Identity Provider ID: $PROVIDER_ID"
echo "------------------------------------------------------------------"

# --- 1. Create OpenShift Namespace and Service Account ---
echo "Creating OpenShift project: $OTEL_NAMESPACE"
oc new-project "$OTEL_NAMESPACE" || true

echo "Creating OpenShift Service Account: $OTEL_SA_NAME in namespace $OTEL_NAMESPACE"
oc create serviceaccount "$OTEL_SA_NAME" -n "$OTEL_NAMESPACE" || true

# --- 2. Create Google Cloud Service Account and Grant IAM Roles ---
echo "Creating Google Cloud Service Account: $GCP_SA_NAME"
GCP_SA_EMAIL=$(gcloud iam service-accounts create "$GCP_SA_NAME" \
  --display-name="OpenTelemetry Google Cloud Exporter Service Account" \
  --project "$PROJECT_ID" \
  --format='value(email)' \
  --quiet || gcloud iam service-accounts describe "$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com" --project "$PROJECT_ID" --format='value(email)')

# Wait for the service account to be ready
echo "Waiting for Google Cloud service account $GCP_SA_EMAIL to be ready..."
MAX_RETRIES=10
RETRY_COUNT=0
while ! gcloud iam service-accounts describe "$GCP_SA_EMAIL" --project "$PROJECT_ID" &> /dev/null; do
  if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
    echo "Error: Google Cloud service account $GCP_SA_EMAIL not found after $MAX_RETRIES retries. Exiting."
    exit 1
  fi
  echo "Google Cloud service account not yet available. Retrying in 5 seconds..."
  sleep 5
  RETRY_COUNT=$((RETRY_COUNT + 1))
done
echo "Google Cloud service account $GCP_SA_EMAIL is ready."

echo "Granting Cloud Trace Agent role to $GCP_SA_EMAIL..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/cloudtrace.agent" \
  --quiet

echo "Granting Monitoring Metric Writer role to $GCP_SA_EMAIL..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.metricWriter" \
  --quiet

echo "Granting Monitoring Editor role to $GCP_SA_EMAIL (required for custom metric descriptors)..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.editor" \
  --quiet

echo "Granting Logging Log Writer role to $GCP_SA_EMAIL..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/logging.logWriter" \
  --quiet

echo "--- GCP IAM permissions configured for $GCP_SA_EMAIL. ---"

# --- 3. Establish Workload Identity Federation: Allow K8s SA to Impersonate GCP SA ---
echo "Granting roles/iam.workloadIdentityUser to Kubernetes Service Account: ${OTEL_NAMESPACE}:${OTEL_SA_NAME}"
gcloud iam service-accounts add-iam-policy-binding "$GCP_SA_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${OTEL_NAMESPACE}:${OTEL_SA_NAME}" \
  --project="$PROJECT_ID" \
  --quiet

echo "--- Workload Identity Federation established. ---"

# --- 4. Create Credential Configuration File and ConfigMap ---
echo "Creating GCP Workload Identity credential configuration file: $CRED_CONFIG_FILE"
gcloud iam workload-identity-pools create-cred-config \
    "projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/providers/$PROVIDER_ID" \
    --service-account="$GCP_SA_EMAIL" \
    --credential-source-file=/var/run/secrets/otel/serviceaccount/token \
    --credential-source-type=text \
    --output-file="$CRED_CONFIG_FILE" \
    --quiet

echo "Importing credential configuration file as ConfigMap: $CONFIGMAP_NAME in namespace $OTEL_NAMESPACE"
oc create configmap "$CONFIGMAP_NAME" \
  --from-file="$CRED_CONFIG_FILE" \
  --namespace="$OTEL_NAMESPACE" \
  --dry-run=client -o yaml | oc apply -f -

echo "Cleaning up local credential file: $CRED_CONFIG_FILE"
rm "$CRED_CONFIG_FILE"

echo "--- Credential ConfigMap $CONFIGMAP_NAME created. ---"
echo "Setup completed successfully!"