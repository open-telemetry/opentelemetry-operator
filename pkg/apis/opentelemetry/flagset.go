package opentelemetry

import (
	"fmt"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-operator/pkg/version"
)

var (
	mu sync.Mutex
	fs *pflag.FlagSet
)

// FlagSet returns this operator's flags
func FlagSet() *pflag.FlagSet {
	if nil == fs {
		mu.Lock()
		defer mu.Unlock()

		otelColVersion := version.Get().OpenTelemetryCollector

		fs = pflag.NewFlagSet("opentelemetry-operator", pflag.ExitOnError)
		fs.String(
			OtelColImageConfigKey,
			fmt.Sprintf("quay.io/opentelemetry/opentelemetry-collector:v%s", otelColVersion),
			"The default image to use for OpenTelemetry Collector when not specified in the individual custom resource (CR)",
		)
		// #nosec G104 (CWE-703): Errors unhandled.
		viper.BindPFlag(OtelColImageConfigKey, fs.Lookup(OtelColImageConfigKey))
	}

	return fs
}

// ResetFlagSet will set the cached flagset to nil
func ResetFlagSet() {
	fs = nil
}
