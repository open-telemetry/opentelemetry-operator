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
	"sync"
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

const (
	defaultAutoDetectFrequency           = 5 * time.Second
	defaultCollectorConfigMapEntry       = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry = "targetallocator.yaml"
)

// Config holds the static configuration for this operator.
type Config struct {
	autoDetect                     autodetect.AutoDetect
	logger                         logr.Logger
	targetAllocatorImage           string
	autoInstrumentationPythonImage string
	collectorImage                 string
	collectorConfigMapEntry        string
	autoInstrumentationDotNetImage string
	targetAllocatorConfigMapEntry  string
	autoInstrumentationNodeJSImage string
	autoInstrumentationJavaImage   string
	onPlatformChange               changeHandler
	labelsFilter                   []string
	platform                       platformStore
	autoDetectFrequency            time.Duration
	autoscalingVersion             autodetect.AutoscalingVersion
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		autoDetectFrequency:           defaultAutoDetectFrequency,
		collectorConfigMapEntry:       defaultCollectorConfigMapEntry,
		targetAllocatorConfigMapEntry: defaultTargetAllocatorConfigMapEntry,
		logger:                        logf.Log.WithName("config"),
		platform:                      newPlatformWrapper(),
		version:                       version.Get(),
		autoscalingVersion:            autodetect.DefaultAutoscalingVersion,
		onPlatformChange:              newOnChange(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	return Config{
		autoDetect:                     o.autoDetect,
		autoDetectFrequency:            o.autoDetectFrequency,
		collectorImage:                 o.collectorImage,
		collectorConfigMapEntry:        o.collectorConfigMapEntry,
		targetAllocatorImage:           o.targetAllocatorImage,
		targetAllocatorConfigMapEntry:  o.targetAllocatorConfigMapEntry,
		logger:                         o.logger,
		onPlatformChange:               o.onPlatformChange,
		platform:                       o.platform,
		autoInstrumentationJavaImage:   o.autoInstrumentationJavaImage,
		autoInstrumentationNodeJSImage: o.autoInstrumentationNodeJSImage,
		autoInstrumentationPythonImage: o.autoInstrumentationPythonImage,
		autoInstrumentationDotNetImage: o.autoInstrumentationDotNetImage,
		labelsFilter:                   o.labelsFilter,
		autoscalingVersion:             o.autoscalingVersion,
	}
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
	c.logger.V(2).Info("auto-detecting the configuration based on the environment")

	plt, err := c.autoDetect.Platform()
	if err != nil {
		return err
	}

	if c.platform.Get() != plt {
		c.logger.V(1).Info("platform detected", "platform", plt)
		c.platform.Set(plt)
		if err = c.onPlatformChange.Do(); err != nil {
			// Don't fail if the callback failed, as auto-detection itself worked.
			c.logger.Error(err, "configuration change notification failed for callback")
		}
	}

	hpaVersion, err := c.autoDetect.HPAVersion()
	if err != nil {
		return err
	}
	c.autoscalingVersion = hpaVersion
	c.logger.V(2).Info("autoscaling version detected", "autoscaling-version", c.autoscalingVersion.String())

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
	return c.platform.Get()
}

// AutoscalingVersion represents the preferred version of autoscaling.
func (c *Config) AutoscalingVersion() autodetect.AutoscalingVersion {
	return c.autoscalingVersion
}

// AutoInstrumentationJavaImage returns OpenTelemetry Java auto-instrumentation container image.
func (c *Config) AutoInstrumentationJavaImage() string {
	return c.autoInstrumentationJavaImage
}

// AutoInstrumentationNodeJSImage returns OpenTelemetry NodeJS auto-instrumentation container image.
func (c *Config) AutoInstrumentationNodeJSImage() string {
	return c.autoInstrumentationNodeJSImage
}

// AutoInstrumentationPythonImage returns OpenTelemetry Python auto-instrumentation container image.
func (c *Config) AutoInstrumentationPythonImage() string {
	return c.autoInstrumentationPythonImage
}

// AutoInstrumentationDotNetImage returns OpenTelemetry DotNet auto-instrumentation container image.
func (c *Config) AutoInstrumentationDotNetImage() string {
	return c.autoInstrumentationDotNetImage
}

// Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
func (c *Config) LabelsFilter() []string {
	return c.labelsFilter
}

// RegisterPlatformChangeCallback registers the given function as a callback that
// is called when the platform detection detects a change.
func (c *Config) RegisterPlatformChangeCallback(f func() error) {
	c.onPlatformChange.Register(f)
}

type platformStore interface {
	Set(plt platform.Platform)
	Get() platform.Platform
}

func newPlatformWrapper() platformStore {
	return &platformWrapper{}
}

type platformWrapper struct {
	mu      sync.Mutex
	current platform.Platform
}

func (p *platformWrapper) Set(plt platform.Platform) {
	p.mu.Lock()
	p.current = plt
	p.mu.Unlock()
}

func (p *platformWrapper) Get() platform.Platform {
	p.mu.Lock()
	plt := p.current
	p.mu.Unlock()
	return plt
}
