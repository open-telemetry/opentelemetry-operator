// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package config contains the operator's runtime configuration.
package config

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/signalfx/splunk-otel-operator/internal/version"
	"github.com/signalfx/splunk-otel-operator/pkg/autodetect"
	"github.com/signalfx/splunk-otel-operator/pkg/platform"
)

const (
	defaultAutoDetectFrequency           = 5 * time.Second
	defaultCollectorConfigMapEntry       = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry = "targetallocator.yaml"
)

// Config holds the static configuration for this operator.
type Config struct {
	// Registers a callback, to be called once a configuration change happens
	OnChange func() error

	logger              logr.Logger
	autoDetect          autodetect.AutoDetect
	autoDetectFrequency time.Duration
	onChange            []func() error

	// config state
	collectorImage                string
	collectorConfigMapEntry       string
	targetAllocatorImage          string
	targetAllocatorConfigMapEntry string
	platform                      platform.Platform
	version                       version.Version
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		autoDetectFrequency:           defaultAutoDetectFrequency,
		collectorConfigMapEntry:       defaultCollectorConfigMapEntry,
		targetAllocatorConfigMapEntry: defaultTargetAllocatorConfigMapEntry,
		logger:                        logf.Log.WithName("config"),
		platform:                      platform.Unknown,
		version:                       version.Get(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	// this is derived from another option, so, we need to first parse the options, then set a default
	// if there's no explicit value being set
	if len(o.collectorImage) == 0 {
		o.collectorImage = fmt.Sprintf("quay.io/signalfx/splunk-otel-collector:%s", o.version.SplunkOtelCollector)
	}

	if len(o.targetAllocatorImage) == 0 {
		o.targetAllocatorImage = fmt.Sprintf("quay.io/opentelemetry/target-allocator:%s", o.version.TargetAllocator)
	}

	return Config{
		autoDetect:                    o.autoDetect,
		autoDetectFrequency:           o.autoDetectFrequency,
		collectorImage:                o.collectorImage,
		collectorConfigMapEntry:       o.collectorConfigMapEntry,
		targetAllocatorImage:          o.targetAllocatorImage,
		targetAllocatorConfigMapEntry: o.targetAllocatorConfigMapEntry,
		logger:                        o.logger,
		onChange:                      o.onChange,
		platform:                      o.platform,
		version:                       o.version,
	}
}

// FlagSet binds the flags to the user-modifiable values of the operator's configuration.
func (c *Config) FlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("splunk-otel-operator", pflag.ExitOnError)
	pflag.StringVar(&c.collectorImage,
		"otelcol-image",
		c.collectorImage,
		"The default image to use for Splunk OpenTelemetry Collector when not specified in the individual custom resource (CR)",
	)

	return fs
}

// StartAutoDetect attempts to automatically detect relevant information for this operator. This will block until the first
// run is executed and will schedule periodic updates.
func (c *Config) StartAutoDetect() error {
	err := c.AutoDetect()
	go c.periodicAutoDetect()

	return err
}

func (c *Config) periodicAutoDetect() {
	ticker := time.NewTicker(c.autoDetectFrequency)

	for range ticker.C {
		if err := c.AutoDetect(); err != nil {
			c.logger.Info("auto-detection failed", "error", err)
		}
	}
}

// AutoDetect attempts to automatically detect relevant information for this operator.
func (c *Config) AutoDetect() error {
	changed := false
	c.logger.V(2).Info("auto-detecting the configuration based on the environment")

	// TODO: once new things need to be detected, extract this into individual detection routines
	if c.platform == platform.Unknown {
		plt, err := c.autoDetect.Platform()
		if err != nil {
			return err
		}

		if c.platform != plt {
			c.logger.V(1).Info("platform detected", "platform", plt)
			c.platform = plt
			changed = true
		}
	}

	if changed {
		for _, callback := range c.onChange {
			if err := callback(); err != nil {
				// we don't fail if the callback failed, as the auto-detection itself
				// did work
				c.logger.Error(err, "configuration change notification failed for callback")
			}
		}
	}

	return nil
}

// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
func (c *Config) CollectorImage() string {
	return c.collectorImage
}

// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
func (c *Config) CollectorConfigMapEntry() string {
	return c.collectorConfigMapEntry
}

// TargetAllocatorImage represents the flag to override the OpenTelemetry TargetAllocator container image.
func (c *Config) TargetAllocatorImage() string {
	return c.targetAllocatorImage
}

// TargetAllocatorConfigMapEntry represents the configuration file name for the TargetAllocator. Immutable.
func (c *Config) TargetAllocatorConfigMapEntry() string {
	return c.targetAllocatorConfigMapEntry
}

// Platform represents the type of the platform this operator is running.
func (c *Config) Platform() platform.Platform {
	return c.platform
}

// Version holds the versions used by this operator.
func (c *Config) Version() version.Version {
	return c.version
}
