// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package config contains the operator's runtime configuration.
package config

import (
	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

const (
	defaultCollectorConfigMapEntry           = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry     = "targetallocator.yaml"
	defaultOperatorOpAMPBridgeConfigMapEntry = "remoteconfiguration.yaml"
)

// Config holds the static configuration for this operator.
type Config struct {
	logger logr.Logger
	// TargetAllocatorImage represents the flag to override the OpenTelemetry TargetAllocator container image.
	TargetAllocatorImage string
	// OperatorOpAMPBridgeImage represents the flag to override the OpAMPBridge container image.
	OperatorOpAMPBridgeImage string
	// AutoInstrumentationPythonImage is the OpenTelemetry Python auto-instrumentation container image.
	AutoInstrumentationPythonImage string
	// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
	CollectorImage string
	// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
	CollectorConfigMapEntry string
	// CreateRBACPermissions is true when the operator can create RBAC permissions for SAs running a collector instance. Immutable.
	CreateRBACPermissions autoRBAC.Availability
	// EnableMultiInstrumentation is true when the operator supports multi instrumentation.
	EnableMultiInstrumentation bool
	// EnableApacheHttpdAutoInstrumentation is true when the operator supports ApacheHttpd auto instrumentation.
	EnableApacheHttpdInstrumentation bool
	// EnableDotNetAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnableDotNetInstrumentation bool
	// EnableGoAutoInstrumentation is true when the operator supports Go auto instrumentation.
	EnableGoAutoInstrumentation bool
	// EnableNginxAutoInstrumentation is true when the operator supports nginx auto instrumentation.
	EnableNginxAutoInstrumentation bool
	// EnablePythonAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnablePythonAutoInstrumentation bool
	// EnableNodeJSAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnableNodeJSAutoInstrumentation bool
	// EnableJavaAutoInstrumentation is true when the operator supports java auto instrumentation.
	EnableJavaAutoInstrumentation bool
	// AutoInstrumentationDotNetImage is the OpenTelemetry DotNet auto-instrumentation container image.
	AutoInstrumentationDotNetImage string
	// AutoInstrumentationGoImage is the OpenTelemetry Go auto-instrumentation container image.
	AutoInstrumentationGoImage string
	// AutoInstrumentationApacheHttpdImage is the OpenTelemetry ApacheHttpd auto-instrumentation container image.
	AutoInstrumentationApacheHttpdImage string
	// AutoInstrumentationNginxImage is the OpenTelemetry Nginx auto-instrumentation container image.
	AutoInstrumentationNginxImage string
	// TargetAllocatorConfigMapEntry represents the configuration file name for the TargetAllocator. Immutable.
	TargetAllocatorConfigMapEntry string
	// OperatorOpAMPBridgeImageConfigMapEntry represents the configuration file name for the OpAMPBridge. Immutable.
	OperatorOpAMPBridgeConfigMapEntry string
	// AutoInstrumentationNodeJSImage is the OpenTelemetry NodeJS auto-instrumentation container image.
	AutoInstrumentationNodeJSImage string
	// AutoInstrumentationJavaImage returns OpenTelemetry Java auto-instrumentation container image.
	AutoInstrumentationJavaImage string

	// OpenShiftRoutesAvailability represents the availability of the OpenShift Routes API.
	OpenShiftRoutesAvailability openshift.RoutesAvailability
	// PrometheusCRAvailability represents the availability of the Prometheus Operator CRDs.
	PrometheusCRAvailability prometheus.Availability
	// CertManagerAvailability represents the availability of the Cert-Manager.
	CertManagerAvailability certmanager.Availability
	// TargetAllocatorAvailability represents the availability of the TargetAllocator CRD.
	TargetAllocatorAvailability targetallocator.Availability
	// CollectorAvailability represents the availability of the OpenTelemetryCollector CRD.
	CollectorAvailability collector.Availability
	// IgnoreMissingCollectorCRDs is true if the operator can ignore missing OpenTelemetryCollector CRDs.
	IgnoreMissingCollectorCRDs bool
	// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	LabelsFilter []string
	// AnnotationsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	AnnotationsFilter []string
}

// New constructs a new configuration based on the given options.
func New(opts ...Option) Config {
	// initialize with the default values
	o := options{
		prometheusCRAvailability:          prometheus.NotAvailable,
		openshiftRoutesAvailability:       openshift.RoutesNotAvailable,
		createRBACPermissions:             autoRBAC.NotAvailable,
		certManagerAvailability:           certmanager.NotAvailable,
		targetAllocatorAvailability:       targetallocator.NotAvailable,
		collectorAvailability:             collector.NotAvailable,
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
		CollectorImage:                      o.collectorImage,
		CollectorConfigMapEntry:             o.collectorConfigMapEntry,
		EnableMultiInstrumentation:          o.enableMultiInstrumentation,
		EnableApacheHttpdInstrumentation:    o.enableApacheHttpdInstrumentation,
		EnableDotNetInstrumentation:         o.enableDotNetInstrumentation,
		EnableGoAutoInstrumentation:         o.enableGoInstrumentation,
		EnableNginxAutoInstrumentation:      o.enableNginxInstrumentation,
		EnablePythonAutoInstrumentation:     o.enablePythonInstrumentation,
		EnableNodeJSAutoInstrumentation:     o.enableNodeJSInstrumentation,
		EnableJavaAutoInstrumentation:       o.enableJavaInstrumentation,
		TargetAllocatorImage:                o.targetAllocatorImage,
		OperatorOpAMPBridgeImage:            o.operatorOpAMPBridgeImage,
		TargetAllocatorConfigMapEntry:       o.targetAllocatorConfigMapEntry,
		OperatorOpAMPBridgeConfigMapEntry:   o.operatorOpAMPBridgeConfigMapEntry,
		logger:                              o.logger,
		OpenShiftRoutesAvailability:         o.openshiftRoutesAvailability,
		PrometheusCRAvailability:            o.prometheusCRAvailability,
		CertManagerAvailability:             o.certManagerAvailability,
		TargetAllocatorAvailability:         o.targetAllocatorAvailability,
		CollectorAvailability:               o.collectorAvailability,
		IgnoreMissingCollectorCRDs:          o.ignoreMissingCollectorCRDs,
		AutoInstrumentationJavaImage:        o.autoInstrumentationJavaImage,
		AutoInstrumentationNodeJSImage:      o.autoInstrumentationNodeJSImage,
		AutoInstrumentationPythonImage:      o.autoInstrumentationPythonImage,
		AutoInstrumentationDotNetImage:      o.autoInstrumentationDotNetImage,
		AutoInstrumentationGoImage:          o.autoInstrumentationGoImage,
		AutoInstrumentationApacheHttpdImage: o.autoInstrumentationApacheHttpdImage,
		AutoInstrumentationNginxImage:       o.autoInstrumentationNginxImage,
		LabelsFilter:                        o.labelsFilter,
		AnnotationsFilter:                   o.annotationsFilter,
		CreateRBACPermissions:               o.createRBACPermissions,
	}
}
