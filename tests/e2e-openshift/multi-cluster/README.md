# Multi-Cluster TLS Communication Test

This test demonstrates secure multi-cluster telemetry communication between OpenTelemetry Collectors using mutual TLS (mTLS) authentication via OpenShift Routes.

## Test Overview

This test sets up:
1. **Receiver cluster**: Deploys an OpenTelemetry Collector that receives traces via OTLP with TLS
2. **Sender cluster**: Deploys an OpenTelemetry Collector that sends traces to the receiver cluster
3. **TLS certificates**: Generates and configures mTLS certificates for secure communication
4. **Tempo storage**: Stores received traces in a Tempo instance
5. **Route exposure**: Exposes the receiver via OpenShift Routes with passthrough TLS

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- Tempo Operator installed
- `oc` CLI tool configured
- `openssl` and `jq` utilities
- Appropriate cluster permissions

## Configuration Resources

### Namespace Setup

The test creates two namespaces:
- `chainsaw-multi-cluster-receive` - For the receiver cluster components
- `chainsaw-multi-cluster-send` - For the sender cluster components

### TLS Certificate Generation

The test generates TLS certificates using this script (`generate_certs.sh`):

```bash
#!/bin/bash

# Create a directory to store certificates
CERT_DIR="/tmp/chainsaw-certs"
rm -rf "$CERT_DIR"
mkdir -p "$CERT_DIR"

# Get hostname domain from OpenShift
hostname_domain="*.apps.$(oc get dns cluster -o jsonpath='{.spec.baseDomain}')"

# Set certificate information
CERT_SUBJECT="/C=US/ST=California/L=San Francisco/O=My Organization/CN=opentelemetry"

# Create OpenSSL configuration file for SANs
openssl_config="$CERT_DIR/openssl.cnf"
cat <<EOF > "$openssl_config"
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = v3_req

[ req_distinguished_name ]
countryName                = Country Name (2 letter code)
countryName_default        = US
stateOrProvinceName        = State or Province Name (full name)
stateOrProvinceName_default= California
localityName               = Locality Name (eg, city)
localityName_default       = San Francisco
organizationName           = Organization Name (eg, company)
organizationName_default   = My Organization
commonName                 = Common Name (eg, your name or your server's hostname)
commonName_max             = 64

[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = opentelemetry
DNS.2 = $hostname_domain
EOF

# Generate certificates
openssl genpkey -algorithm RSA -out "$CERT_DIR/server.key"
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "$CERT_SUBJECT" -config "$openssl_config"
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" -extensions v3_req -extfile "$openssl_config"
openssl req -new -x509 -days 365 -key "$CERT_DIR/server.key" -out "$CERT_DIR/ca.crt" -subj "$CERT_SUBJECT"

# Create ConfigMaps in both namespaces
kubectl create configmap chainsaw-certs -n chainsaw-multi-cluster-send \
  --from-file=server.crt="$CERT_DIR/server.crt" \
  --from-file=server.key="$CERT_DIR/server.key" \
  --from-file=ca.crt="$CERT_DIR/ca.crt"

kubectl create configmap chainsaw-certs -n chainsaw-multi-cluster-receive \
  --from-file=server.crt="$CERT_DIR/server.crt" \
  --from-file=server.key="$CERT_DIR/server.key" \
  --from-file=ca.crt="$CERT_DIR/ca.crt"
```

### Tempo Instance

The receiver cluster includes a Tempo instance for trace storage:

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: multicluster
  namespace: chainsaw-multi-cluster-receive
spec: {}
```

### OTLP Receiver Collector

The receiver collector accepts OTLP traffic over TLS and forwards to Tempo:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otlp-receiver
  namespace: chainsaw-multi-cluster-receive
spec:
  config: |
    receivers:
      otlp:
        protocols:
          http:
            tls:
              cert_file: /certs/server.crt
              key_file: /certs/server.key
              client_ca_file: /certs/ca.crt
          grpc:
            tls:
              cert_file: /certs/server.crt
              key_file: /certs/server.key
              client_ca_file: /certs/ca.crt
    exporters:
      otlp:
        endpoint: "tempo-multicluster.chainsaw-multi-cluster-receive.svc:4317"
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [otlp]
  ingress:
    route:
      termination: passthrough
    type: route
  mode: deployment
  volumeMounts:
  - mountPath: /certs
    name: chainsaw-certs
  volumes:
  - configMap:
      name: chainsaw-certs
    name: chainsaw-certs
```

### RBAC Configuration

The sender requires RBAC permissions to collect cluster information:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: chainsaw-multi-cluster
  namespace: chainsaw-multi-cluster-send
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: chainsaw-multi-cluster
rules:
- apiGroups:
  - config.openshift.io
  resources:
  - infrastructures
  - infrastructures/status
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - apps
  resources:
  - replicasets
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
  - watch
  - list
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
  - watch
  - list
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: chainsaw-multi-cluster
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: chainsaw-multi-cluster
subjects:
- kind: ServiceAccount
  name: chainsaw-multi-cluster
  namespace: chainsaw-multi-cluster-send
