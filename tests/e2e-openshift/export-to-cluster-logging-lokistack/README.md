# Export to Cluster Logging LokiStack Test

This test demonstrates how to export OpenTelemetry logs to OpenShift's cluster logging infrastructure using LokiStack for centralized log management and analysis.

## Test Overview

This test creates:
1. A MinIO instance for LokiStack object storage
2. A LokiStack instance for log storage and querying
3. An OpenTelemetry Collector that processes and exports logs to LokiStack
4. Log generation to test the end-to-end flow
5. Integration with OpenShift logging UI plugin

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Red Hat OpenShift Logging Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### MinIO Object Storage

Deploy MinIO for LokiStack storage backend:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  labels:
    app.kubernetes.io/name: minio
  name: minio
  namespace: openshift-logging
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: openshift-logging
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: minio
  strategy:
    type: Recreate
  template:
    metadata:
      labels:
        app.kubernetes.io/name: minio
    spec:
      containers:
        - command:
            - /bin/sh
            - -c
            - |
              mkdir -p /storage/tempo && \
              minio server /storage
          env:
            - name: MINIO_ACCESS_KEY
              value: tempo  # Demo credentials for testing only
            - name: MINIO_SECRET_KEY
              value: supersecret  # Demo credentials for testing only
          image: minio/minio
          name: minio
          ports:
            - containerPort: 9000
          volumeMounts:
            - mountPath: /storage
              name: storage
      volumes:
        - name: storage
          persistentVolumeClaim:
            claimName: minio

---
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: openshift-logging
spec:
  ports:
    - port: 9000
      protocol: TCP
      targetPort: 9000
  selector:
    app.kubernetes.io/name: minio
  type: ClusterIP

---
apiVersion: v1
kind: Secret
metadata:
  name: logging-loki-s3
  namespace: openshift-logging
stringData:
  endpoint: http://minio:9000
  bucketnames: tempo
  access_key_id: tempo  # Demo credentials for testing only
  access_key_secret: supersecret  # Demo credentials for testing only
type: Opaque
```

### LokiStack Instance

Deploy LokiStack for log storage:

```yaml
apiVersion: loki.grafana.com/v1
kind: LokiStack
metadata:
  name: logging-loki 
  namespace: openshift-logging 
spec:
  size: 1x.demo
  storage:
    schemas:
    - version: v13
      effectiveDate: "2023-10-15"
    secret:
      name: logging-loki-s3 
      type: s3
  storageClassName: ($STORAGE_CLASS_NAME)
  tenants:
    mode: openshift-logging
```

### OpenTelemetry Collector Configuration

Deploy collector with LokiStack integration:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otel-collector-deployment
  namespace: openshift-logging

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-collector-logs-writer
rules:
 - apiGroups: ["loki.grafana.com"]
   resourceNames: ["logs"]
   resources: ["application"]
   verbs: ["create", "get"]
 - apiGroups: [""]
   resources: ["pods", "namespaces", "nodes"]
   verbs: ["get", "watch", "list"]
 - apiGroups: ["apps"]
   resources: ["replicasets"]
   verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otel-collector-logs-writer
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otel-collector-logs-writer
subjects:
  - kind: ServiceAccount
    name: otel-collector-deployment
    namespace: openshift-logging

---
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel
  namespace: openshift-logging
spec:
  image: ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector-contrib:0.116.1
  serviceAccount: otel-collector-deployment
  config:
    extensions:
      bearertokenauth:
        filename: "/var/run/secrets/kubernetes.io/serviceaccount/token"
    receivers:
      otlp:
        protocols:
          grpc: {}
          http: {}
    processors:
      k8sattributes: {}
      resource:
        attributes:
          - key:  kubernetes.namespace_name
            from_attribute: k8s.namespace.name
            action: upsert
          - key:  kubernetes.pod_name
            from_attribute: k8s.pod.name
            action: upsert
          - key: kubernetes.container_name
            from_attribute: k8s.container.name
            action: upsert
          - key: log_type
            value: application
            action: upsert
      transform:
        log_statements:
          - context: log
            statements:
              - set(attributes["level"], ConvertCase(severity_text, "lower"))
    exporters:
      otlphttp:
        endpoint: https://logging-loki-gateway-http.openshift-logging.svc.cluster.local:8080/api/logs/v1/application/otlp
        encoding: json
        tls:
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
        auth:
          authenticator: bearertokenauth
      debug:
        verbosity: detailed
    service:
      extensions: [bearertokenauth]
      pipelines:
        logs:
          receivers: [otlp]
          processors: [k8sattributes, transform, resource]
          exporters: [otlphttp]
        logs/test:
          receivers: [otlp]
          processors: []
          exporters: [debug]
```

