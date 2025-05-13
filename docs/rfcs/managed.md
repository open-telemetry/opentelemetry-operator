# Managed CRD

**Status:** *Draft*

**Author:** Antoine Toulme (@atoulme)

**Date:** 2025-03-20

## Objective

Condense all best practices and deployment patterns of OpenTelemetry components into a single custom resource
that covers the use cases fulfilled by the OpenTelemetry helm charts.

## Summary
This request for comment aims to establish a new approach to management of resources in Kubernetes clusters,
with the explicit goal of taking over the whole cluster, and making decisions
on how to best instrument all resources in the cluster on behalf of the administrator based on best practices,
OpenTelemetry standards and semantic conventions.

The operator project offers custom resources (CR) to allow Kubernetes users/administrators to define OpenTelemetry components
in their Kubernetes environments. Users however do not want to manage components themselves, especially if they need
to think about component versioning and upgrade patterns, and would rather have an out-of-the-box experience that 
requires little insight and gets them to a working state effortlessly.

## Goals and non-goals

### Give a comprehensive, out-of the box experience of OpenTelemetry for a first-time user
The goal is to delight first-time users by giving them a complete view of what is possible with OpenTelemetry.

### Do not offer a way to turn on and off specific features
We mean to offer a specific functional coverage of a specific use case that offers best practices of OpenTelemetry.

We don't allow toggling features specifically because we want to avoid increasing the need for functional test coverage.

If Users want to increase their level of control of the feature set, they can remove the managed custom resource and instead
follow the traditional route of setting up collectors, target allocators, instrumentation custom resources as is possible today.

### OTLP, single endpoint support

The solution exports data to a single endpoint using the OTLP protocol. Users may choose to deploy a gateway
to transform, filter, redact data before sending it to storage.

### Context-aware

Best practices differ between platforms, cloud providers, available endpoints, node operating system and so on.

The solution will enable features based on those parameters. For example, it will deploy Openshift-specific features when deployed on the Openshift Cloud Platform.

## Use cases for proposal

### Initial installation

The user wants to install the operator and immediately get value from the installation.

Upon installation, they install the custom resource with information on an OTLP endpoint where they can direct all data.

### Upgrade

The user wants to upgrade their operator. All assets are upgraded out of the box in concert as part of the upgrade,
and are guaranteed to work through functional test coverage.

### Deletion and reinstallation

The user can cleanly delete the custom resource, which triggers the deletion of all OpenTelemetry assets under
management by the operator. No stragglers are left.

### Infrastructure monitoring

The user upon installation of the custom resource can see Kubernetes cluster metrics, and kubeletstats metrics.

The user can follow events and receive Kubernetes entity data.

The user upon installation receives metrics from hostmetrics receiver with its default configuration. Note the metrics map to the host, meaning the collector will have access to host volumes to scrape their utilization.

### Workload monitoring

All standard CRDs for instrumentations are enabled by default for all namespaces.

### Node logs

The user upon installation will receive logs from the node operating system, as well as pod logs from Kubernetes, scraped with the filelog receiver from the file system.

## Configuration

The user can enable exporting signals individually: metrics, logs, traces and profiles.

## Struct Design

```go
import "go.opentelemetry.io/collector/exporter/otlpexporter"

type ClusterObservability struct {
	# List of signals supported: `logs`, `metrics`, `traces`, `profiles`
	Signals        []string `yaml:"signals"`
	# OTLP exporter configuration
	ExporterConfig otlpexporter.Config `yaml:"exporter"`
}
```

```yaml
---
apiVersion: apiextensions.k8s.io/v1
kind: ClusterObservability
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.32.0
  name: clusterobservability.opentelemetry.io
spec:
  group: opentelemetry.io
  names:
    kind: ClusterObservability
    listKind: ClusterObservabilityList
    plural: co11ys
    singular: co11y
  scope: Cluster
  versions:
  - additionalPrinterColumns:
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    - jsonPath: .spec.exporter.endpoint
      name: Endpoint
      type: string
    name: v1alpha1
    schema:
      openAPIV3Schema:
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            properties:
              exporter:
                <OTLP exporter config>
            type: object
        type: object
    served: true
    storage: true

```
## Rollout Plan

See https://github.com/open-telemetry/opentelemetry-operator/issues/3818

## Limitations

This managed resource is not aiming to please all users of the operator. This is a limited experience to deliver best practices and a comprehensive feature set encompassing many elements of OpenTelemetry.