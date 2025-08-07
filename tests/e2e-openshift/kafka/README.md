# Kafka Integration Test

This test demonstrates how to use Apache Kafka as a messaging layer for OpenTelemetry telemetry data, enabling scalable and decoupled telemetry processing.

## Test Overview

This test creates:
1. A Kafka cluster using AMQ Streams (Strimzi) operator with KRaft mode
2. KafkaNodePools for controller and broker roles
3. Kafka topics for telemetry data
4. An OpenTelemetry Collector that exports traces to Kafka
5. Another OpenTelemetry Collector that receives traces from Kafka
6. Trace generation to test the end-to-end flow

## Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- AMQ Streams Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

## Configuration Resources

### Kafka Cluster

Deploy a Kafka cluster using AMQ Streams:

**Configuration:** [`00-create-kafka-instance.yaml`](./00-create-kafka-instance.yaml)

Creates a Kafka cluster with:
- KRaft mode enabled (no ZooKeeper required)
- 1 controller node and 1 broker node via KafkaNodePools
- Ephemeral storage for testing
- Plain text (port 9092) and TLS (port 9093) listeners
- Kafka version 3.9.0
- Resource limits: 1 CPU and 4Gi memory per node

### KafkaNodePools

The configuration includes two KafkaNodePools:

**Controller NodePool**: Manages Kafka metadata and cluster coordination
- 1 replica with `controller` role
- Ephemeral storage for testing
- Resource allocation: 1 CPU, 4Gi memory

**Broker NodePool**: Handles message storage and client connections
- 1 replica with `broker` role  
- Ephemeral storage for testing
- Resource allocation: 1 CPU, 4Gi memory

### Kafka Topics

Create topics for telemetry data:

**Configuration:** [`01-create-kafka-topics.yaml`](./01-create-kafka-topics.yaml)

Creates the `otlp-spans` topic with:
- 3 partitions for load distribution
- 1 replica for testing
- 7-day retention policy

### Kafka Receiver Collector

OpenTelemetry Collector that receives data from Kafka and processes it:

**Configuration:** [`02-otel-kakfa-receiver.yaml`](./02-otel-kakfa-receiver.yaml)

Configures a collector that:
- Receives traces from the `otlp-spans` Kafka topic
- Uses Kafka protocol version 3.5.0
- Outputs to debug exporter for verification

### Kafka Exporter Collector

OpenTelemetry Collector that receives OTLP data and exports to Kafka:

**Configuration:** [`03-otel-kakfa-exporter.yaml`](./03-otel-kakfa-exporter.yaml)

Configures a collector that:
- Receives OTLP traces via HTTP and gRPC
- Exports traces to the `otlp-spans` Kafka topic
- Uses internal Kafka broker service for connectivity

### Trace Generator

Generate test traces to validate the Kafka integration:

**Configuration:** [`04-generate-traces.yaml`](./04-generate-traces.yaml)

Creates a job that:
- Generates 50 test traces
- Sends traces to the exporter collector via OTLP HTTP
- Uses `kafka-test-service` as the service name for identification

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
- **Kafka Cluster**: `my-cluster` with KRaft mode enabled
- **KafkaNodePools**: 
  - `controller` - 1 replica with controller role
  - `broker` - 1 replica with broker role
- **Kafka Topic**: `otlp-spans` with 3 partitions
- **Services**: Kafka broker services for internal communication

### OpenTelemetry Collectors
- **Kafka Exporter**: `kafka-exporter-collector` - receives OTLP, exports to Kafka
- **Kafka Receiver**: `kafka-receiver-collector` - receives from Kafka, outputs to debug

### Trace Generation
- **Job**: `generate-traces` - sends test traces to the exporter collector

## Testing the Configuration

The test includes verification logic in the Chainsaw test configuration.

**Verification Script:** [`check_traces.sh`](./check_traces.sh)

The script verifies:
- Trace generation job completes successfully
- Traces are visible in Kafka receiver collector logs
- Expected service name (`kafka-test-service`) appears in traces
- End-to-end trace flow through Kafka messaging

## Verification

The test verifies:
- ✅ Kafka cluster with KRaft mode is ready and accessible
- ✅ KafkaNodePools (controller and broker) are deployed successfully
- ✅ Controller pod (`my-cluster-controller-1`) is running
- ✅ Broker pod (`my-cluster-broker-0`) is running
- ✅ Kafka topics are created successfully
- ✅ Kafka exporter collector is deployed and ready
- ✅ Kafka receiver collector is deployed and ready
- ✅ Trace generation job completes successfully
- ✅ Traces are sent to Kafka via exporter collector
- ✅ Traces are received from Kafka by receiver collector
- ✅ End-to-end trace flow through Kafka is working

## Key Features

- **Kafka Integration**: Uses Apache Kafka as a messaging layer for telemetry
- **KRaft Mode**: Uses Kafka's native metadata management (no ZooKeeper dependency)
- **Node Pool Architecture**: Separates controller and broker roles via KafkaNodePools
- **Decoupled Architecture**: Separates telemetry producers from consumers
- **Scalable Processing**: Kafka enables horizontal scaling of telemetry processing
- **Protocol Version Support**: Uses Kafka protocol version 3.5.0
- **Topic-based Routing**: Routes traces to specific Kafka topics
- **AMQ Streams**: Leverages Red Hat AMQ Streams (Strimzi) for Kafka management

## Configuration Notes

- The Kafka cluster uses KRaft mode with ephemeral storage for testing purposes
- KafkaNodePools separate controller and broker roles for better resource management
- Plain text communication is used between collectors and Kafka (internal cluster)
- Kafka brokers are accessible via internal Kubernetes service DNS
- The `otlp-spans` topic has 3 partitions for load distribution
- Collectors use the Kafka exporter/receiver from OpenTelemetry contrib
- Trace generation creates 50 test traces with 100ms span duration
- Entity operator reconciliation intervals: 90s for topics, 120s for users 