// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package config contains the operator's runtime configuration.
package config

import (
	"fmt"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/opampbridge"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

const (
	defaultCollectorConfigMapEntry           = "collector.yaml"
	defaultTargetAllocatorConfigMapEntry     = "targetallocator.yaml"
	defaultOperatorOpAMPBridgeConfigMapEntry = "remoteconfiguration.yaml"
)

// Config holds the static configuration for this operator.
type Config struct {
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
	// OpAmpBridgeAvailability represents the availability of the OpAmpBridge CRD.
	OpAmpBridgeAvailability opampbridge.Availability
	// IgnoreMissingCollectorCRDs is true if the operator can ignore missing OpenTelemetryCollector CRDs.
	IgnoreMissingCollectorCRDs bool
	// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	LabelsFilter []string
	// AnnotationsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	AnnotationsFilter []string
}

// New constructs a new configuration.
func New() Config {
	v := version.Get()
	return Config{
		CollectorImage:                      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:%s", v.OpenTelemetryCollector),
		CollectorConfigMapEntry:             defaultCollectorConfigMapEntry,
		EnableMultiInstrumentation:          true,
		EnableApacheHttpdInstrumentation:    true,
		EnableDotNetInstrumentation:         true,
		EnableGoAutoInstrumentation:         false,
		EnableNginxAutoInstrumentation:      false,
		EnablePythonAutoInstrumentation:     true,
		EnableNodeJSAutoInstrumentation:     true,
		EnableJavaAutoInstrumentation:       true,
		TargetAllocatorImage:                fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/target-allocator:%s", v.TargetAllocator),
		OperatorOpAMPBridgeImage:            fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:%s", v.OperatorOpAMPBridge),
		TargetAllocatorConfigMapEntry:       defaultTargetAllocatorConfigMapEntry,
		OperatorOpAMPBridgeConfigMapEntry:   defaultOperatorOpAMPBridgeConfigMapEntry,
		OpenShiftRoutesAvailability:         openshift.RoutesNotAvailable,
		PrometheusCRAvailability:            prometheus.NotAvailable,
		CertManagerAvailability:             certmanager.NotAvailable,
		TargetAllocatorAvailability:         targetallocator.NotAvailable,
		CollectorAvailability:               collector.NotAvailable,
		IgnoreMissingCollectorCRDs:          false,
		AutoInstrumentationJavaImage:        fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:%s", v.AutoInstrumentationJava),
		AutoInstrumentationNodeJSImage:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:%s", v.AutoInstrumentationNodeJS),
		AutoInstrumentationPythonImage:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:%s", v.AutoInstrumentationPython),
		AutoInstrumentationDotNetImage:      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-dotnet:%s", v.AutoInstrumentationDotNet),
		AutoInstrumentationGoImage:          fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-go-instrumentation/autoinstrumentation-go:%s", v.AutoInstrumentationGo),
		AutoInstrumentationApacheHttpdImage: fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-apache-httpd:%s", v.AutoInstrumentationApacheHttpd),
		AutoInstrumentationNginxImage:       fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-apache-httpd:%s", v.AutoInstrumentationNginx),
		LabelsFilter:                        []string{},
		AnnotationsFilter:                   []string{constants.KubernetesLastAppliedConfigurationAnnotation},
		CreateRBACPermissions:               autoRBAC.NotAvailable,
		OpAmpBridgeAvailability:             opampbridge.NotAvailable,
	}
}
