// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCgroupRuntimeSuppressedEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		config    any
		want      []string
		wantErr   bool
	}{
		{
			name:      "nil config",
			extension: "cgroup_runtime",
			config:    nil,
			want:      []string{"GOMEMLIMIT", "GOMAXPROCS"},
		},
		{
			name:      "deprecated type",
			extension: "cgroupruntime",
			config:    map[string]any{},
			want:      []string{"GOMEMLIMIT", "GOMAXPROCS"},
		},
		{
			name:      "named instance with gomemlimit disabled",
			extension: "cgroup_runtime/custom",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": false, "ratio": 0.5},
			},
			want: []string{"GOMAXPROCS"},
		},
		{
			name:      "gomaxprocs disabled",
			extension: "cgroup_runtime",
			config: map[string]any{
				"gomaxprocs": map[string]any{"enabled": false},
			},
			want: []string{"GOMEMLIMIT"},
		},
		{
			name:      "both disabled",
			extension: "cgroup_runtime",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": false},
				"gomaxprocs": map[string]any{"enabled": false},
			},
			want: nil,
		},
		{
			name:      "invalid config",
			extension: "cgroup_runtime",
			config: map[string]any{
				"gomemlimit": map[string]any{"enabled": "yes"},
			},
			wantErr: true,
		},
		{
			name:      "other extension",
			extension: "health_check",
			config:    map[string]any{},
			want:      nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParserFor(tt.extension).GetSuppressedEnvVars(logr.Discard(), tt.config)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
