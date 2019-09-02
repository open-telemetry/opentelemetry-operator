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
			OtelSvcImageConfigKey,
			"quay.io/jpkroehling/opentelemetry-service:latest",
			"The default image to use for OpenTelemetry Service when not specified in the individual custom resource (CR)",
		)
		viper.BindPFlag(OtelSvcImageConfigKey, fs.Lookup(OtelSvcImageConfigKey))
	}

	return fs
}
