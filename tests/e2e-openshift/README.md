# OpenTelemetry Operator OpenShift End-to-End Test Suite

This directory contains a comprehensive set of OpenShift-specific end-to-end tests for the OpenTelemetry Operator. These tests serve as **configuration blueprints** for users to understand and deploy various OpenTelemetry observability patterns on OpenShift.

## 🎯 Purpose

These test scenarios provide OpenTelemetry configuration blueprints that demonstrate:
- Integration with OpenShift-specific features (Routes, Monitoring, Security)
- Real-world observability patterns and configurations
- Step-by-step deployment instructions for various use cases

## 📋 Test Scenarios Overview

| Scenario | Purpose | Key Features |
|----------|---------|-------------|
| [route](./route/) | External Access via OpenShift Routes | Route ingress, OTLP HTTP/gRPC endpoints |
| [scrape-in-cluster-monitoring](./scrape-in-cluster-monitoring/) | Prometheus Metrics Federation | In-cluster monitoring integration, metrics scraping |
| [otlp-metrics-traces](./otlp-metrics-traces/) | OTLP Endpoint with Tempo | Metrics & traces collection, Tempo integration |
| [multi-cluster](./multi-cluster/) | Secure Multi-Cluster Communication | TLS certificates, cross-cluster telemetry |
| [must-gather](./must-gather/) | Diagnostic Information Collection | Must-gather functionality, target allocator |
| [monitoring](./monitoring/) | Platform Monitoring Integration | OpenShift monitoring stack integration |
| [kafka](./kafka/) | Messaging Layer for Telemetry | Kafka-based telemetry distribution |
| [export-to-cluster-logging-lokistack](./export-to-cluster-logging-lokistack/) | Log Export to LokiStack | Log shipping to OpenShift logging |

## 🔗 OpenTelemetry Collector Components Tests

