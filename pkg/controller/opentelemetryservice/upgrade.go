package opentelemetryservice

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/version"
)

func (r *ReconcileOpenTelemetryService) applyUpgrades(ctx context.Context) error {
	otelsvc := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryService)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	currentVersions := version.Get()

	upgraded := false
	if len(otelsvc.Status.Version) > 0 {
		if otelsvc.Status.Version != currentVersions.OpenTelemetryService {
			// in theory, the version from the Status could be higher than currentVersions.Jaeger, but we let the upgrade routine
			// check/handle it
			u, err := upgrade.ManagedInstance(ctx, r.client, otelsvc)
			if err != nil {
				return err
			}
			otelsvc = u
			upgraded = true
		}
	}

	// at this point, the Jaeger we are managing is in sync with the Operator's version
	// if this is a new object, no upgrade was made, so, we just set the version
	otelsvc.Status.Version = version.Get().OpenTelemetryService

	if upgraded {
		logger.V(2).Info("managed instance upgraded", "version", otelsvc.Status.Version)
	}

	return nil
}