```

### OTLP Sender Collector

The sender collector is created dynamically using a script that discovers the receiver routes:

```bash
#!/bin/bash

# Get the HTTP and GRPC routes from receiver collector
otlp_route_http=$(oc -n chainsaw-multi-cluster-receive get route otlp-http-otlp-receiver-route -o json | jq '.spec.host' -r)
otlp_route_grpc=$(oc -n chainsaw-multi-cluster-receive get route otlp-grpc-otlp-receiver-route -o json | jq '.spec.host' -r)

# Create sender collector configuration
collector_content=$(cat <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel-sender
  namespace: chainsaw-multi-cluster-send
spec:
  mode: deployment
  serviceAccount: chainsaw-multi-cluster
  volumes:
    - name: chainsaw-certs
      configMap: 
        name: chainsaw-certs
  volumeMounts:
    - name: chainsaw-certs
      mountPath: /certs
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:
    processors:
      batch:
      memory_limiter:
        check_interval: 1s
        limit_percentage: 50
        spike_limit_percentage: 30
    exporters:
      otlphttp:
        endpoint: "https://${otlp_route_http}:443"
        tls:
          insecure: false
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          ca_file: /certs/ca.crt
      otlp:
        endpoint: "${otlp_route_grpc}:443"
        tls:
          insecure: false
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          ca_file: /certs/ca.crt
    service:
      pipelines:
        traces:    
          receivers: [otlp]
          processors: [memory_limiter, batch]
          exporters: [otlphttp, otlp]
EOF
)

echo "$collector_content" | oc -n chainsaw-multi-cluster-send create -f -
```

### Trace Generator Jobs

The test creates jobs to generate traces for both HTTP and gRPC protocols:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces-http
  namespace: chainsaw-multi-cluster-send
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.124.1
        args:
        - traces
        - --otlp-endpoint=otel-sender-collector:4318
        - --traces=100
        - --otlp-http
        - --otlp-insecure=true
        - --service=telemetrygen-http
        - --otlp-attributes=protocol="http"
      restartPolicy: Never
---
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces-grpc
  namespace: chainsaw-multi-cluster-send
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.124.1
        args:
        - traces
        - --otlp-endpoint=otel-sender-collector:4317
        - --traces=100
        - --otlp-insecure=true
        - --service=telemetrygen-grpc
        - --otlp-attributes=protocol="grpc"
      restartPolicy: Never
```

## Deployment Steps

1. **Create namespaces:**
   ```bash
   oc apply -f 00-create-namespaces.yaml
   ```

2. **Install Tempo in receiver namespace:**
   ```bash
   oc apply -f 01-create-tempo.yaml
   ```

3. **Deploy OTLP receiver collector:**
   ```bash
   oc apply -f 02-otlp-receiver.yaml
   ```

4. **Create RBAC and sender collector:**
   ```bash
   oc apply -f 03-otlp-sender.yaml
   ```

5. **Generate and configure TLS certificates:**
   ```bash
   ./generate_certs.sh
   ```

6. **Create dynamic sender collector:**
   ```bash
   ./create_otlp_sender.sh
   ```

7. **Generate test traces:**
   ```bash
   oc apply -f 04-generate-traces.yaml
   ```

8. **Verify traces are received:**
   ```bash
   oc apply -f verify-traces.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Receiver Namespace
- **Tempo Instance**: `tempo-multicluster`
- **Collector**: `otlp-receiver-collector`
- **Routes**: HTTP and gRPC routes with passthrough TLS
- **ConfigMap**: `chainsaw-certs` with TLS certificates

### Sender Namespace
- **Service Account**: `chainsaw-multi-cluster`
- **Collector**: `otel-sender-collector`
- **Jobs**: Trace generator jobs for HTTP and gRPC
- **ConfigMap**: `chainsaw-certs` with TLS certificates

## Verification

The test verifies:
- ✅ Both namespaces are created
- ✅ Tempo instance is ready in receiver namespace
- ✅ OTLP receiver collector is deployed and ready
- ✅ Routes are created with passthrough TLS termination
- ✅ RBAC permissions are configured for sender
- ✅ OTLP sender collector is deployed and ready
- ✅ TLS certificates are generated and mounted
- ✅ Trace generation jobs complete successfully
- ✅ Traces are received and stored in Tempo

## Key Features

- **Mutual TLS Authentication**: Full mTLS configuration between sender and receiver
- **Passthrough TLS**: Routes use passthrough termination for end-to-end encryption
- **Dual Protocol Support**: Both HTTP and gRPC OTLP protocols over TLS
- **Certificate Management**: Automated certificate generation and distribution
- **Cross-Cluster Communication**: Secure telemetry communication between clusters
- **Route Discovery**: Dynamic discovery of receiver routes for sender configuration

## Configuration Notes

- Routes use `passthrough` termination to maintain end-to-end TLS
- Certificates include Subject Alternative Names (SANs) for OpenShift route domains
- The sender collector dynamically discovers receiver route endpoints
- Both HTTP (port 443) and gRPC (port 443) endpoints are exposed via routes
- TLS certificates are shared between sender and receiver via ConfigMaps 