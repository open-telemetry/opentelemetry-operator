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

// Options represents the possible options for the configuration
var Options options

type options struct {
	autoDetect              autodetect.AutoDetect
	autoDetectFrequency     time.Duration
	collectorImage          string
	collectorConfigMapEntry string
	logger                  logr.Logger
	onChange                []func() error
	platform                platform.Platform
	version                 version.Version
}

func (options) AutoDetect(a autodetect.AutoDetect) Option {
	return func(o *options) {
		o.autoDetect = a
	}
}
func (options) AutoDetectFrequency(t time.Duration) Option {
	return func(o *options) {
		o.autoDetectFrequency = t
	}
}
func (options) CollectorImage(s string) Option {
	return func(o *options) {
		o.collectorImage = s
	}
}
func (options) CollectorConfigMapEntry(s string) Option {
	return func(o *options) {
		o.collectorConfigMapEntry = s
	}
}
func (options) Logger(logger logr.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
func (options) OnChange(f func() error) Option {
	return func(o *options) {
		if o.onChange == nil {
			o.onChange = []func() error{}
		}
		o.onChange = append(o.onChange, f)
	}
}
func (options) Platform(plt platform.Platform) Option {
	return func(o *options) {
		o.platform = plt
	}
}
func (options) Version(v version.Version) Option {
	return func(o *options) {
		o.version = v
	}
}
