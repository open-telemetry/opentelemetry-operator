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
	"context"
	"time"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
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
	createRBACPermissions               autoRBAC.Availability
	enableMultiInstrumentation          bool
	enableApacheHttpdInstrumentation    bool
	enableDotNetInstrumentation         bool
	enableGoInstrumentation             bool
	enableNginxInstrumentation          bool
	enablePythonInstrumentation         bool
	enableNodeJSInstrumentation         bool
	enableJavaInstrumentation           bool
	autoInstrumentationDotNetImage      string
	autoInstrumentationGoImage          string
	autoInstrumentationApacheHttpdImage string
	autoInstrumentationNginxImage       string
	targetAllocatorConfigMapEntry       string
	operatorOpAMPBridgeConfigMapEntry   string
	autoInstrumentationNodeJSImage      string
	autoInstrumentationJavaImage        string

	openshiftRoutesAvailability openshift.RoutesAvailability
	prometheusCRAvailability    prometheus.Availability
	labelsFilter                []string
	annotationsFilter           []string
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		prometheusCRAvailability:          prometheus.NotAvailable,
		openshiftRoutesAvailability:       openshift.RoutesNotAvailable,
		createRBACPermissions:             autoRBAC.NotAvailable,
		collectorConfigMapEntry:           defaultCollectorConfigMapEntry,
		targetAllocatorConfigMapEntry:     defaultTargetAllocatorConfigMapEntry,
		operatorOpAMPBridgeConfigMapEntry: defaultOperatorOpAMPBridgeConfigMapEntry,
		logger:                            logf.Log.WithName("config"),
		version:                           version.Get(),
		enableJavaInstrumentation:         true,
		annotationsFilter:                 []string{"kubectl.kubernetes.io/last-applied-configuration"},
	}

	for _, opt := range opts {
		opt(&o)
	}

	return Config{
		autoDetect:                          o.autoDetect,
		collectorImage:                      o.collectorImage,
		collectorConfigMapEntry:             o.collectorConfigMapEntry,
		enableMultiInstrumentation:          o.enableMultiInstrumentation,
		enableApacheHttpdInstrumentation:    o.enableApacheHttpdInstrumentation,
		enableDotNetInstrumentation:         o.enableDotNetInstrumentation,
		enableGoInstrumentation:             o.enableGoInstrumentation,
		enableNginxInstrumentation:          o.enableNginxInstrumentation,
		enablePythonInstrumentation:         o.enablePythonInstrumentation,
		enableNodeJSInstrumentation:         o.enableNodeJSInstrumentation,
		enableJavaInstrumentation:           o.enableJavaInstrumentation,
		targetAllocatorImage:                o.targetAllocatorImage,
		operatorOpAMPBridgeImage:            o.operatorOpAMPBridgeImage,
		targetAllocatorConfigMapEntry:       o.targetAllocatorConfigMapEntry,
		operatorOpAMPBridgeConfigMapEntry:   o.operatorOpAMPBridgeConfigMapEntry,
		logger:                              o.logger,
		openshiftRoutesAvailability:         o.openshiftRoutesAvailability,
		prometheusCRAvailability:            o.prometheusCRAvailability,
		autoInstrumentationJavaImage:        o.autoInstrumentationJavaImage,
		autoInstrumentationNodeJSImage:      o.autoInstrumentationNodeJSImage,
		autoInstrumentationPythonImage:      o.autoInstrumentationPythonImage,
		autoInstrumentationDotNetImage:      o.autoInstrumentationDotNetImage,
		autoInstrumentationGoImage:          o.autoInstrumentationGoImage,
		autoInstrumentationApacheHttpdImage: o.autoInstrumentationApacheHttpdImage,
		autoInstrumentationNginxImage:       o.autoInstrumentationNginxImage,
		labelsFilter:                        o.labelsFilter,
		annotationsFilter:                   o.annotationsFilter,
		createRBACPermissions:               o.createRBACPermissions,
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
	c.logger.V(2).Info("openshift routes detected", "availability", ora)

	pcrd, err := c.autoDetect.PrometheusCRsAvailability()
	if err != nil {
		return err
	}
	c.prometheusCRAvailability = pcrd
	c.logger.V(2).Info("prometheus cr detected", "availability", pcrd)

	rAuto, err := c.autoDetect.RBACPermissions(context.Background())
	if err != nil {
		c.logger.V(2).Info("the rbac permissions are not set for the operator", "reason", err)
	}
	c.createRBACPermissions = rAuto
	c.logger.V(2).Info("create rbac permissions detected", "availability", rAuto)

	return nil
}

// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
func (c *Config) CollectorImage() string {
	return c.collectorImage
}

// EnableMultiInstrumentation is true when the operator supports multi instrumentation.
func (c *Config) EnableMultiInstrumentation() bool {
	return c.enableMultiInstrumentation
}

// EnableApacheHttpdAutoInstrumentation is true when the operator supports ApacheHttpd auto instrumentation.
func (c *Config) EnableApacheHttpdAutoInstrumentation() bool {
	return c.enableApacheHttpdInstrumentation
}

// EnableDotNetAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
func (c *Config) EnableDotNetAutoInstrumentation() bool {
	return c.enableDotNetInstrumentation
}

// EnableGoAutoInstrumentation is true when the operator supports Go auto instrumentation.
func (c *Config) EnableGoAutoInstrumentation() bool {
	return c.enableGoInstrumentation
}

// EnableNginxAutoInstrumentation is true when the operator supports nginx auto instrumentation.
func (c *Config) EnableNginxAutoInstrumentation() bool {
	return c.enableNginxInstrumentation
}

// EnableJavaAutoInstrumentation is true when the operator supports nginx auto instrumentation.
func (c *Config) EnableJavaAutoInstrumentation() bool {
	return c.enableJavaInstrumentation
}

// EnablePythonAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
func (c *Config) EnablePythonAutoInstrumentation() bool {
	return c.enablePythonInstrumentation
}

// EnableNodeJSAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
func (c *Config) EnableNodeJSAutoInstrumentation() bool {
	return c.enableNodeJSInstrumentation
}

// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
func (c *Config) CollectorConfigMapEntry() string {
	return c.collectorConfigMapEntry
}

// CreateRBACPermissions is true when the operator can create RBAC permissions for SAs running a collector instance. Immutable.
func (c *Config) CreateRBACPermissions() autoRBAC.Availability {
	return c.createRBACPermissions
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

// PrometheusCRAvailability represents the availability of the Prometheus Operator CRDs.
func (c *Config) PrometheusCRAvailability() prometheus.Availability {
	return c.prometheusCRAvailability
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

// AnnotationsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
func (c *Config) AnnotationsFilter() []string {
	return c.annotationsFilter
}
