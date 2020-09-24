package reconcile

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Self updates this instance's self data. This should be the last item in the reconciliation, as it causes changes
// making params.Instance obsolete. Default values should be set in the Defaulter webhook, this should only be used
// for the Status, which can't be set by the defaulter.
func Self(ctx context.Context, params Params) error {
	if params.Instance.Status.Version != "" {
		// a version is already set, let the upgrade mechanism take care of it!
		return nil
	}

	changed := params.Instance
	changed.Status.Version = version.OpenTelemetryCollector()

	// this is only a change for new instances: existing instances are reconciled when the operator is first started
	statusPatch := client.MergeFrom(&params.Instance)
	if err := params.Client.Status().Patch(ctx, &changed, statusPatch); err != nil {
		return fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}

	return nil
}
