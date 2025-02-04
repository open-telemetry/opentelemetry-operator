// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
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
	createRBACPermissions               autoRBAC.Availability
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
	certManagerAvailability             certmanager.Availability
	targetAllocatorAvailability         targetallocator.Availability
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

func WithRBACPermissions(rAuto autoRBAC.Availability) Option {
	return func(o *options) {
		o.createRBACPermissions = rAuto
	}
}

func WithCertManagerAvailability(cmAvl certmanager.Availability) Option {
	return func(o *options) {
		o.certManagerAvailability = cmAvl
	}
}

func WithLabelFilters(labelFilters []string) Option {
	return func(o *options) {
		o.labelsFilter = append(o.labelsFilter, labelFilters...)
	}
}

// WithAnnotationFilters is additive if called multiple times. It works off of a few default filters
// to prevent unnecessary rollouts. The defaults include the following:
// * kubectl.kubernetes.io/last-applied-configuration.
func WithAnnotationFilters(annotationFilters []string) Option {
	return func(o *options) {
		o.annotationsFilter = append(o.annotationsFilter, annotationFilters...)
	}
}

func WithEncodeLevelFormat(s string) zapcore.LevelEncoder {
	if s == "lowercase" {
		return zapcore.LowercaseLevelEncoder
	} else {
		return zapcore.CapitalLevelEncoder
	}
}
