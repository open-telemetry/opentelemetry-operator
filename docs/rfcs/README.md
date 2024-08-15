# RFC Process

The RFC process for the OpenTelemetry Operator exists so that community members can effectively review and understand
major decisions made by the Operator SIG. The RFC process also allows users to comment asynchronously on design
decisions.

We aim to improve the experience for using OpenTelemetry tools in a Kubernetes
environment. Proposals here should only focus on improving that experience by composing existing
OpenTelemetry components such as the OpenTelemetry Collector, OpenTelemetry Instrumentation, OpAMP, etc.
If a proposal requires new components to exist that do not logically fit within that mission, it's recommended to
open proposals with the proper SIGs.

## Process

1. Copy and fill the [template.md](./template.md) document
2. Open a PR to get initial feedback on RFC
3. The RFC must be discussed at the Operator SIG Meeting at least once prior to merge
4. Upon merge, the RFC's status will still be Draft. At this point, the RFC has been accepted and an implementation
can be started
   1. The initial implementation's PR should change the status to accepted
   2. If any significant changes are made that deviate from the RFC, the RFC should be updated to reflect that

## Recommendations

During the RFC process, the [template.md](./template.md) must be filled out. We recommend also doing the following:

* Include a proof-of-concept to confirm the design
* Share alternatives considered and tradeoffs
  * A valid alternative to consider is always "do nothing"
* Pair with a SIG member to sort through unknowns / ask for help
