# Scrape In-Cluster Monitoring Test

This test demonstrates how to configure an OpenTelemetry Collector to scrape metrics from OpenShift's built-in Prometheus monitoring system using federation.

## Test Overview

This test creates an OpenTelemetry Collector that federates metrics from OpenShift's monitoring stack. It uses the Prometheus receiver to scrape specific metrics (`kube_namespace_labels`) from the cluster monitoring Prometheus instance and exports them via debug exporter.

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Cluster Role Binding

The test creates a cluster role binding to allow the collector to access monitoring data:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: chainsaw-scrape-in-cluster-monitoring-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-monitoring-view 
subjects:
  - kind: ServiceAccount
    name: otel-collector
    namespace: chainsaw-scrape-in-cluster-monitoring
```

### CA Bundle ConfigMap

A ConfigMap is created to inject the service CA bundle for TLS communication:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: cabundle
  namespace: chainsaw-scrape-in-cluster-monitoring
  annotations:
    service.beta.openshift.io/inject-cabundle: "true"
```

### OpenTelemetry Collector Configuration

The test deploys this OpenTelemetry Collector configuration:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel
  namespace: chainsaw-scrape-in-cluster-monitoring
spec:
  volumeMounts:
    - name: cabundle-volume
      mountPath: /etc/pki/ca-trust/source/service-ca
      readOnly: true
  volumes:
    - name: cabundle-volume
      configMap:
        name: cabundle
  mode: deployment
  config: |
    receivers:
      prometheus: 
        config:
          scrape_configs:
            - job_name: 'federate'
              scrape_interval: 15s
              scheme: https
              tls_config:
                ca_file: /etc/pki/ca-trust/source/service-ca/service-ca.crt
              bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
              # honor_labels needs to be set to false due to bug https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/32555
              honor_labels: false
              params:
                'match[]':
                  - '{__name__="kube_namespace_labels"}'
              metrics_path: '/federate'
              static_configs:
                - targets:
                  - "prometheus-k8s.openshift-monitoring.svc.cluster.local:9091"

    exporters:
      debug: 
        verbosity: detailed

    service:
      pipelines:
        metrics:
          receivers: [prometheus]
          exporters: [debug]
```

## Deployment Steps

1. **Apply the RBAC configuration:**
   ```bash
   oc apply -f create-clusterrolebinding.yaml
   ```

2. **Apply the OpenTelemetry Collector configuration:**
   ```bash
   oc apply -f create-otel-instance.yaml
   ```

3. **Verify the deployment is ready:**
   ```bash
   oc wait --for=condition=ready pod -l app.kubernetes.io/name=otel-collector --timeout=300s
   ```

## Expected Resources

The test creates and verifies these resources:

### Deployment
- **Name**: `otel-collector`
- **Status**: Ready replica available

### Service
- **Name**: `otel-collector`
- **Ports**: Exposes collector service endpoints

## Testing the Configuration

The test includes a script (`check_logs.sh`) that verifies the collector is receiving metrics:

```bash
#!/bin/bash
set -ex

# Function to check logs for specific metric
check_metric_in_logs() {
    local metric_name="$1"
    local max_attempts=30
    local attempt=1

    echo "Checking for metric: $metric_name"
    
    while [ $attempt -le $max_attempts ]; do
        echo "Attempt $attempt/$max_attempts"
        
        # Get logs from the collector
        if oc logs -l app.kubernetes.io/name=otel-collector --tail=50 | grep -q "$metric_name"; then
            echo "Found metric $metric_name in logs!"
            return 0
        fi
        
        sleep 10
        attempt=$((attempt + 1))
    done
    
    echo "Failed to find metric $metric_name in logs after $max_attempts attempts"
    echo "Recent logs:"
    oc logs -l app.kubernetes.io/name=otel-collector --tail=20
    return 1
}

# Check for the federated metric
check_metric_in_logs "kube_namespace_labels"
```

## Verification

The test verifies:
- ✅ Cluster role binding is created
- ✅ ConfigMap with CA bundle is created and populated
- ✅ OpenTelemetry Collector deployment is ready
- ✅ Collector service is created
- ✅ Collector successfully scrapes `kube_namespace_labels` metric from Prometheus

## Key Features

- **Prometheus Federation**: Uses Prometheus receiver to federate metrics from cluster monitoring
- **TLS Authentication**: Properly configured TLS with service CA certificate
- **Bearer Token Auth**: Uses service account token for authentication
- **Selective Scraping**: Only scrapes specific metrics (`kube_namespace_labels`)
- **RBAC Integration**: Uses `cluster-monitoring-view` role for proper permissions

## Configuration Notes

- The collector targets the federate endpoint of OpenShift's Prometheus
- `honor_labels` is set to `false` due to a known bug in the Prometheus receiver
- TLS configuration uses the injected service CA bundle
- Bearer token authentication uses the mounted service account token
- Only the `kube_namespace_labels` metric is scraped for testing purposes 