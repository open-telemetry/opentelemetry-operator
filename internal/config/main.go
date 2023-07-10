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
)

const (
	defaultAutoDetectFrequency           = 5 * time.Second
	defaultCollectorConfigMapEntry       = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry = "targetallocator.yaml"
)

// Config holds the static configuration for this operator.
type Config struct {
	autoDetect                          autodetect.AutoDetect
	logger                              logr.Logger
	targetAllocatorImage                string
	operatorOpAMPBridgeImage            string
	autoInstrumentationPythonImage      string
	collectorImage                      string
	collectorConfigMapEntry             string
	autoInstrumentationDotNetImage      string
	autoInstrumentationGoImage          string
	autoInstrumentationApacheHttpdImage string
	targetAllocatorConfigMapEntry       string
	autoInstrumentationNodeJSImage      string
	autoInstrumentationJavaImage        string
	onOpenShiftRoutesChange             changeHandler
	labelsFilter                        []string
	openshiftRoutes                     openshiftRoutesStore
	autoDetectFrequency                 time.Duration
	hpaVersion                          hpaVersionStore
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		autoDetectFrequency:           defaultAutoDetectFrequency,
		collectorConfigMapEntry:       defaultCollectorConfigMapEntry,
		targetAllocatorConfigMapEntry: defaultTargetAllocatorConfigMapEntry,
		logger:                        logf.Log.WithName("config"),
		openshiftRoutes:               newOpenShiftRoutesWrapper(),
		hpaVersion:                    newHPAVersionWrapper(),
		version:                       version.Get(),
		onOpenShiftRoutesChange:       newOnChange(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	return Config{
		autoDetect:                          o.autoDetect,
		autoDetectFrequency:                 o.autoDetectFrequency,
		collectorImage:                      o.collectorImage,
		collectorConfigMapEntry:             o.collectorConfigMapEntry,
		targetAllocatorImage:                o.targetAllocatorImage,
		operatorOpAMPBridgeImage:            o.operatorOpAMPBridgeImage,
		targetAllocatorConfigMapEntry:       o.targetAllocatorConfigMapEntry,
		logger:                              o.logger,
		openshiftRoutes:                     o.openshiftRoutes,
		hpaVersion:                          o.hpaVersion,
		onOpenShiftRoutesChange:             o.onOpenShiftRoutesChange,
		autoInstrumentationJavaImage:        o.autoInstrumentationJavaImage,
		autoInstrumentationNodeJSImage:      o.autoInstrumentationNodeJSImage,
		autoInstrumentationPythonImage:      o.autoInstrumentationPythonImage,
		autoInstrumentationDotNetImage:      o.autoInstrumentationDotNetImage,
		autoInstrumentationApacheHttpdImage: o.autoInstrumentationApacheHttpdImage,
		labelsFilter:                        o.labelsFilter,
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

	ora, err := c.autoDetect.OpenShiftRoutesAvailability()
	if err != nil {
		return err
	}

	if c.openshiftRoutes.Get() != ora {
		c.logger.V(1).Info("openshift routes detected", "available", ora)
		c.openshiftRoutes.Set(ora)
		if err = c.onOpenShiftRoutesChange.Do(); err != nil {
			// Don't fail if the callback failed, as auto-detection itself worked.
			c.logger.Error(err, "configuration change notification failed for callback")
		}
	}

	hpaV, err := c.autoDetect.HPAVersion()
	if err != nil {
		return err
	}
	if c.hpaVersion.Get() != hpaV {
		c.logger.V(1).Info("HPA version detected", "version", hpaV)
		c.hpaVersion.Set(hpaV)
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

// OpenShiftRoutes represents the availability of the OpenShift Routes API.
func (c *Config) OpenShiftRoutes() autodetect.OpenShiftRoutesAvailability {
	return c.openshiftRoutes.Get()
}

// AutoscalingVersion represents the preferred version of autoscaling.
func (c *Config) AutoscalingVersion() autodetect.AutoscalingVersion {
	return c.hpaVersion.Get()
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

// AutoInstrumentationGoImage returns OpenTelemetry Go auto-instrumentation container image.
func (c *Config) AutoInstrumentationGoImage() string {
	return c.autoInstrumentationGoImage
}

// AutoInstrumentationApacheHttpdImage returns OpenTelemetry ApacheHttpd auto-instrumentation container image.
func (c *Config) AutoInstrumentationApacheHttpdImage() string {
	return c.autoInstrumentationApacheHttpdImage
}

// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
func (c *Config) LabelsFilter() []string {
	return c.labelsFilter
}

// RegisterOpenShiftRoutesChangeCallback registers the given function as a callback that
// is called when the OpenShift Routes detection detects a change.
func (c *Config) RegisterOpenShiftRoutesChangeCallback(f func() error) {
	c.onOpenShiftRoutesChange.Register(f)
}

type hpaVersionStore interface {
	Set(hpaV autodetect.AutoscalingVersion)
	Get() autodetect.AutoscalingVersion
}

func newHPAVersionWrapper() hpaVersionStore {
	return &hpaVersionWrapper{
		current: autodetect.AutoscalingVersionUnknown,
	}
}

type hpaVersionWrapper struct {
	mu      sync.Mutex
	current autodetect.AutoscalingVersion
}

func (p *hpaVersionWrapper) Set(hpaV autodetect.AutoscalingVersion) {
	p.mu.Lock()
	p.current = hpaV
	p.mu.Unlock()
}

func (p *hpaVersionWrapper) Get() autodetect.AutoscalingVersion {
	p.mu.Lock()
	hpaV := p.current
	p.mu.Unlock()
	return hpaV
}

type openshiftRoutesStore interface {
	Set(ora autodetect.OpenShiftRoutesAvailability)
	Get() autodetect.OpenShiftRoutesAvailability
}

func newOpenShiftRoutesWrapper() openshiftRoutesStore {
	return &openshiftRoutesWrapper{
		current: autodetect.OpenShiftRoutesNotAvailable,
	}
}

type openshiftRoutesWrapper struct {
	mu      sync.Mutex
	current autodetect.OpenShiftRoutesAvailability
}

func (p *openshiftRoutesWrapper) Set(ora autodetect.OpenShiftRoutesAvailability) {
	p.mu.Lock()
	p.current = ora
	p.mu.Unlock()
}

func (p *openshiftRoutesWrapper) Get() autodetect.OpenShiftRoutesAvailability {
	p.mu.Lock()
	ora := p.current
	p.mu.Unlock()
	return ora
}
