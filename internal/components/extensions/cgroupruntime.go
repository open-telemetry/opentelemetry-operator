// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

type cgroupRuntimeConfig struct {
	GoMaxProcs struct {
		Enabled *bool `mapstructure:"enabled"`
	} `mapstructure:"gomaxprocs"`
	GoMemLimit struct {
		Enabled *bool `mapstructure:"enabled"`
	} `mapstructure:"gomemlimit"`
}

func newCgroupRuntimeParserBuilder() components.Builder[cgroupRuntimeConfig] {
	return components.NewBuilder[cgroupRuntimeConfig]().
		WithName("cgroup_runtime").
		WithAlias("cgroupruntime").
		WithSuppressedEnvVarsGen(cgroupRuntimeSuppressedEnvVars)
}

// cgroupRuntimeSuppressedEnvVars returns the Go runtime variables the extension sets itself.
// The extension does nothing when a variable is already present in the environment.
// See open-telemetry/opentelemetry-operator#5651.
func cgroupRuntimeSuppressedEnvVars(_ logr.Logger, config cgroupRuntimeConfig) ([]string, error) {
	var names []string
	// Both settings default to true in the extension.
	if config.GoMemLimit.Enabled == nil || *config.GoMemLimit.Enabled {
		names = append(names, "GOMEMLIMIT")
	}
	if config.GoMaxProcs.Enabled == nil || *config.GoMaxProcs.Enabled {
		names = append(names, "GOMAXPROCS")
	}
	return names, nil
}
