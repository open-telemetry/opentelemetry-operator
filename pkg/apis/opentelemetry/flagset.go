package opentelemetry

import (
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

		fs = pflag.NewFlagSet("opentelemetry-operator", pflag.ExitOnError)
		fs.String(
			OtelColImageConfigKey,
			"quay.io/opentelemetry/opentelemetry-collector:v0.0.2",
			"The default image to use for OpenTelemetry Collector when not specified in the individual custom resource (CR)",
		)
		viper.BindPFlag(OtelColImageConfigKey, fs.Lookup(OtelColImageConfigKey))
	}

	return fs
}
