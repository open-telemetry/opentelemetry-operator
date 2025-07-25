# Kafka Integration Test

This test demonstrates how to use Apache Kafka as a messaging layer for OpenTelemetry telemetry data, enabling scalable and decoupled telemetry processing.

## Test Overview

This test creates:
1. A Kafka cluster using AMQ Streams (Strimzi) operator
2. Kafka topics for telemetry data
3. An OpenTelemetry Collector that exports traces to Kafka
4. Another OpenTelemetry Collector that receives traces from Kafka
5. Trace generation to test the end-to-end flow

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- AMQ Streams Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Kafka Cluster

Deploy a Kafka cluster using AMQ Streams:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: chainsaw-kafka

---
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: my-cluster
  namespace: chainsaw-kafka
spec:
  entityOperator:
    topicOperator:
      reconciliationIntervalSeconds: 90
    userOperator:
      reconciliationIntervalSeconds: 120
  kafka:
    config:
      log.message.format.version: 3.9.0
      message.max.bytes: 10485760
      offsets.topic.replication.factor: 1
      ssl.cipher.suites: TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
      ssl.enabled.protocols: TLSv1.2
      ssl.protocol: TLSv1.2
      transaction.state.log.min.isr: 1
      transaction.state.log.replication.factor: 1
    jvmOptions:
      -Xms: 1024m
      -Xmx: 1024m
    listeners:
    - configuration:
        useServiceDnsDomain: true
      name: plain
      port: 9092
      tls: false
      type: internal
    - authentication:
        type: tls
      name: tls
      port: 9093
      tls: true
      type: internal
    replicas: 1
    resources:
      limits:
        cpu: "1"
        memory: 4Gi
      requests:
        cpu: "1"
        memory: 4Gi
    storage:
      type: ephemeral
    version: 3.9.0
  zookeeper:
    replicas: 1
    storage:
      type: ephemeral
```

### Kafka Topics

Create topics for telemetry data:

```yaml
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: otlp-spans
  namespace: chainsaw-kafka
  labels:
    strimzi.io/cluster: my-cluster
spec:
  partitions: 3
  replicas: 1
  config:
    retention.ms: 604800000
    segment.ms: 86400000
```

### Kafka Exporter Collector

OpenTelemetry Collector that receives OTLP data and exports to Kafka:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: kafka-exporter
  namespace: chainsaw-kafka
spec:
  mode: deployment
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
          http:
    processors:
    exporters:
      kafka/traces:
        brokers: ["my-cluster-kafka-brokers.chainsaw-kafka.svc:9092"]
        protocol_version: 3.5.0
        topic: otlp-spans
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [kafka/traces]
```

### Kafka Receiver Collector

OpenTelemetry Collector that receives data from Kafka and processes it:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: kafka-receiver
  namespace: chainsaw-kafka
spec:
  mode: "deployment"
  config: |
    receivers:
      kafka/traces:
        brokers: ["my-cluster-kafka-brokers.chainsaw-kafka.svc:9092"]
        protocol_version: 3.5.0
        topic: otlp-spans
    exporters:
      debug:
        verbosity: detailed
    service:
      pipelines:
        traces:
          receivers: [kafka/traces]
          exporters: [debug]
```

### Trace Generator

Generate test traces to validate the Kafka integration:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: generate-traces
  namespace: chainsaw-kafka
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.124.1
        args:
        - traces
        - --otlp-endpoint=kafka-exporter-collector:4318
        - --traces=50
        - --otlp-http
        - --otlp-insecure=true
        - --service=kafka-test-service
        - --span-duration=100ms
      restartPolicy: Never
```

## Deployment Steps

1. **Create Kafka cluster:**
   ```bash
   oc apply -f 00-create-kafka-instance.yaml
   ```

2. **Create Kafka topics:**
   ```bash
   oc apply -f 01-create-kafka-topics.yaml
   ```

3. **Deploy Kafka receiver collector:**
   ```bash
   oc apply -f 02-otel-kakfa-receiver.yaml
   ```

4. **Deploy Kafka exporter collector:**
   ```bash
   oc apply -f 03-otel-kakfa-exporter.yaml
   ```

5. **Generate test traces:**
   ```bash
   oc apply -f 04-generate-traces.yaml
   ```

## Expected Resources

The test creates and verifies these resources:

### Kafka Infrastructure
- **Namespace**: `chainsaw-kafka`
- **Kafka Cluster**: `my-cluster` with 1 broker and 1 zookeeper
- **Kafka Topic**: `otlp-spans` with 3 partitions
- **Services**: Kafka broker services for internal communication

### OpenTelemetry Collectors
- **Kafka Exporter**: `kafka-exporter-collector` - receives OTLP, exports to Kafka
- **Kafka Receiver**: `kafka-receiver-collector` - receives from Kafka, outputs to debug

### Trace Generation
- **Job**: `generate-traces` - sends test traces to the exporter collector

## Testing the Configuration

The test includes a script (`check_traces.sh`) to verify traces flow through Kafka:

```bash
#!/bin/bash
set -ex

# Function to check if traces are flowing through Kafka
check_kafka_traces() {
    local max_attempts=30
    local attempt=1

    echo "Checking for traces in Kafka receiver logs..."
    
    while [ $attempt -le $max_attempts ]; do
        echo "Attempt $attempt/$max_attempts"
        
        # Check Kafka receiver logs for trace data
        if oc logs -l app.kubernetes.io/name=kafka-receiver-collector --tail=50 | grep -q "traces"; then
            echo "✅ Found traces in Kafka receiver logs!"
            
            # Also check for specific trace data
            if oc logs -l app.kubernetes.io/name=kafka-receiver-collector --tail=50 | grep -q "kafka-test-service"; then
                echo "✅ Found expected service name in traces!"
                return 0
            fi
        fi
        
        sleep 10
        attempt=$((attempt + 1))
    done
    
    echo "❌ Failed to find traces in logs after $max_attempts attempts"
    echo "Recent Kafka receiver logs:"
    oc logs -l app.kubernetes.io/name=kafka-receiver-collector --tail=20
    return 1
}

# Check trace generation job completion
echo "Checking trace generation job..."
oc wait --for=condition=complete job/generate-traces --timeout=300s

# Check if traces are flowing through Kafka
check_kafka_traces
```

## Verification

The test verifies:
- ✅ Kafka cluster is ready and accessible
- ✅ Kafka topics are created successfully
- ✅ Kafka exporter collector is deployed and ready
- ✅ Kafka receiver collector is deployed and ready
- ✅ Trace generation job completes successfully
- ✅ Traces are sent to Kafka via exporter collector
- ✅ Traces are received from Kafka by receiver collector
- ✅ End-to-end trace flow through Kafka is working

## Key Features

- **Kafka Integration**: Uses Apache Kafka as a messaging layer for telemetry
- **Decoupled Architecture**: Separates telemetry producers from consumers
- **Scalable Processing**: Kafka enables horizontal scaling of telemetry processing
- **Protocol Version Support**: Uses Kafka protocol version 3.5.0
- **Topic-based Routing**: Routes traces to specific Kafka topics
- **AMQ Streams**: Leverages Red Hat AMQ Streams (Strimzi) for Kafka management

## Configuration Notes

- The Kafka cluster uses ephemeral storage for testing purposes
- Plain text communication is used between collectors and Kafka (internal cluster)
- Kafka brokers are accessible via internal Kubernetes service DNS
- The `otlp-spans` topic has 3 partitions for load distribution
- Collectors use the Kafka exporter/receiver from OpenTelemetry contrib
- Trace generation creates 50 test traces with 100ms span duration 