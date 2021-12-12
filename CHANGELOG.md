Changes by Version
==================

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
