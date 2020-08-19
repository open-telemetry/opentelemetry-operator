package upgrade

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// Version represents a known OpenTelemetry Collector version
type Version struct {
	// Tag represents the tag for this version, like: 0.0.1
	Tag string

	upgrade func(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error)
	next    *Version
}

var (
	v0_0_1 = Version{Tag: "0.0.1", upgrade: noop, next: &v0_0_2}
	v0_0_2 = Version{Tag: "0.0.2", upgrade: noop, next: &v0_2_0}
	v0_2_0 = Version{Tag: "0.2.0", upgrade: upgrade0_2_0}

	// Latest represents the latest known version for the OpenTelemetry Collector
	Latest = &v0_2_0

	versions = map[string]Version{
		v0_0_1.Tag: v0_0_1,
		v0_0_2.Tag: v0_0_2,
		v0_2_0.Tag: v0_2_0,
	}
)

func noop(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	return otelcol, nil
}
