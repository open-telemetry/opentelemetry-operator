Changes by Version
==================

0.11.0 (unreleased)
-------------------

- Initial release after the migration to `kubebuilder`
- Support for OpenTelemetry Collector v0.11.0
- Features:
  - Provisioning of an OpenTelemetry Collector based on the CR definition
  - Sidecar injected via webhook
  - Deployment modes: `daemonset`, `deployment`, `sidecar`
  - Automatic upgrade between collector versions
- CRs from the older version should still work with this operator
