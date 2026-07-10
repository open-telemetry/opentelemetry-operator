# Upgrades

The OpenTelemetry Collector format continues to evolve. The operator makes a best-effort attempt to upgrade all managed OpenTelemetryCollector resources.

In certain scenarios, it may be desirable to prevent the operator from upgrading certain `OpenTelemetryCollector` resources. For example, when a resource is configured with a custom `.Spec.Image`, end users may wish to manage configuration themselves as opposed to having the operator upgrade it. This can be configured on a resource by resource basis with the exposed property `.Spec.UpgradeStrategy`.

By configuring a resource's `.Spec.UpgradeStrategy` to `none`, the operator will skip the given instance during the upgrade routine.

The default and only other acceptable value for `.Spec.UpgradeStrategy` is `automatic`.
