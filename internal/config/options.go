package config

import (
	"time"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

// Option represents one specific configuration option
type Option func(c *options)

type options struct {
	autoDetect              autodetect.AutoDetect
	autoDetectFrequency     time.Duration
	collectorImage          string
	collectorConfigMapEntry string
	logger                  logr.Logger
	onChange                []func() error
	platform                platform.Platform
	version                 version.Version
	watchedNamespaces       []string
}

func WithAutoDetect(a autodetect.AutoDetect) Option {
	return func(o *options) {
		o.autoDetect = a
	}
}
func WithAutoDetectFrequency(t time.Duration) Option {
	return func(o *options) {
		o.autoDetectFrequency = t
	}
}
func WithCollectorImage(s string) Option {
	return func(o *options) {
		o.collectorImage = s
	}
}
func WithCollectorConfigMapEntry(s string) Option {
	return func(o *options) {
		o.collectorConfigMapEntry = s
	}
}
func WithLogger(logger logr.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
func WithOnChange(f func() error) Option {
	return func(o *options) {
		if o.onChange == nil {
			o.onChange = []func() error{}
		}
		o.onChange = append(o.onChange, f)
	}
}
func WithPlatform(plt platform.Platform) Option {
	return func(o *options) {
		o.platform = plt
	}
}
func WithVersion(v version.Version) Option {
	return func(o *options) {
		o.version = v
	}
}
func WithWatchedNamespaces(nss []string) Option {
	return func(o *options) {
		o.watchedNamespaces = nss
	}
}
