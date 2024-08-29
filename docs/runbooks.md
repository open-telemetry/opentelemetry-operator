# OpenTelemetry Operator Runbooks

See [`README.md`](../README.md) for more details about the OpenTelemetry Operator.

## Manager Rules

### [ReconcileErrors](#reconcileerrors)

|||
|-:|-|
| Meaning | The OpenTelemetry Operator cannot succeed in the reconciliation step, probably because of a misconfigured OpenTelemetryCollector. |
| Impact | No impact on already running deployments or new correct ones. |
| Diagnosis | Check manager logs for reasons why this might happen. |
| Mitigation | Find out which OpenTelemetryCollector is causing the errors and fix the config. |

### [WorkqueueDepth](#workqueuedepth)

|||
|-:|-|
| Meaning | The working queue for the operator is larger than 0. |
| Impact | No impact if the queue depth reverts to 0 quickly. More investigation is needed if the problem persists. |
| Diagnosis | Check manager logs for reasons why this might happen. |
| Mitigation | This could be caused by many errors. Act based on what the logs are showing. |