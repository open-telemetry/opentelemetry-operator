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

func TestInvalidYAML(t *testing.T) {
	// test
	config, err := adapters.ConfigFromString("🦄")

	// verify
	assert.Nil(t, config)
	assert.Equal(t, adapters.ErrInvalidYAML, err)
}

func TestEmptyString(t *testing.T) {
	// test and verify
	res, err := adapters.ConfigFromString("")
	assert.NoError(t, err)
	assert.Empty(t, res, 0)
}

func TestConfigFromStructRoundTrip(t *testing.T) {
	cfg := &v1beta1.Config{
		Receivers: v1beta1.AnyConfig{Object: map[string]any{
			"prometheus": map[string]any{
				"config": map[string]any{
					"scrape_configs": []any{map[string]any{"job_name": "0e12", "port": float64(9090), "honor": true, "labels": nil}},
				},
			},
		}},
		Exporters:  v1beta1.AnyConfig{Object: map[string]any{"debug": map[string]any{}}},
		Processors: &v1beta1.AnyConfig{Object: map[string]any{"batch": nil}},
		Service: v1beta1.Service{
			Extensions: []string{"health_check"},
			Telemetry:  &v1beta1.AnyConfig{Object: map[string]any{"metrics": map[string]any{"level": "detailed"}}},
			Pipelines: map[string]*v1beta1.Pipeline{
				"metrics": {Receivers: []string{"prometheus"}, Processors: []string{"batch"}, Exporters: []string{"debug"}},
			},
		},
	}

	config, err := adapters.ConfigFromStruct(cfg)
	require.NoError(t, err)
	assert.Equal(t, map[any]any{
		"receivers": map[any]any{
			"prometheus": map[any]any{
				"config": map[any]any{
					"scrape_configs": []any{map[any]any{"job_name": "0e12", "port": float64(9090), "honor": true, "labels": nil}},
				},
			},
		},
		"exporters":  map[any]any{"debug": map[any]any{}},
		"processors": map[any]any{"batch": nil},
		"service": map[any]any{
			"extensions": []any{"health_check"},
			"telemetry":  map[any]any{"metrics": map[any]any{"level": "detailed"}},
			"pipelines": map[any]any{
				"metrics": map[any]any{"receivers": []any{"prometheus"}, "processors": []any{"batch"}, "exporters": []any{"debug"}},
			},
		},
	}, config)

	// The map is independent of the config.
	config["receivers"].(map[any]any)["prometheus"].(map[any]any)["target_allocator"] = map[any]any{"endpoint": "http://ta:80"}
	assert.NotContains(t, cfg.Receivers.Object["prometheus"], "target_allocator")

	roundTripped, err := adapters.ConfigToStruct(config)
	require.NoError(t, err)
	expected := cfg.DeepCopy()
	expected.Receivers.Object["prometheus"].(map[string]any)["target_allocator"] = map[string]any{"endpoint": "http://ta:80"}
	assert.Equal(t, expected, roundTripped)
}

func TestConfigToStructRejectsNonStringKeys(t *testing.T) {
	_, err := adapters.ConfigToStruct(map[any]any{"receivers": map[any]any{1: nil}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "map key 1 is a int, not a string")
}
