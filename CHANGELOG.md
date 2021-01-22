Changes by Version
==================

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
