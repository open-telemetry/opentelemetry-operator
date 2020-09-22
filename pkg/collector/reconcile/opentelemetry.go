package reconcile

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Self updates this instance's self data. This should be the last item in the reconciliation, as it causes changes
// making params.Instance obsolete
func Self(ctx context.Context, params Params) error {
	changed := params.Instance

	labels := changed.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	// if it's not empty, and it's not opentelemetry-operator, then it's set to something else,
	// and we don't want to change it
	if labels["app.kubernetes.io/managed-by"] != "" && labels["app.kubernetes.io/managed-by"] != "opentelemetry-operator" {
		// don't change anything
		return nil
	}

	labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	changed.Labels = labels
	changed.Status.Version = version.OpenTelemetryCollector()

	patch := client.MergeFrom(&params.Instance)
	if err := params.Client.Patch(ctx, &changed, patch); err != nil {
		return fmt.Errorf("failed to apply changes to the OpenTelemetry CR: %w", err)
	}

	// this is only a change for new instances: existing instances are reconciled when the operator is first started
	statusPatch := client.MergeFrom(&params.Instance)
	if err := params.Client.Status().Patch(ctx, &changed, statusPatch); err != nil {
		return fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}

	return nil
}