For detailed component-specific configurations and testing patterns, see the **OpenTelemetry Component E2E Test Suite** in the [distributed-tracing-qe](https://github.com/openshift/distributed-tracing-qe.git) repository:

**📡 Receivers:**
- [filelog](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/filelog) - File-based log collection from Kubernetes pods
- [hostmetricsreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/hostmetricsreceiver) - Host system metrics (CPU, memory, disk, network)
- [journaldreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/journaldreceiver) - Systemd journal log collection
- [k8sclusterreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/k8sclusterreceiver) - Kubernetes cluster-wide metrics
- [k8seventsreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/k8seventsreceiver) - Kubernetes events collection
- [k8sobjectsreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/k8sobjectsreceiver) - Kubernetes objects monitoring
- [kubeletstatsreceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/kubeletstatsreceiver) - Kubelet and container metrics
- [otlpjsonfilereceiver](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/otlpjsonfilereceiver) - OTLP JSON file log ingestion

**📤 Exporters:**
- [awscloudwatchlogsexporter](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/awscloudwatchlogsexporter) - AWS CloudWatch Logs integration
- [awsxrayexporter](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/awsxrayexporter) - AWS X-Ray tracing export
- [googlemanagedprometheusexporter](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/googlemanagedprometheusexporter) - Google Cloud Managed Prometheus
- [loadbalancingexporter](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/loadbalancingexporter) - High availability load balancing
- [prometheusremotewriteexporter](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/prometheusremotewriteexporter) - Prometheus remote write integration

**⚙️ Processors:**
- [batchprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/batchprocessor) - Batching for performance optimization
- [filterprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/filterprocessor) - Selective data filtering
- [groupbyattrsprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/groupbyattrsprocessor) - Attribute-based data grouping
- [memorylimiterprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/memorylimiterprocessor) - Memory usage protection
- [resourceprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/resourceprocessor) - Resource attribute manipulation
- [tailsamplingprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/tailsamplingprocessor) - Intelligent trace sampling
- [transformprocessor](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/transformprocessor) - Advanced data transformation

**🔗 Connectors:**
- [forwardconnector](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/forwardconnector) - Data forwarding between pipelines
- [routingconnector](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/routingconnector) - Conditional data routing
- [countconnector](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/countconnector) - Metrics generation from telemetry data

**🔧 Extensions:**
- [oidcauthextension](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/oidcauthextension) - OIDC authentication
- [filestorageextension](https://github.com/openshift/distributed-tracing-qe/tree/main/tests/e2e-otel/filestorageextension) - Persistent file storage

These component test blueprints provide configurations for individual OpenTelemetry components that can be combined with the OpenShift integration patterns documented here.

## 🚀 Quick Start

### Prerequisites

- OpenShift cluster (4.12+)
- OpenTelemetry Operator installed
- `oc` CLI tool configured
- Appropriate cluster permissions

### Running Tests

These tests use [Chainsaw](https://kyverno.github.io/chainsaw/) for end-to-end testing:

```bash
# Run all OpenShift tests
chainsaw test --test-dir tests/e2e-openshift/

# Run specific test scenario
chainsaw test --test-dir tests/e2e-openshift/route/
```

### Using as Configuration Templates

Each test directory contains:
- **Configuration Files**: YAML configuration blueprints
- **README.md**: Step-by-step deployment instructions
- **Scripts**: Verification and setup automation

## 📁 Directory Structure

```
tests/e2e-openshift/
├── README.md                                    # This overview
├── route/                                       # External access patterns
├── scrape-in-cluster-monitoring/               # Prometheus integration
├── otlp-metrics-traces/                        # OTLP with Tempo
├── multi-cluster/                              # Cross-cluster telemetry
├── must-gather/                                # Diagnostic collection
├── monitoring/                                 # Platform monitoring
├── kafka/                                      # Messaging patterns
└── export-to-cluster-logging-lokistack/       # Log export patterns
```

## 🔧 Configuration Patterns

### Common OpenShift Integrations

1. **Security Context Constraints (SCCs)**
   - Automated SCC annotations for namespaces
   - Service account configurations

2. **OpenShift Routes**
   - TLS termination options
   - External endpoint exposure

3. **Monitoring Stack Integration**
   - Prometheus federation
   - Platform monitoring labels

4. **RBAC Configurations**
   - Cluster roles and bindings
   - Service account permissions

## 📖 Documentation

Each test scenario includes:
- **Configuration blueprints** for reference and adaptation
- **Step-by-step instructions** for manual deployment
- **Verification steps** to ensure proper operation
- **Troubleshooting guidance** for common issues

## 🏷️ Labels and Annotations

OpenShift-specific labels and annotations used across scenarios:
- `openshift.io/cluster-monitoring=true` - Enable platform monitoring
- `openshift.io/sa.scc.uid-range` - UID range for security contexts
- `openshift.io/sa.scc.supplemental-groups` - Supplemental groups for SCCs

## 🤝 Contributing

When adding new test scenarios:
1. Include comprehensive README with step-by-step instructions
2. Provide configuration blueprint examples
3. Add verification scripts for testing
4. Document OpenShift-specific considerations

## 📝 Documentation Note

The comprehensive READMEs in this test suite were generated using Claude AI to provide detailed, step-by-step configuration blueprints for OpenTelemetry deployments on OpenShift. These AI-generated guides aim to accelerate user adoption by providing clear, actionable documentation for complex observability scenarios. 

To regenerate / update the docs, the following prompt can be used.

> To enhance our OpenShift End-to-End (E2E) tests, we need to create comprehensive README files within the tests/e2e-openshift directory. These READMEs should provide a maintained set of OpenTelemetry configuration blueprints to assist users in easily deploying and configuring their observability stack, enabling them to quickly access and learn from out-of-the-box observability collection patterns. Each README must include step-by-step instructions referenced directly from the test cases, citing test resources and scripts. Since the test cases are written using the Chainsaw E2E testing tool, the READMEs should be designed from a user perspective for clear and easy follow-through.

## 📚 Additional Resources

- [OpenTelemetry Operator Documentation](https://github.com/open-telemetry/opentelemetry-operator)
- [OpenShift Documentation](https://docs.openshift.com/)
- [Chainsaw Testing Framework](https://kyverno.github.io/chainsaw/) 