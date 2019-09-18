package version

import (
	"fmt"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
)

var (
	version   string
	buildDate string
	otelSvc   string
)

// Version holds this Operator's version as well as the version of some of the components it uses
type Version struct {
	Operator             string `json:"opentelemetry-operator"`
	BuildDate            string `json:"build-date"`
	OpenTelemetryService string `json:"opentelemetry-service-version"`
	Go                   string `json:"go-version"`
	OperatorSdk          string `json:"operator-sdk-version"`
}

// Get returns the Version object with the relevant information
func Get() Version {
	return Version{
		Operator:             version,
		BuildDate:            buildDate,
		OpenTelemetryService: DefaultOpenTelemetryService(),
		Go:                   runtime.Version(),
		OperatorSdk:          sdkVersion.Version,
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', OpenTelemetryService='%v', Go='%v', OperatorSDK='%v')",
		v.Operator,
		v.BuildDate,
		v.OpenTelemetryService,
		v.Go,
		v.OperatorSdk,
	)
}

// DefaultOpenTelemetryService returns the default OpenTelemetryService to use when no versions are specified via CLI or configuration
func DefaultOpenTelemetryService() string {
	if len(otelSvc) > 0 {
		// this should always be set, as it's specified during the build
		return otelSvc
	}

	// fallback value, useful for tests
	return "0.0.0"
}