### Log Generation

Generate test logs to validate the pipeline:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-logs
  namespace: openshift-logging
spec:
  template:
    spec:
      containers:
      - name: log-generator
        image: curlimages/curl:latest
        command:
        - /bin/sh
        - -c
        - |
          set -ex
          
          # Generate structured logs in OTLP format
          for i in $(seq 1 20); do
            LOG_TIME=$(date +%s%N)
            
            # Create OTLP log entry
            curl -X POST http://otel-collector:4318/v1/logs \
              -H "Content-Type: application/json" \
              -d "{
                \"resourceLogs\": [{
                  \"resource\": {
                    \"attributes\": [{
                      \"key\": \"service.name\",
                      \"value\": {\"stringValue\": \"log-generator\"}
                    }, {
                      \"key\": \"service.version\",
                      \"value\": {\"stringValue\": \"1.0.0\"}
                    }]
                  },
                  \"scopeLogs\": [{
                    \"logRecords\": [{
                      \"timeUnixNano\": \"${LOG_TIME}\",
                      \"severityText\": \"INFO\",
                      \"body\": {
                        \"stringValue\": \"Test log message ${i} - This is a sample log entry for LokiStack integration testing\"
                      },
                      \"attributes\": [{
                        \"key\": \"log.sequence\",
                        \"value\": {\"intValue\": ${i}}
                      }, {
                        \"key\": \"log.source\",
                        \"value\": {\"stringValue\": \"test-generator\"}
                      }, {
                        \"key\": \"environment\",
                        \"value\": {\"stringValue\": \"development\"}
                      }]
                    }]
                  }]
                }]
              }"
            
            echo "Generated log entry ${i}"
            sleep 1
          done
          
          echo "Log generation completed - sent 20 log entries"
      restartPolicy: Never
```

### Logging UI Plugin

Enable the logging UI plugin for log visualization:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: logging-view-plugin
  namespace: openshift-logging
data:
  plugin-manifest.yaml: |
    apiVersion: console.openshift.io/v1alpha1
    kind: ConsolePlugin
    metadata:
      name: logging-view-plugin
    spec:
      displayName: "OpenShift Logging"
      backend:
        type: Service
        service:
          name: logging-view-plugin
          namespace: openshift-logging
          port: 9001
          basePath: /
```

## Deployment Steps

1. **Install MinIO for object storage:**
   ```bash
   oc apply -f install-minio.yaml
   ```

2. **Deploy LokiStack instance:**
   ```bash
   oc apply -f install-loki.yaml
   ```

3. **Deploy OpenTelemetry Collector with RBAC:**
   ```bash
   oc apply -f otel-collector.yaml
   ```

4. **Enable logging UI plugin:**
   ```bash
   oc apply -f logging-uiplugin.yaml
   ```

