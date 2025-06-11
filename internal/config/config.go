// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package config contains the operator's runtime configuration.
package config

import (
	"fmt"

	"github.com/goccy/go-yaml"

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
	TargetAllocatorImage string `yaml:"targetallocator-image"`
	// OperatorOpAMPBridgeImage represents the flag to override the OpAMPBridge container image.
	OperatorOpAMPBridgeImage string `yaml:"operatoropampbridge-image"`
	// AutoInstrumentationPythonImage is the OpenTelemetry Python auto-instrumentation container image.
	AutoInstrumentationPythonImage string `yaml:"auto-instrumentation-python-image"`
	// CollectorImage represents the flag to override the OpenTelemetry Collector container image.
	CollectorImage string `yaml:"collector-image"`
	// CollectorConfigMapEntry represents the configuration file name for the collector. Immutable.
	CollectorConfigMapEntry string `yaml:"collector-configmap-entry"`
	// CreateRBACPermissions is true when the operator can create RBAC permissions for SAs running a collector instance. Immutable.
	CreateRBACPermissions autoRBAC.Availability `yaml:"create-rbac-permissions"`
	// EnableMultiInstrumentation is true when the operator supports multi instrumentation.
	EnableMultiInstrumentation bool `yaml:"enable-multi-instrumentation"`
	// EnableApacheHttpdAutoInstrumentation is true when the operator supports ApacheHttpd auto instrumentation.
	EnableApacheHttpdInstrumentation bool `yaml:"enable-apache-httpd-instrumentation"`
	// EnableDotNetAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnableDotNetAutoInstrumentation bool `yaml:"enable-dot-net-auto-instrumentation"`
	// EnableGoAutoInstrumentation is true when the operator supports Go auto instrumentation.
	EnableGoAutoInstrumentation bool `yaml:"enable-go-auto-instrumentation"`
	// EnableNginxAutoInstrumentation is true when the operator supports nginx auto instrumentation.
	EnableNginxAutoInstrumentation bool `yaml:"enable-nginx-auto-instrumentation"`
	// EnablePythonAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnablePythonAutoInstrumentation bool `yaml:"enable-python-auto-instrumentation"`
	// EnableNodeJSAutoInstrumentation is true when the operator supports dotnet auto instrumentation.
	EnableNodeJSAutoInstrumentation bool `yaml:"enable-node-js-auto-instrumentation"`
	// EnableJavaAutoInstrumentation is true when the operator supports java auto instrumentation.
	EnableJavaAutoInstrumentation bool `yaml:"enable-java-auto-instrumentation"`
	// AutoInstrumentationDotNetImage is the OpenTelemetry DotNet auto-instrumentation container image.
	AutoInstrumentationDotNetImage string `yaml:"auto-instrumentation-dot-net-image"`
	// AutoInstrumentationGoImage is the OpenTelemetry Go auto-instrumentation container image.
	AutoInstrumentationGoImage string `yaml:"auto-instrumentation-go-image"`
	// AutoInstrumentationApacheHttpdImage is the OpenTelemetry ApacheHttpd auto-instrumentation container image.
	AutoInstrumentationApacheHttpdImage string `yaml:"auto-instrumentation-apache-httpd-image"`
	// AutoInstrumentationNginxImage is the OpenTelemetry Nginx auto-instrumentation container image.
	AutoInstrumentationNginxImage string `yaml:"auto-instrumentation-nginx-image"`
	// TargetAllocatorConfigMapEntry represents the configuration file name for the TargetAllocator. Immutable.
	TargetAllocatorConfigMapEntry string `yaml:"target-allocator-configmap-entry"`
	// OperatorOpAMPBridgeImageConfigMapEntry represents the configuration file name for the OpAMPBridge. Immutable.
	OperatorOpAMPBridgeConfigMapEntry string `yaml:"operator-op-amp-bridge-configmap-entry"`
	// AutoInstrumentationNodeJSImage is the OpenTelemetry NodeJS auto-instrumentation container image.
	AutoInstrumentationNodeJSImage string `yaml:"auto-instrumentation-node-js-image"`
	// AutoInstrumentationJavaImage returns OpenTelemetry Java auto-instrumentation container image.
	AutoInstrumentationJavaImage string `yaml:"auto-instrumentation-java-image"`

	// OpenShiftRoutesAvailability represents the availability of the OpenShift Routes API.
	OpenShiftRoutesAvailability openshift.RoutesAvailability `yaml:"open-shift-routes-availability"`
	// PrometheusCRAvailability represents the availability of the Prometheus Operator CRDs.
	PrometheusCRAvailability prometheus.Availability `yaml:"prometheus-cr-availability"`
	// CertManagerAvailability represents the availability of the Cert-Manager.
	CertManagerAvailability certmanager.Availability `yaml:"cert-manager-availability"`
	// TargetAllocatorAvailability represents the availability of the TargetAllocator CRD.
	TargetAllocatorAvailability targetallocator.Availability `yaml:"target-allocator-availability"`
	// CollectorAvailability represents the availability of the OpenTelemetryCollector CRD.
	CollectorAvailability collector.Availability `yaml:"collector-availability"`
	// OpAmpBridgeAvailability represents the availability of the OpAmpBridge CRD.
	OpAmpBridgeAvailability opampbridge.Availability `yaml:"opampbridge-availability"`
	// IgnoreMissingCollectorCRDs is true if the operator can ignore missing OpenTelemetryCollector CRDs.
	IgnoreMissingCollectorCRDs bool `yaml:"ignore-missing-collector-crds"`
	// LabelsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	LabelsFilter []string `yaml:"labels-filter"`
	// AnnotationsFilter Returns the filters converted to regex strings used to filter out unwanted labels from propagations.
	AnnotationsFilter []string `yaml:"annotations-filter"`
}

// New constructs a new configuration.
func New() Config {
	v := version.Get()
	return Config{
		CollectorImage:                      fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:%s", v.OpenTelemetryCollector),
		CollectorConfigMapEntry:             defaultCollectorConfigMapEntry,
		EnableMultiInstrumentation:          true,
		EnableApacheHttpdInstrumentation:    true,
		EnableDotNetAutoInstrumentation:     true,
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

func (c Config) String() string {
	b, _ := yaml.Marshal(c)
	return string(b)
}

func (c Config) ToStringMap() map[string]string {
	b, _ := yaml.Marshal(c)
	var m map[string]string
	_ = yaml.Unmarshal(b, &m)
	return m
}
