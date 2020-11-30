package upgrade

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

func upgrade0_15_0(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	delete(otelcol.Spec.Args, "--new-metrics")
	delete(otelcol.Spec.Args, "--legacy-metrics")
	return otelcol, nil
}
