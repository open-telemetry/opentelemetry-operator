Changes by Version
==================

0.33.0 (2021-08-20)
-------------------
* Bumped OpenTelemetry Collector to v0.33.0
* Add statsd receiver ([#364](https://github.com/open-telemetry/opentelemetry-operator/pull/364), [@VineethReddy02](https://github.com/VineethReddy02))
* Allow running daemonset in hostNetwork mode ([#393](https://github.com/open-telemetry/opentelemetry-operator/pull/393), [@owais](https://github.com/owais))
* Target Allocator implementation (Part 1 - OTEL Operator Enhancements) ([#351](https://github.com/open-telemetry/opentelemetry-operator/pull/351), [@]())
* Change the default port for OTLP HTTP ([#373](https://github.com/open-telemetry/opentelemetry-operator/pull/373), [@joaopgrassi](https://github.com/joaopgrassi))
* Add Kubernetes 1.22 to the test matrix ([#382](https://github.com/open-telemetry/opentelemetry-operator/pull/382), [@jpkrohling](https://github.com/jpkrohling))
* Add `protocol: TCP` value under `ports` key to avoid the known limitation for Kubernetes 1.19 ([#372](https://github.com/open-telemetry/opentelemetry-operator/pull/372), [@Saber-W](https://github.com/Saber-W))
* Add fluentforward receiver ([#367](https://github.com/open-telemetry/opentelemetry-operator/pull/367), [@VineethReddy02](https://github.com/VineethReddy02))

0.32.0
-------------------
* We skipped this release.

0.31.0 (2021-07-29)
-------------------
* Bumped OpenTelemetry Collector to v0.31.0

0.30.0 (2021-07-15)
-------------------
* Bumped OpenTelemetry Collector to v0.30.0
* Container Security Context ([#332](https://github.com/open-telemetry/opentelemetry-operator/pull/332), [@owais](https://github.com/owais))

0.29.0 (2021-06-25)
-------------------
* Bumped OpenTelemetry Collector to v0.29.0
* Add delete webhook ([#313](https://github.com/open-telemetry/opentelemetry-operator/pull/313), [@VineethReddy02](https://github.com/VineethReddy02))

0.28.0 (2021-06-12)
-------------------
* Bumped OpenTelemetry Collector to v0.28.0
* Tolerations support in OpenTelemetryCollector CRD ([#302](https://github.com/open-telemetry/opentelemetry-operator/pull/302), [@VineethReddy02](https://github.com/VineethReddy02))
* Copy desired service ports when reconciling ([#299](https://github.com/open-telemetry/opentelemetry-operator/pull/299), [@thib92](https://github.com/thib92))
* Remove the OTLP receiver legacy gRPC port(55680) references ([#293](https://github.com/open-telemetry/opentelemetry-operator/pull/293), [@mxiamxia](https://github.com/mxiamxia))

0.27.0 (2021-05-20)
-------------------
* Bumped OpenTelemetry Collector to v0.27.0

0.26.0 (2021-05-12)
-------------------
* Bumped OpenTelemetry Collector to v0.26.0

0.25.0 (2021-05-06)
-------------------
* Bumped OpenTelemetry Collector to v0.25.0

0.24.0 (2021-04-20)
-------------------
* Bumped OpenTelemetry Collector to v0.24.0 ([#251](https://github.com/open-telemetry/opentelemetry-operator/pull/251), [@jnodorp-jaconi](https://github.com/jnodorp-jaconi))
* Allow resource configuration on collector spec ([#248](https://github.com/open-telemetry/opentelemetry-operator/pull/248), [@jnodorp-jaconi](https://github.com/jnodorp-jaconi))

0.23.0 (2021-04-04)
-------------------
* Bumped OpenTelemetry Collector to v0.23.0

0.22.0 (2021-03-11)
-------------------
* Bumped OpenTelemetry Collector to v0.22.0

0.21.0 (2021-03-09)
-------------------
* Bumped OpenTelemetry Collector to v0.21.0
* Restart collector pod when config is updated ([#215](https://github.com/open-telemetry/opentelemetry-operator/pull/215), [@bhiravabhatla](https://github.com/bhiravabhatla))
* Add permissions for opentelemetry finalizer resource ([#212](https://github.com/open-telemetry/opentelemetry-operator/pull/212), [@rubenvp8510](https://github.com/rubenvp8510))
* fix: collector selection should not fail if there is a single sidecar ([#210](https://github.com/open-telemetry/opentelemetry-operator/pull/210), [@vbehar](https://github.com/vbehar))

0.20.0 (2021-02-11)
-------------------
* Bumped OpenTelemetry Collector to v0.20.0
* Add correct boundary to integer parsing ([#187](https://github.com/open-telemetry/opentelemetry-operator/pull/187), [@jpkrohling](https://github.com/jpkrohling))

0.19.0 (2021-01-27)
-------------------
* Bumped OpenTelemetry Collector to v0.19.0


0.18.1 (2021-01-25)
-------------------
* Fixed testing image from being used in non-test artifacts (fixes #170) ([#171](https://github.com/open-telemetry/opentelemetry-operator/pull/171), [@gramidt](https://github.com/gramidt))


0.18.0 (2021-01-22)
-------------------
* Bumped OpenTelemetry Collector to v0.18.0 ([#169](https://github.com/open-telemetry/opentelemetry-operator/pull/169), [@jpkrohling](https://github.com/jpkrohling))


0.17.1 (2020-12-17)
-------------------
* Set env vars correctly in workflow steps ([#152](https://github.com/open-telemetry/opentelemetry-operator/pull/152), [@jpkrohling](https://github.com/jpkrohling))
* Add permissions for leases.coordination.k8s.io ([#151](https://github.com/open-telemetry/opentelemetry-operator/pull/151), [@jpkrohling](https://github.com/jpkrohling))
* Adjust container image tags ([#148](https://github.com/open-telemetry/opentelemetry-operator/pull/148), [@jpkrohling](https://github.com/jpkrohling))

0.17.0 (2020-12-16)
-------------------
* Bumped OpenTelemetry Collector to v0.17.0 ([#144](https://github.com/open-telemetry/opentelemetry-operator/pull/144), [@jpkrohling](https://github.com/jpkrohling))
* Refactor how images are pushed ([#138](https://github.com/open-telemetry/opentelemetry-operator/pull/138), [@jpkrohling](https://github.com/jpkrohling))

0.16.0 (2020-12-03)
-------------------
* Bumped OpenTelemetry Collector to v0.16.0 ([#135](https://github.com/open-telemetry/opentelemetry-operator/pull/135), [@jpkrohling](https://github.com/jpkrohling))
* Fix image prefix for release image ([#133](https://github.com/open-telemetry/opentelemetry-operator/pull/133), [@jpkrohling](https://github.com/jpkrohling))
* Explicitly set Service Port Protocol for Jaeger Receivers ([#117](https://github.com/open-telemetry/opentelemetry-operator/pull/117), [@KingJ](https://github.com/KingJ))

_Note: The default port for the OTLP receiver has been changed from 55680 to 4317. To keep compatibility with your existing workload, the operator is now generating a service with the two port numbers by default. Both have 4317 as the target port._

0.15.0 (2020-11-27)
-------------------
* Bumped OpenTelemetry Collector to v0.15.0 ([#131](https://github.com/open-telemetry/opentelemetry-operator/pull/131), [@jpkrohling](https://github.com/jpkrohling)) 

0.14.0 (2020-11-09)
-------------------
* Bumped OpenTelemetry Collector to v0.14.0 ([#112](https://github.com/open-telemetry/opentelemetry-operator/pull/112), [@jpkrohling](https://github.com/jpkrohling)) 

_Note: The `tailsampling` processor was moved to the contrib repository, requiring a manual intervention in case this processor is being used: either replace the image with the contrib one (v0.14.0, which includes this processor), or remove the processor._

0.13.0 (2020-10-22)
-------------------

* Bumped OpenTelemetry Collector to v0.13.0 ([#101](https://github.com/open-telemetry/opentelemetry-operator/pull/101), [@dengliming](https://github.com/dengliming)) 
* Allow for spec.Env to be set on the OTEL Collector Spec ([#94](https://github.com/open-telemetry/opentelemetry-operator/pull/94), [@ekarlso](https://github.com/ekarlso))

_Note: The `groupbytrace` processor was moved to the contrib repository, requiring a manual intervention in case this processor is being used: either replace the image with the contrib one (v0.13.1, which includes this processor), or remove the processor._

0.12.0 (2020-10-12)
-------------------

* Bumped OpenTelemetry Collector to v0.12.0 ([#81](https://github.com/open-telemetry/opentelemetry-operator/pull/81), [@jpkrohling](https://github.com/jpkrohling))
* Remove use of deprecated controller runtime log API ([#78](https://github.com/open-telemetry/opentelemetry-operator/pull/78), [@bvwells](https://github.com/bvwells))

0.11.0 (2020-09-30)
-------------------

- Initial release after the migration to `kubebuilder`
- Support for OpenTelemetry Collector v0.11.0
- Features:
  - Provisioning of an OpenTelemetry Collector based on the CR definition
  - Sidecar injected via webhook
  - Deployment modes: `daemonset`, `deployment`, `sidecar`
  - Automatic upgrade between collector versions
- CRs from the older version should still work with this operator
