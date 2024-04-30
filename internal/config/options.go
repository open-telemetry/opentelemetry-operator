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

package config

import (
	"regexp"
	"strings"

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

// Option represents one specific configuration option.
type Option func(c *options)

type options struct {
	autoDetect                          autodetect.AutoDetect
	version                             version.Version
	logger                              logr.Logger
	autoInstrumentationDotNetImage      string
	autoInstrumentationGoImage          string
	autoInstrumentationJavaImage        string
	autoInstrumentationNodeJSImage      string
	autoInstrumentationPythonImage      string
	autoInstrumentationApacheHttpdImage string
	autoInstrumentationNginxImage       string
	collectorImage                      string
	collectorConfigMapEntry             string
	createRBACPermissions               bool
	enableMultiInstrumentation          bool
	enableApacheHttpdInstrumentation    bool
	enableDotNetInstrumentation         bool
	enableGoInstrumentation             bool
	enableNginxInstrumentation          bool
	enablePythonInstrumentation         bool
	enableNodeJSInstrumentation         bool
	enableJavaInstrumentation           bool
	targetAllocatorConfigMapEntry       string
	operatorOpAMPBridgeConfigMapEntry   string
	targetAllocatorImage                string
	operatorOpAMPBridgeImage            string
	openshiftRoutesAvailability         openshift.RoutesAvailability
	prometheusCRAvailability            prometheus.Availability
	labelsFilter                        []string
	annotationsFilter                   []string
}

func WithAutoDetect(a autodetect.AutoDetect) Option {
	return func(o *options) {
		o.autoDetect = a
	}
}
func WithTargetAllocatorImage(s string) Option {
	return func(o *options) {
		o.targetAllocatorImage = s
	}
}
func WithOperatorOpAMPBridgeImage(s string) Option {
	return func(o *options) {
		o.operatorOpAMPBridgeImage = s
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
func WithCreateRBACPermissions(s bool) Option {
	return func(o *options) {
		o.createRBACPermissions = s
	}
}
func WithEnableMultiInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableMultiInstrumentation = s
	}
}
func WithEnableApacheHttpdInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableApacheHttpdInstrumentation = s
	}
}
func WithEnableDotNetInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableDotNetInstrumentation = s
	}
}
func WithEnableGoInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableGoInstrumentation = s
	}
}
func WithEnableNginxInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableNginxInstrumentation = s
	}
}
func WithEnableJavaInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableJavaInstrumentation = s
	}
}
func WithEnablePythonInstrumentation(s bool) Option {
	return func(o *options) {
		o.enablePythonInstrumentation = s
	}
}
func WithEnableNodeJSInstrumentation(s bool) Option {
	return func(o *options) {
		o.enableNodeJSInstrumentation = s
	}
}
func WithTargetAllocatorConfigMapEntry(s string) Option {
	return func(o *options) {
		o.targetAllocatorConfigMapEntry = s
	}
}
func WithOperatorOpAMPBridgeConfigMapEntry(s string) Option {
	return func(o *options) {
		o.operatorOpAMPBridgeConfigMapEntry = s
	}
}
func WithLogger(logger logr.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
func WithVersion(v version.Version) Option {
	return func(o *options) {
		o.version = v
	}
}

func WithAutoInstrumentationJavaImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationJavaImage = s
	}
}

func WithAutoInstrumentationNodeJSImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationNodeJSImage = s
	}
}

func WithAutoInstrumentationPythonImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationPythonImage = s
	}
}

func WithAutoInstrumentationDotNetImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationDotNetImage = s
	}
}

func WithAutoInstrumentationGoImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationGoImage = s
	}
}

func WithAutoInstrumentationApacheHttpdImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationApacheHttpdImage = s
	}
}

func WithAutoInstrumentationNginxImage(s string) Option {
	return func(o *options) {
		o.autoInstrumentationNginxImage = s
	}
}

func WithOpenShiftRoutesAvailability(os openshift.RoutesAvailability) Option {
	return func(o *options) {
		o.openshiftRoutesAvailability = os
	}
}

func WithPrometheusCRAvailability(pcrd prometheus.Availability) Option {
	return func(o *options) {
		o.prometheusCRAvailability = pcrd
	}
}

func WithLabelFilters(labelFilters []string) Option {
	return func(o *options) {

		filters := []string{}
		for _, pattern := range labelFilters {
			var result strings.Builder

			for i, literal := range strings.Split(pattern, "*") {

				// Replace * with .*
				if i > 0 {
					result.WriteString(".*")
				}
				// Quote any regular expression meta characters in the
				// literal text.
				result.WriteString(regexp.QuoteMeta(literal))
			}
			filters = append(filters, result.String())
		}
		o.labelsFilter = filters
	}
}

// WithAnnotationFilters is additive if called multiple times. It works off of a few default filters
// to prevent unnecessary rollouts. The defaults include the following:
// * kubectl.kubernetes.io/last-applied-configuration.
func WithAnnotationFilters(annotationFilters []string) Option {
	return func(o *options) {
		filters := o.annotationsFilter
		for _, pattern := range annotationFilters {
			var result strings.Builder

			for i, literal := range strings.Split(pattern, "*") {

				// Replace * with .*
				if i > 0 {
					result.WriteString(".*")
				}
				// Quote any regular expression meta characters in the
				// literal text.
				result.WriteString(regexp.QuoteMeta(literal))
			}
			filters = append(filters, result.String())
		}
		o.annotationsFilter = filters
	}
}
