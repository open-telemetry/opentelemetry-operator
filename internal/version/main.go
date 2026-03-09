// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package version contains the operator's version, as well as versions of underlying components.
package version

import (
	"fmt"
	"runtime"
)

var (
	version                        string
	buildDate                      string
	otelCol                        string
	targetAllocator                string
	operatorOpAMPBridge            string
	autoInstrumentationJava        string
	autoInstrumentationNodeJS      string
	autoInstrumentationPython      string
	autoInstrumentationDotNet      string
	autoInstrumentationApacheHttpd string
	autoInstrumentationNginx       string
	autoInstrumentationGo          string
)

// Version holds this Operator's version as well as the version of some of the components it uses.
type Version struct {
	Operator                       string `json:"opentelemetry-operator"`
	BuildDate                      string `json:"build-date"`
	OpenTelemetryCollector         string `json:"opentelemetry-collector-version"`
	Go                             string `json:"go-version"`
	TargetAllocator                string `json:"target-allocator-version"`
	OperatorOpAMPBridge            string `json:"operator-opamp-bridge"`
	AutoInstrumentationJava        string `json:"auto-instrumentation-java"`
	AutoInstrumentationNodeJS      string `json:"auto-instrumentation-nodejs"`
	AutoInstrumentationPython      string `json:"auto-instrumentation-python"`
	AutoInstrumentationDotNet      string `json:"auto-instrumentation-dotnet"`
	AutoInstrumentationGo          string `json:"auto-instrumentation-go"`
	AutoInstrumentationApacheHttpd string `json:"auto-instrumentation-apache-httpd"`
	AutoInstrumentationNginx       string `json:"auto-instrumentation-nginx"`
}

// Get returns the Version object with the relevant information.
func Get() Version {
	return Version{
		Operator:                       version,
		BuildDate:                      buildDate,
		OpenTelemetryCollector:         OpenTelemetryCollector(),
		Go:                             runtime.Version(),
		TargetAllocator:                TargetAllocator(),
		OperatorOpAMPBridge:            OperatorOpAMPBridge(),
		AutoInstrumentationJava:        AutoInstrumentationJava(),
		AutoInstrumentationNodeJS:      AutoInstrumentationNodeJS(),
		AutoInstrumentationPython:      AutoInstrumentationPython(),
		AutoInstrumentationDotNet:      AutoInstrumentationDotNet(),
		AutoInstrumentationGo:          AutoInstrumentationGo(),
		AutoInstrumentationApacheHttpd: AutoInstrumentationApacheHttpd(),
		AutoInstrumentationNginx:       AutoInstrumentationNginx(),
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', OpenTelemetryCollector='%v', Go='%v', TargetAllocator='%v', OperatorOpAMPBridge='%v', AutoInstrumentationJava='%v', AutoInstrumentationNodeJS='%v', AutoInstrumentationPython='%v', AutoInstrumentationDotNet='%v', AutoInstrumentationGo='%v', AutoInstrumentationApacheHttpd='%v', AutoInstrumentationNginx='%v')",
		v.Operator,
		v.BuildDate,
		v.OpenTelemetryCollector,
		v.Go,
		v.TargetAllocator,
		v.OperatorOpAMPBridge,
		v.AutoInstrumentationJava,
		v.AutoInstrumentationNodeJS,
		v.AutoInstrumentationPython,
		v.AutoInstrumentationDotNet,
		v.AutoInstrumentationGo,
		v.AutoInstrumentationApacheHttpd,
		v.AutoInstrumentationNginx,
	)
}

// OpenTelemetryCollector returns the default OpenTelemetryCollector to use when no versions are specified via CLI or configuration.
func OpenTelemetryCollector() string {
	if otelCol != "" {
		// this should always be set, as it's specified during the build
		return otelCol
	}

	// fallback value, useful for tests
	return "0.0.0"
}

// TargetAllocator returns the default TargetAllocator to use when no versions are specified via CLI or configuration.
func TargetAllocator() string {
	if targetAllocator != "" {
		// this should always be set, as it's specified during the build
		return targetAllocator
	}

	// fallback value, useful for tests
	return "0.0.0"
}

// OperatorOpAMPBridge returns the default OperatorOpAMPBridge to use when no versions are specified via CLI or configuration.
func OperatorOpAMPBridge() string {
	if operatorOpAMPBridge != "" {
		// this should always be set, as it's specified during the build
		return operatorOpAMPBridge
	}

	// fallback value, useful for tests
	return "0.0.0"
}

func AutoInstrumentationJava() string {
	if autoInstrumentationJava != "" {
		return autoInstrumentationJava
	}
	return "0.0.0"
}

func AutoInstrumentationNodeJS() string {
	if autoInstrumentationNodeJS != "" {
		return autoInstrumentationNodeJS
	}
	return "0.0.0"
}

func AutoInstrumentationPython() string {
	if autoInstrumentationPython != "" {
		return autoInstrumentationPython
	}
	return "0.0.0"
}

func AutoInstrumentationDotNet() string {
	if autoInstrumentationDotNet != "" {
		return autoInstrumentationDotNet
	}
	return "0.0.0"
}

func AutoInstrumentationApacheHttpd() string {
	if autoInstrumentationApacheHttpd != "" {
		return autoInstrumentationApacheHttpd
	}
	return "0.0.0"
}

func AutoInstrumentationNginx() string {
	if autoInstrumentationNginx != "" {
		return autoInstrumentationNginx
	}
	return "0.0.0"
}

func AutoInstrumentationGo() string {
	if autoInstrumentationGo != "" {
		return autoInstrumentationGo
	}
	return "0.0.0"
}
