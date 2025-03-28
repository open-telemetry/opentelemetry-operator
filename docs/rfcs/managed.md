# Managed CRD

**Status:** *Draft*

**Author:** Antoine Toulme (@atoulme)

**Date:** 2025-03-20

## Objective

Condense all best practices and deployment patterns of OpenTelemetry components into a single custom resource
that covers the feature set of the OpenTelemetry Helm chart.

## Summary
This request for comment aims to establish a new approach to management of resources in Kubernetes clusters,
with the explicit goal of taking over the whole cluster, and making decisions
on how to best instrument all resources in the cluster on behalf of the administrator based on best practices,
OpenTelemetry standards and semantic conventions.

The operator project offers custom resources (CR) to allow Kubernetes operators to define OpenTelemetry components
in their Kubernetes environments. Customers however do not want to manage components themselves, especially if they need
to think about component versioning and upgrade patterns, and would rather have an out-of-the-box experience that 
requires little insight and gets them to a working state effortlessly.

## Goals and non-goals

### Give a comprehensive, out-of the box experience of OpenTelemetry for a first-time user
The goal is to delight first-time customers by giving them a complete view of what is possible with OpenTelemetry.

### Do not offer a way to turn on and off specific features
We mean to offer a specific functional coverage of a specific use case that offers best practices of OpenTelemetry.

We don't allow to toggle features in and out specifically because we want to avoid increasing the need for functional test coverage.

If customers want to increase their level of control of the feature set, they can remove the managed custom resource and instead
follow the traditional route of setting up collectors, target allocators, instrumentation custom resources as is possible today.

## Use cases for proposal

### Initial installation

The customer wants to install the operator and immediately get value from the installation.

Upon installation, they install the custom resource with information on an OTLP endpoint where they can direct all data.

### Upgrade

The customer wants to upgrade their operator. All assets are upgraded out of the box in concert as part of the upgrade,
and are guaranteed to work through functional test coverage.

### Deletion and reinstallation

The customer can cleanly delete the custom resource, which triggers the deletion of all OpenTelemetry assets under
management by the operator. No stragglers are left.

### Kubernetes metrics

The customer upon installation of the custom resource can see Kubernetes cluster metrics as reported by the k8sclusterreceiver with default configuration.

### Kubeletstats metrics

The customer upon installation receives metrics from all Kubeletstats metrics from the default configuration of the kubeletstats receiver.

### Automatic instrumentation

All standard CRDs for instrumentations are all enabled by default for all namespaces.

### Node host metrics

The customer upon installation receives metrics from hostmetrics receiver with its default configuration.

Note the metrics map to the host, meaning the collector will have access to host volumes to scrape their utilization.

### Node logs

The customer upon installation will receive all logs from Kubernetes, scraped with the filelog receiver from the file system.

### Kubernetes entities

When entities become viable, the customer will receive Kubernetes entities as reported by the k8sclusterreceiver.

## Struct Design

```go
import "go.opentelemetry.io/collector/exporter/otlpexporter"

type ManagedCustomResource struct {
	ExporterConfig otlpexporter.Config `yaml:"exporter"`
}
```

```yaml
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.32.0
  name: managed.opentelemetry.io
spec:
  group: opentelemetry.io
  names:
    kind: Managed
    listKind: ManagedList
    plural: manageds
    singular: managed
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