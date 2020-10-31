// Package version contains the operator's version, as well as versions of underlying components
package version

import (
	"fmt"
	"runtime"
)

var (
	version   string
	buildDate string
	otelCol   string
)

// Version holds this Operator's version as well as the version of some of the components it uses
type Version struct {
	Operator               string `json:"opentelemetry-operator"`
	BuildDate              string `json:"build-date"`
	OpenTelemetryCollector string `json:"opentelemetry-collector-version"`
	Go                     string `json:"go-version"`
}

// Get returns the Version object with the relevant information
func Get() Version {
	return Version{
		Operator:               version,
		BuildDate:              buildDate,
		OpenTelemetryCollector: OpenTelemetryCollector(),
		Go:                     runtime.Version(),
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', OpenTelemetryCollector='%v', Go='%v')",
		v.Operator,
		v.BuildDate,
		v.OpenTelemetryCollector,
		v.Go,
	)
}

// OpenTelemetryCollector returns the default OpenTelemetryCollector to use when no versions are specified via CLI or configuration
func OpenTelemetryCollector() string {
	if len(otelCol) > 0 {
		// this should always be set, as it's specified during the build
		return otelCol
	}

	// fallback value, useful for tests
	return "0.0.0"
}
