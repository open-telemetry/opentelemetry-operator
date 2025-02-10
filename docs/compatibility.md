# Compatibility

This document details compatibility guarantees the OpenTelemetry Operator offers for its dependencies and platforms.

## Go

When productised as a go libary or custom distribution the OpenTelemetry Operator project attempts to follow the supported go versions as [defined by the Go team](https://go.dev/doc/devel/release#policy).

Similar to the [opentelemetry collector](https://github.com/open-telemetry/opentelemetry-collector?tab=readme-ov-file#compatibility), removing support for an unsupported Go version is not considered a breaking change.

Support for Go versions on the OpenTelemetry Operator is updated as follows:

    The first release after the release of a new Go minor version N will add build and tests steps for the new Go minor version.
    The first release after the release of a new Go minor version N will remove support for Go version N-2.

Official OpenTelemetry Operator binaries may be built with any supported Go version.

## Kubernetes

As a rule, the operator tries to be compatible with as wide a range of Kubernetes versions as possible.

We will *always* support all the versions maintained by the upstream Kubernetes project, as detailed on its [releases page][kubernetes_releases].

We will make every effort to support all Kubernetes versions maintained by popular distributions and hosted platforms. For example, you can realistically expect us to always support all versions offered by [OpenShift][openshift_support] and [AWS EKS][aws_support].

Whenever we do remove support for a Kubernetes version, we will give at least one month's notice beforehand.

The [compatibility matrix](#compatibility-matrix) below precisely shows the supported Kubernetes versions for each operator release.

## OpenTelemetry Operator vs. OpenTelemetry Collector

The OpenTelemetry Operator follows the same versioning as the operand (OpenTelemetry Collector) up to the minor part of the version. For example, the OpenTelemetry Operator v0.18.1 tracks OpenTelemetry Collector 0.18.0. The patch part of the version indicates the patch level of the operator itself, not that of OpenTelemetry Collector. Whenever a new patch version is released for OpenTelemetry Collector, we'll release a new patch version of the operator.

By default, the OpenTelemetry Operator ensures consistent versioning between itself and the managed `OpenTelemetryCollector` resources. That is, if the OpenTelemetry Operator is based on version `0.40.0`, it will create resources with an underlying OpenTelemetry Collector at version `0.40.0`.

When a custom `Spec.Image` is used with an `OpenTelemetryCollector` resource, the OpenTelemetry Operator will not manage this versioning and upgrading. In this scenario, it is best practice that the OpenTelemetry Operator version should match the underlying core version. Given a `OpenTelemetryCollector` resource with a `Spec.Image` configured to a custom image based on underlying OpenTelemetry Collector at version `0.40.0`, it is recommended that the OpenTelemetry Operator is kept at version `0.40.0`.

## Compatibility matrix

We use `cert-manager` for some features of this operator and the third column shows the versions of the `cert-manager` that are known to work with this operator's versions.

The Target Allocator supports prometheus-operator CRDs like ServiceMonitor, and it does so by using packages imported from prometheus-operator itself. The table shows which version is shipped with a given operator version.
Generally speaking, these are backwards compatible, but specific features require the appropriate package versions.

The OpenTelemetry Operator _might_ work on versions outside of the given range, but when opening new issues, please make sure to test your scenario on a supported version.

| OpenTelemetry Operator | Kubernetes     | Cert-Manager | Prometheus-Operator |
|------------------------|----------------|--------------|---------------------|
| v0.118.0               | v1.23 to v1.32 | v1           | v0.76.2             |
| v0.117.0               | v1.23 to v1.32 | v1           | v0.76.2             |
| v0.116.0               | v1.23 to v1.31 | v1           | v0.76.2             |
| v0.115.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.114.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.113.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.112.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.111.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.110.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.109.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.108.0               | v1.23 to v1.31 | v1           | v0.76.0             |
| v0.107.0               | v1.23 to v1.30 | v1           | v0.75.0             |
| v0.106.0               | v1.23 to v1.30 | v1           | v0.75.0             |
| v0.105.0               | v1.23 to v1.30 | v1           | v0.74.0             |
| v0.104.0               | v1.23 to v1.30 | v1           | v0.74.0             |
| v0.103.0               | v1.23 to v1.30 | v1           | v0.74.0             |
| v0.102.0               | v1.23 to v1.30 | v1           | v0.71.2             |
| v0.101.0               | v1.23 to v1.30 | v1           | v0.71.2             |
| v0.100.0               | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.99.0                | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.98.0                | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.97.0                | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.96.0                | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.95.0                | v1.23 to v1.29 | v1           | v0.71.2             |
| v0.94.0                | v1.23 to v1.29 | v1           | v0.71.0             |

[kubernetes_releases]: https://kubernetes.io/releases/
[openshift_support]: https://access.redhat.com/support/policy/updates/openshift
[aws_support]: https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html
