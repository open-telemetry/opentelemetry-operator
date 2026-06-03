#!/bin/bash
set -euo pipefail

# --- User-configurable GCP & OpenShift Vars ---
PROJECT_ID=$(gcloud config get-value project)
OTEL_NAMESPACE="chainsaw-googlecloudexporter"
OTEL_SA_NAME="chainsaw-googlecloudexporter-sa"
GCP_SA_NAME="otel-googlecloudexporter-sa"
GCP_SA_EMAIL="$GCP_SA_NAME@$PROJECT_ID.iam.gserviceaccount.com"
CONFIGMAP_NAME="gcp-wif-credentials"

echo "--- Starting OpenTelemetry Workload Identity Federation cleanup ---"
echo "  GCP Project ID: $PROJECT_ID"
echo "  OpenShift Namespace: $OTEL_NAMESPACE"
echo "  OpenShift Service Account: $OTEL_SA_NAME"
echo "  Google Cloud Service Account: $GCP_SA_EMAIL"
echo "----------------------------------------------------------------"

# --- 1. Delete OpenShift Resources ---
echo "Deleting OpenShift ConfigMap: $CONFIGMAP_NAME"
oc delete configmap "$CONFIGMAP_NAME" -n "$OTEL_NAMESPACE" --ignore-not-found=true

echo "Deleting OpenShift Service Account: $OTEL_SA_NAME"
oc delete serviceaccount "$OTEL_SA_NAME" -n "$OTEL_NAMESPACE" --ignore-not-found=true

echo "Deleting OpenShift project: $OTEL_NAMESPACE"
oc delete project "$OTEL_NAMESPACE" --ignore-not-found=true

# --- 2. Remove IAM Policy Bindings ---
echo "Removing Cloud Trace Agent role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/cloudtrace.agent" \
  --quiet || true

echo "Removing Monitoring Metric Writer role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.metricWriter" \
  --quiet || true

echo "Removing Monitoring Editor role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/monitoring.editor" \
  --quiet || true

echo "Removing Logging Log Writer role from $GCP_SA_EMAIL..."
gcloud projects remove-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$GCP_SA_EMAIL" \
  --role="roles/logging.logWriter" \
  --quiet || true

# --- 3. Delete Google Cloud Service Account ---
echo "Deleting Google Cloud Service Account: $GCP_SA_EMAIL"
gcloud iam service-accounts delete "$GCP_SA_EMAIL" \
  --project "$PROJECT_ID" \
  --quiet || true

echo "--- Cleanup completed successfully! ---"