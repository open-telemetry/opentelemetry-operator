package upgrade

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

func upgrade0_0_2(client client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// this has the same content as `noop`, but it's added a separate function
	// to serve as template for versions with an actual upgrade procedure
	return otelcol, nil
}
