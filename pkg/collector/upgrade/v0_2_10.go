package upgrade

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// this is our first version under otel/opentelemetry-collector
func upgrade0_2_10(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// this is a no-op, but serves to keep the skeleton here for the future versions
	return otelcol, nil
}
