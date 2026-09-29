// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/mitchellh/mapstructure"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

// cgroupRuntimeTypes holds the current type of the cgroup runtime extension and its
// deprecated name, which the collector still accepts.
// See open-telemetry/opentelemetry-collector-contrib#46773.
var cgroupRuntimeTypes = map[string]struct{}{
	"cgroup_runtime": {},
	"cgroupruntime":  {},
}

// cgroupRuntimeConfig holds the fields of the cgroup runtime extension config
// that decide which Go runtime variables the extension sets at startup.
type cgroupRuntimeConfig struct {
	GoMaxProcs struct {
		Enabled *bool `mapstructure:"enabled"`
	} `mapstructure:"gomaxprocs"`
	GoMemLimit struct {
		Enabled *bool `mapstructure:"enabled"`
	} `mapstructure:"gomemlimit"`
}

// GoRuntimeEnvManagedBy reports whether the named extension sets GOMEMLIMIT and GOMAXPROCS itself.
func GoRuntimeEnvManagedBy(name string, config any) (memLimit, maxProcs bool, err error) {
	if _, ok := cgroupRuntimeTypes[components.ComponentType(name)]; !ok {
		return false, false, nil
	}
	var parsed cgroupRuntimeConfig
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return false, false, err
	}
	// Both settings default to true in the extension.
	memLimit = parsed.GoMemLimit.Enabled == nil || *parsed.GoMemLimit.Enabled
	maxProcs = parsed.GoMaxProcs.Enabled == nil || *parsed.GoMaxProcs.Enabled
	return memLimit, maxProcs, nil
}
