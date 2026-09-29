// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoRuntimeEnvManagedBy(t *testing.T) {
	tests := []struct {
		name         string
		extension    string
		config       any
		wantMemLimit bool
		wantMaxProcs bool
		wantErr      bool
	}{
		{
			name:      "other extension",
			extension: "health_check",
			config:    map[string]any{},
		},
		{
			name:         "current type with nil config",
			extension:    "cgroup_runtime",
			config:       nil,
			wantMemLimit: true,
			wantMaxProcs: true,
		},
		{
			name:         "deprecated type with nil config",
			extension:    "cgroupruntime",
			config:       nil,
			wantMemLimit: true,
			wantMaxProcs: true,
		},
		{
			name:         "empty config",
			extension:    "cgroup_runtime",
			config:       map[string]any{},
			wantMemLimit: true,
			wantMaxProcs: true,
		},
		{
			name:      "named instance with gomemlimit disabled",
			extension: "cgroup_runtime/custom",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": false, "ratio": 0.5},
			},
			wantMemLimit: false,
			wantMaxProcs: true,
		},
		{
			name:      "gomaxprocs disabled",
			extension: "cgroupruntime",
			config: map[string]any{
				"gomaxprocs": map[string]any{"enabled": false},
			},
			wantMemLimit: true,
			wantMaxProcs: false,
		},
		{
			name:      "both disabled",
			extension: "cgroupruntime",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": false},
				"gomaxprocs": map[string]any{"enabled": false},
			},
		},
		{
			name:      "invalid config",
			extension: "cgroupruntime",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": "yes"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memLimit, maxProcs, err := GoRuntimeEnvManagedBy(tt.extension, tt.config)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMemLimit, memLimit)
			assert.Equal(t, tt.wantMaxProcs, maxProcs)
		})
	}
}
