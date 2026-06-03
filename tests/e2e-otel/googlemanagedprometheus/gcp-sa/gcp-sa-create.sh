#!/bin/bash
set -euo pipefail

# --- User-configurable GCP & OpenShift Vars ---
PROJECT_ID=$(gcloud config get-value project)
OTEL_NAMESPACE="chainsaw-gmpmetrics"
OTEL_SA_NAME="chainsaw-gmpmetrics-sa"
GCP_SA_NAME="otel-gmpmetrics-sa"
SA_KEY_FILE="/tmp/credential-configuration.json"
SECRET_NAME="gcp-service-account-key"

echo "--- Starting OpenTelemetry Google Cloud Service Account Key setup ---"
echo "  GCP Project ID: $PROJECT_ID"
echo "  OpenShift Namespace: $OTEL_NAMESPACE"
echo "  OpenShift Service Account: $OTEL_SA_NAME"
echo "  Google Cloud Service Account: $GCP_SA_NAME"
echo "  Service Account Key File (local): $SA_KEY_FILE"
echo "  OpenShift Secret Name: $SECRET_NAME"
echo "-----------------------------------------------------------------------"

# --- 1. Create OpenShift Namespace and Service Account ---
echo "Creating OpenShift project: $OTEL_NAMESPACE"
oc new-project "$OTEL_NAMESPACE" || true

echo "Creating OpenShift Service Account: $OTEL_SA_NAME in namespace $OTEL_NAMESPACE"
oc create serviceaccount "$OTEL_SA_NAME" -n "$OTEL_NAMESPACE" || true

# --- 2. Create Google Cloud Service Account and Grant IAM Roles ---
echo "Creating Google Cloud Service Account: $GCP_SA_NAME"
GCP_SA_EMAIL=$(gcloud iam service-accounts create "$GCP_SA_NAME" \
  --display-name="OpenTelemetry GMP Metrics Service Account" \
  --project "$PROJECT_ID" \
  --format='value(email)' \
  --quiet)

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

echo "Granting Monitoring Metric Writer role to $GCP_SA_EMAIL..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.metricWriter" \
  --quiet

echo "Granting Cloud Telemetry Metrics Writer role to $GCP_SA_EMAIL..."
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/telemetry.metricsWriter" \
  --quiet

echo "--- GCP IAM permissions configured for $GCP_SA_EMAIL. ---"

# --- 3. Generate Service Account Key and Create OpenShift Secret ---
echo "Generating JSON key for service account $GCP_SA_EMAIL and saving to $SA_KEY_FILE"
# Creates a new key for the service account and saves it to the specified file
gcloud iam service-accounts keys create "$SA_KEY_FILE" \
  --iam-account="$GCP_SA_EMAIL" \
  --project="$PROJECT_ID" \
  --quiet

echo "Creating OpenShift Secret: $SECRET_NAME in namespace $OTEL_NAMESPACE from key file: $SA_KEY_FILE"
oc create secret generic "$SECRET_NAME" \
  --from-file="sa-key.json=$SA_KEY_FILE" \
  --namespace="$OTEL_NAMESPACE" \
  --dry-run=client -o yaml | oc apply -f -

echo "Cleaning up local service account key file: $SA_KEY_FILE"
rm "$SA_KEY_FILE"

echo "--- OpenShift Secret $SECRET_NAME created and local key file cleaned up. ---"
