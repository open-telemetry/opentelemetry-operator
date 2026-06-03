#!/bin/bash
set -uo pipefail

# --- Set the GCP & OpenShift Vars - MUST MATCH THE CREATION SCRIPT EXACTLY! ---
PROJECT_ID=$(gcloud config get-value project)
OTEL_NAMESPACE="chainsaw-gmpmetrics"
OTEL_SA_NAME="chainsaw-gmpmetrics-sa"
GCP_SA_NAME="otel-gmpmetrics-sa"

# Define the Secret name used in the creation script
SECRET_NAME="gcp-service-account-key"

echo "--- Starting cleanup process for OpenTelemetry Google Cloud Service Account Key resources ---"
echo "  GCP Project ID: $PROJECT_ID"
echo "  OpenShift Namespace: $OTEL_NAMESPACE"
echo "  OpenShift Service Account: $OTEL_SA_NAME"
echo "  Google Cloud Service Account: $GCP_SA_NAME"
echo "  OpenShift Secret Name: $SECRET_NAME"
echo "----------------------------------------------------------------------"

# --- 1. Delete Google Cloud Service Account and Revoke IAM Bindings ---

# Get the full email of the Google Service Account
GCP_SA_EMAIL="${GCP_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

# Important: We no longer need to remove the workloadIdentityUser role as it was not granted.

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

# Note: Deleting the service account will automatically delete all associated keys.
echo "Deleting Google Cloud Service Account: $GCP_SA_EMAIL (This will also delete its keys)..."
gcloud iam service-accounts delete "$GCP_SA_EMAIL" \
  --project "$PROJECT_ID" \
  --quiet || true

echo "--- Google Cloud IAM roles and Service Account successfully removed. ---"

# --- 2. Delete OpenShift Resources ---

echo "Deleting Kubernetes Secret: $SECRET_NAME in namespace $OTEL_NAMESPACE..."
oc delete secret "$SECRET_NAME" -n "$OTEL_NAMESPACE" \
  --ignore-not-found=true || true

echo "Deleting OpenShift Service Account: $OTEL_SA_NAME in namespace $OTEL_NAMESPACE..."
oc delete serviceaccount "$OTEL_SA_NAME" -n "$OTEL_NAMESPACE" \
  --ignore-not-found=true

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