[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc] [![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/open-telemetry/opentelemetry-operator/badge)](https://securityscorecards.dev/viewer/?uri=github.com/open-telemetry/opentelemetry-operator)

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

The operator manages:

- [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
- [auto-instrumentation](https://opentelemetry.io/docs/concepts/instrumentation/automatic/) of the workloads using OpenTelemetry instrumentation libraries

## Documentation

User documentation lives under [`docs/`](docs/README.md):

- [Getting started](docs/getting-started/README.md) — installation, upgrades, compatibility
- [Concepts](docs/concepts/README.md) — what the operator manages
- [Collector](docs/collector/README.md) — `OpenTelemetryCollector` CRD: deployment modes, sidecar injection, observability
- [Auto-instrumentation](docs/auto-instrumentation/README.md) — `Instrumentation` CRD: per-language injection, resource attributes
- [Target Allocator](docs/target-allocator/README.md) — Prometheus scrape target distribution and discovery
- [OpAMP Bridge](docs/opamp-bridge/README.md) — `OpAMPBridge` CRD
- [Troubleshooting](docs/troubleshooting/README.md) — debug tips and known issues
- [Reference](docs/reference/README.md) — API docs, CRD changelog, feature gates
- [RFCs](docs/rfcs/README.md) — design proposals
- [Official OpenTelemetry Operator page](https://opentelemetry.io/docs/platforms/kubernetes/operator/)

## Helm Charts

You can install OpenTelemetry Operator via [Helm Chart](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-operator) from the opentelemetry-helm-charts repository. More information is available in [here](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-operator).

## Getting started

To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/) and run:

```bash
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create your first OpenTelemetry Collector instance. See [Creating an OpenTelemetry Collector](./docs/getting-started/collector.md) for the walkthrough.

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

In addition to the [core responsibilities](https://github.com/open-telemetry/community/blob/main/community-membership.md) the operator project requires approvers and maintainers to be responsible for releasing the project. See [RELEASE.md](./RELEASE.md) for more information and release schedule.

### Maintainers

- [Benedikt Bongartz](https://github.com/frzifus), Red Hat
- [Jacob Aronoff](https://github.com/jaronoff97), Omlet
- [Mikołaj Świątek](https://github.com/swiatekm), Elastic
- [Pavol Loffay](https://github.com/pavolloffay), Red Hat

For more information about the maintainer role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#maintainer).

### Approvers

- [Antoine Toulme](https://github.com/atoulme), Splunk
- [Israel Blancas](https://github.com/iblancasa), Coralogix
- [Tyler Helmuth](https://github.com/TylerHelmuth), Grafana Labs
- [Yuri Oliveira Sa](https://github.com/yuriolisa), OllyGarden

For more information about the approver role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#approver).

### Triagers

For more information about the triager role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#triager).

### Emeritus Maintainers

- [Alex Boten](https://github.com/codeboten)
- [Bogdan Drutu](https://github.com/BogdanDrutu)
- [Juraci Paixão Kröhling](https://github.com/jpkrohling)
- [Tigran Najaryan](https://github.com/tigrannajaryan)
- [Vineeth Pothulapati](https://github.com/VineethReddy02)

For more information about the emeritus role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#emeritus-maintainerapprovertriager).

### Emeritus Approvers

- [Anthony Mirabella](https://github.com/Aneurysm9)
- [Dmitrii Anoshin](https://github.com/dmitryax)
- [James Bebbington](https://github.com/james-bebbington)
- [Jay Camp](https://github.com/jrcamp)
- [Owais Lone](https://github.com/owais)
- [Pablo Baeyens](https://github.com/mx-psi)

For more information about the emeritus role, see the [community repository](https://github.com/open-telemetry/community/blob/main/guides/contributor/membership.md#emeritus-maintainerapprovertriager).

Thanks to all the people who already contributed!

[![Contributors][contributors-img]][contributors]

## License

[Apache 2.0 License](./LICENSE).

[github-workflow]: https://github.com/open-telemetry/opentelemetry-operator/actions
[github-workflow-img]: https://github.com/open-telemetry/opentelemetry-operator/workflows/Continuous%20Integration/badge.svg
[goreport-img]: https://goreportcard.com/badge/github.com/open-telemetry/opentelemetry-operator
[goreport]: https://goreportcard.com/report/github.com/open-telemetry/opentelemetry-operator
[godoc-img]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator?status.svg
[godoc]: https://godoc.org/github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#OpenTelemetryCollector
[contributors]: https://github.com/open-telemetry/opentelemetry-operator/graphs/contributors
[contributors-img]: https://contributors-img.web.app/image?repo=open-telemetry/opentelemetry-operator
