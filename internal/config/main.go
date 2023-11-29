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
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

const (
	defaultAutoDetectFrequency               = 5 * time.Second
	defaultCollectorConfigMapEntry           = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry     = "targetallocator.yaml"
	defaultOperatorOpAMPBridgeConfigMapEntry = "remoteconfiguration.yaml"
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
	autoInstrumentationNginxImage       string
	targetAllocatorConfigMapEntry       string
	operatorOpAMPBridgeConfigMapEntry   string
	autoInstrumentationNodeJSImage      string
	autoInstrumentationJavaImage        string
	openshiftRoutesAvailability         openshift.RoutesAvailability
	labelsFilter                        []string
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		openshiftRoutesAvailability:       openshift.RoutesNotAvailable,
		collectorConfigMapEntry:           defaultCollectorConfigMapEntry,
		targetAllocatorConfigMapEntry:     defaultTargetAllocatorConfigMapEntry,
		operatorOpAMPBridgeConfigMapEntry: defaultOperatorOpAMPBridgeConfigMapEntry,
		logger:                            logf.Log.WithName("config"),
		version:                           version.Get(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	return Config{
		autoDetect:                          o.autoDetect,
		collectorImage:                      o.collectorImage,
		collectorConfigMapEntry:             o.collectorConfigMapEntry,
		targetAllocatorImage:                o.targetAllocatorImage,
		operatorOpAMPBridgeImage:            o.operatorOpAMPBridgeImage,
		targetAllocatorConfigMapEntry:       o.targetAllocatorConfigMapEntry,
		operatorOpAMPBridgeConfigMapEntry:   o.operatorOpAMPBridgeConfigMapEntry,
		logger:                              o.logger,
		openshiftRoutesAvailability:         o.openshiftRoutesAvailability,
		autoInstrumentationJavaImage:        o.autoInstrumentationJavaImage,
		autoInstrumentationNodeJSImage:      o.autoInstrumentationNodeJSImage,
		autoInstrumentationPythonImage:      o.autoInstrumentationPythonImage,
		autoInstrumentationDotNetImage:      o.autoInstrumentationDotNetImage,
		autoInstrumentationGoImage:          o.autoInstrumentationGoImage,
		autoInstrumentationApacheHttpdImage: o.autoInstrumentationApacheHttpdImage,
		autoInstrumentationNginxImage:       o.autoInstrumentationNginxImage,
		labelsFilter:                        o.labelsFilter,
	}
}

// AutoDetect attempts to automatically detect relevant information for this operator.
func (c *Config) AutoDetect() error {
	c.logger.V(2).Info("auto-detecting the configuration based on the environment")

	ora, err := c.autoDetect.OpenShiftRoutesAvailability()
	if err != nil {
		return err
	}
	c.openshiftRoutesAvailability = ora
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

// OperatorOpAMPBridgeImage represents the flag to override the OpAMPBridge container image.
func (c *Config) OperatorOpAMPBridgeImage() string {
	return c.operatorOpAMPBridgeImage
}

// TargetAllocatorConfigMapEntry represents the configuration file name for the TargetAllocator. Immutable.
func (c *Config) TargetAllocatorConfigMapEntry() string {
	return c.targetAllocatorConfigMapEntry
}

// OperatorOpAMPBridgeImageConfigMapEntry represents the configuration file name for the OpAMPBridge. Immutable.
func (c *Config) OperatorOpAMPBridgeConfigMapEntry() string {
	return c.operatorOpAMPBridgeConfigMapEntry
}

// OpenShiftRoutesAvailability represents the availability of the OpenShift Routes API.
func (c *Config) OpenShiftRoutesAvailability() openshift.RoutesAvailability {
	return c.openshiftRoutesAvailability
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

// AutoInstrumentationNginxImage returns OpenTelemetry Nginx auto-instrumentation container image.
func (c *Config) AutoInstrumentationNginxImage() string {
	return c.autoInstrumentationNginxImage
}

// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
func (c *Config) LabelsFilter() []string {
	return c.labelsFilter
}
