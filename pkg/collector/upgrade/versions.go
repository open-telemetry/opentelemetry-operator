package upgrade

import (
	"github.com/Masterminds/semver/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

type upgradeFunc func(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error)

type otelcolVersion struct {
	semver.Version
	upgrade upgradeFunc
}

var (
	versions = []otelcolVersion{
		{
			Version: *semver.MustParse("0.2.10"),
			upgrade: upgrade0_2_10,
		},
		{
			Version: *semver.MustParse("0.9.0"),
			upgrade: upgrade0_9_0,
		},
		{
			Version: *semver.MustParse("0.15.0"),
			upgrade: upgrade0_15_0,
		},
	}

	// Latest represents the latest version that we need to upgrade. This is not necessarily the latest known version.
	Latest = versions[len(versions)-1]
)
