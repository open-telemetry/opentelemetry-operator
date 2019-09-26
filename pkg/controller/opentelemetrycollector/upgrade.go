package opentelemetrycollector

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/version"
)

func (r *ReconcileOpenTelemetryCollector) applyUpgrades(ctx context.Context) error {
	otelcol := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	currentVersions := version.Get()

	upgraded := false
	if len(otelcol.Status.Version) > 0 {
		if otelcol.Status.Version != currentVersions.OpenTelemetryCollector {
			// in theory, the version from the Status could be higher than currentVersions.Jaeger, but we let the upgrade routine
			// check/handle it
			u, err := upgrade.ManagedInstance(ctx, r.client, otelcol)
			if err != nil {
				return err
			}
			otelcol = u
			upgraded = true
		}
	}

	// at this point, the Jaeger we are managing is in sync with the Operator's version
	// if this is a new object, no upgrade was made, so, we just set the version
	otelcol.Status.Version = version.Get().OpenTelemetryCollector

	if upgraded {
		logger.V(2).Info("managed instance upgraded", "version", otelcol.Status.Version)
	}

	return nil
}
