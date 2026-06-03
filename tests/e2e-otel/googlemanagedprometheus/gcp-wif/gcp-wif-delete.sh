#!/bin/bash
set -uo pipefail

# Set the GCP vars - MUST MATCH THE CREATION SCRIPT EXACTLY!
PROJECT_ID=$(gcloud config get-value project)
OTEL_SA_NAME="chainsaw-gmpmetrics-sa"
OTEL_NAMESPACE="chainsaw-gmpmetrics"
GCP_SA_NAME="otel-gmpmetrics-impersonate-sa"
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
OIDC_ISSUER=$(oc get authentication.config cluster -o jsonpath='{.spec.serviceAccountIssuer}')
POOL_ID=$(echo "$OIDC_ISSUER" | awk -F'/' '{print $NF}' | sed 's/-oidc$//')

# Define the ConfigMap name used in the creation script
CONFIGMAP_NAME="gcp-wif-credentials"

echo "--- Starting cleanup process for OpenTelemetry Workload Identity resources ---"
echo "  GCP Project ID: $PROJECT_ID"
echo "  OpenShift Namespace: $OTEL_NAMESPACE"
echo "  OpenShift Service Account: $OTEL_SA_NAME"
echo "  Google Cloud Service Account: $GCP_SA_NAME"
echo "----------------------------------------------------------------------"

# --- 1. Revoke Google Cloud IAM Bindings and Delete Google Service Account ---

# Get the full email of the Google Service Account
GCP_SA_EMAIL="${GCP_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "Revoking roles/iam.workloadIdentityUser from Kubernetes Service Account's ability to impersonate $GCP_SA_EMAIL..."
gcloud iam service-accounts remove-iam-policy-binding "$GCP_SA_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principal://iam.googleapis.com/projects/$PROJECT_NUMBER/locations/global/workloadIdentityPools/$POOL_ID/subject/system:serviceaccount:${OTEL_NAMESPACE}:${OTEL_SA_NAME}" \
  --project="$PROJECT_ID" \
  --quiet || true

echo "Revoking Monitoring Metric Writer role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.metricWriter" \
  --quiet || true

echo "Revoking Cloud Telemetry Metrics Writer role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/telemetry.metricsWriter" \
  --quiet || true

echo "Deleting Google Cloud Service Account: $GCP_SA_EMAIL..."
gcloud iam service-accounts delete "$GCP_SA_EMAIL" \
  --project "$PROJECT_ID" \
  --quiet || true

echo "--- Google Cloud IAM roles and Service Account successfully removed. ---"

# --- 2. Delete OpenShift Resources ---

echo "Deleting Kubernetes ConfigMap: $CONFIGMAP_NAME in namespace $OTEL_NAMESPACE..."
oc delete configmap "$CONFIGMAP_NAME" -n "$OTEL_NAMESPACE" \
  --ignore-not-found=true || true # Ignore if ConfigMap doesn't exist

echo "Deleting OpenShift Service Account: $OTEL_SA_NAME in namespace $OTEL_NAMESPACE..."
oc delete serviceaccount "$OTEL_SA_NAME" -n "$OTEL_NAMESPACE" \
  --ignore-not-found=true || true # Ignore if SA doesn't exist

echo "Deleting OpenShift project/namespace: $OTEL_NAMESPACE..."
oc delete project "$OTEL_NAMESPACE" \
  --ignore-not-found=true \
  --wait=false || true

echo "Deleting OpenShift project/namespace: chainsaw-kubeletstatsreceiver..."
oc delete project chainsaw-kubeletstatsreceiver \
  --ignore-not-found=true \
  --wait=false || true

echo "--- OpenShift resources deletion initiated. ---"
echo "Project deletion may take some time to complete in the background."

echo "----------------------------------------------------------------------"
echo "Cleanup script finished. All specified resources are being removed or have been removed."