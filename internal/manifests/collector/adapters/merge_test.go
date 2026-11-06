// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adapters_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func TestMergeDefaults(t *testing.T) {
	for _, tt := range []struct {
		name     string
		base     *v1beta1.Config
		override *v1beta1.Config
		expected *v1beta1.Config
	}{
		{
			name: "override wins on a shared receiver setting",
			base: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{
					"otlp": map[string]any{"protocols": map[string]any{"grpc": map[string]any{"endpoint": "0.0.0.0:4317"}}},
				}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service:   v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
			override: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{
					"otlp": map[string]any{"protocols": map[string]any{"grpc": map[string]any{"endpoint": "0.0.0.0:14317"}}},
				}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service:   v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
			expected: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{
					"otlp": map[string]any{"protocols": map[string]any{"grpc": map[string]any{"endpoint": "0.0.0.0:14317"}}},
				}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service:   v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
		},
		{
			name: "default adds a processor the override lacks",
			base: &v1beta1.Config{
				Receivers:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters:  v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{"memory_limiter": map[string]any{"check_interval": "1s"}}},
				Service:    v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
			override: &v1beta1.Config{
				Receivers:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters:  v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{"batch": map[string]any{}}},
				Service:    v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
			expected: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{
					"memory_limiter": map[string]any{"check_interval": "1s"},
					"batch":          map[string]any{},
				}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{}},
			},
		},
		{
			name: "override pipeline lists replace the default's",
			base: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					"traces": {Receivers: []string{"otlp"}, Processors: []string{"batch"}, Exporters: []string{"debug"}},
				}},
			},
			override: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					"traces": {Receivers: []string{"otlp"}, Exporters: []string{"otlp"}},
				}},
			},
			expected: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}, "otlp": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					// The override keeps batch: its empty processor list cannot clear the default's.
					"traces": {Receivers: []string{"otlp"}, Processors: []string{"batch"}, Exporters: []string{"otlp"}},
				}},
			},
		},
		{
			name: "default pipeline is added when the override omits it",
			base: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					"metrics": {Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
				}},
			},
			override: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					"traces": {Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
				}},
			},
			expected: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
				Service: v1beta1.Service{Pipelines: map[string]*v1beta1.Pipeline{
					"metrics": {Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
					"traces":  {Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
				}},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := adapters.MergeDefaults(tt.base, tt.override)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, merged)
		})
	}
}