5. **Generate test logs:**
   ```bash
   oc apply -f generate-logs.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Storage Infrastructure
- **MinIO**: Object storage backend with `tempo` bucket
- **PVC**: 2Gi persistent volume for MinIO storage
- **Secret**: `logging-loki-s3` with MinIO access credentials

### Logging Stack
- **LokiStack**: `logging-loki` instance in demo mode
- **Gateway**: HTTP gateway for log ingestion
- **Storage Schema**: v13 schema with 2023-10-15 effective date

### OpenTelemetry Integration
- **Service Account**: `otel-collector-deployment` with logging permissions
- **Collector**: `otel-collector` with LokiStack exporter
- **RBAC**: Cluster role for writing to LokiStack

### Log Generation
- **Job**: `generate-logs` creating test log entries
- **UI Plugin**: Logging view plugin for OpenShift console

## Testing the Configuration

The test includes a script (`check_logs.sh`) to verify log flow:

```bash
#!/bin/bash
set -ex

# Function to check if logs are reaching LokiStack
check_logs_in_loki() {
    local max_attempts=30
    local attempt=1

    echo "Checking for logs in LokiStack..."
    
    while [ $attempt -le $max_attempts ]; do
        echo "Attempt $attempt/$max_attempts"
        
        # Check collector logs for successful exports
        if oc logs -l app.kubernetes.io/name=otel-collector --tail=50 | grep -q "successfully sent"; then
            echo "✅ Found successful log exports in collector logs!"
            
            # Check for LokiStack gateway logs
            if oc logs -l app.kubernetes.io/component=gateway --tail=50 | grep -q "POST.*logs"; then
                echo "✅ Found log ingestion in LokiStack gateway logs!"
                return 0
            fi
        fi
        
        sleep 10
        attempt=$((attempt + 1))
    done
    
    echo "❌ Failed to verify logs in LokiStack after $max_attempts attempts"
    echo "Recent collector logs:"
    oc logs -l app.kubernetes.io/name=otel-collector --tail=20
    return 1
}

# Wait for log generation to complete
echo "Waiting for log generation job to complete..."
oc wait --for=condition=complete job/generate-logs --timeout=300s

# Check if logs are flowing to LokiStack
check_logs_in_loki

echo "✅ Log export to LokiStack verification completed successfully!"
```

## Additional Verification Commands

Verify the logging infrastructure:

```bash
# Check LokiStack status
oc get lokistack logging-loki -o yaml

# Check MinIO deployment
oc get deployment minio -o yaml

# View LokiStack gateway service
oc get svc logging-loki-gateway-http

# Check collector service account permissions
oc auth can-i create application --as=system:serviceaccount:openshift-logging:otel-collector-deployment

# Port forward to MinIO for bucket verification
oc port-forward svc/minio 9000:9000 &
curl -u tempo:supersecret http://localhost:9000/minio/health/live  # Using demo test credentials

# Check collector metrics
oc port-forward svc/otel-collector 8888:8888 &
curl http://localhost:8888/metrics | grep otelcol_exporter
```

## Verification

The test verifies:
- ✅ MinIO is deployed and accessible as object storage
- ✅ LokiStack instance is ready and configured
- ✅ OpenTelemetry Collector has proper RBAC permissions
- ✅ Collector is configured with LokiStack OTLP exporter
- ✅ Bearer token authentication is working
- ✅ Log generation job completes successfully
- ✅ Logs are processed through k8sattributes and transform processors
- ✅ Logs are successfully exported to LokiStack
- ✅ Logging UI plugin is enabled for log visualization

## Key Features

- **LokiStack Integration**: Native integration with OpenShift cluster logging
- **Object Storage**: MinIO backend for log persistence
- **Authentication**: Bearer token authentication with service accounts
- **Log Processing**: Kubernetes attributes and log transformation
- **OTLP Protocol**: Uses OTLP HTTP for log export to LokiStack
- **Multi-Pipeline**: Separate pipelines for LokiStack export and debug output
- **UI Integration**: Logging console plugin for log visualization

## Configuration Notes

- LokiStack runs in `1x.demo` size for testing environments
- MinIO uses ephemeral storage with demo credentials (tempo/supersecret) - **FOR TESTING ONLY**
- Collector uses bearer token authentication for LokiStack access
- Service CA certificate is used for TLS communication with LokiStack
- Log processors add Kubernetes metadata and normalize severity levels
- The application log type is set for proper tenant routing in LokiStack 