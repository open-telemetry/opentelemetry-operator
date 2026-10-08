// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
)

func TestPrometheusParser(t *testing.T) {
	param, err := newParams("test/test-img", "testdata/http_sd_config_test.yaml", nil)
	assert.NoError(t, err)

	t.Run("should update config with targetAllocator block if block not present", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(param.OtelCol, param.TargetAllocator)
		assert.NoError(t, err)

		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		prometheusConfig := promCfgMap["config"].(map[any]any)
		assert.NotContains(t, prometheusConfig, "scrape_configs")

		expectedTAConfig := map[any]any{
			"endpoint":     "http://test-targetallocator.default.svc:80",
			"interval":     "30s",
			"collector_id": "${POD_NAME}",
		}
		assert.Equal(t, expectedTAConfig, promCfgMap["target_allocator"])
	})

	t.Run("should update config with targetAllocator block if block already present", func(t *testing.T) {
		paramTa, err := newParams("test/test-img", "testdata/http_sd_config_ta_test.yaml", nil)
		require.NoError(t, err)

		actualConfig, err := ReplaceConfig(paramTa.OtelCol, param.TargetAllocator)
		assert.NoError(t, err)

		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		prometheusConfig := promCfgMap["config"].(map[any]any)
		assert.NotContains(t, prometheusConfig, "scrape_configs")

		expectedTAConfig := map[any]any{
			"endpoint":     "http://test-targetallocator.default.svc:80",
			"interval":     "30s",
			"collector_id": "${POD_NAME}",
		}
		assert.Equal(t, expectedTAConfig, promCfgMap["target_allocator"])
	})

	t.Run("should not update config with http_sd_config", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(param.OtelCol, nil)
		assert.NoError(t, err)

		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		promConfig := promCfgMap["config"].(map[any]any)
		scrapeConfigs := promConfig["scrape_configs"].([]any)
		assert.Len(t, scrapeConfigs, 2)

		expectedJobs := map[string]bool{
			"prometheus": false,
			"service-x":  false,
		}
		for _, sc := range scrapeConfigs {
			scMap := sc.(map[any]any)
			jobName := scMap["job_name"].(string)
			expectedJobs[jobName] = true
			assert.Contains(t, scMap, "file_sd_configs", "job %s should have file_sd_configs", jobName)
			assert.Contains(t, scMap, "static_configs", "job %s should have static_configs", jobName)
		}
		for k, found := range expectedJobs {
			assert.True(t, found, "expected job %s not found", k)
		}
		assert.NotContains(t, promCfgMap, "target_allocator")
	})
}

func TestReplaceConfig(t *testing.T) {
	param, err := newParams("test/test-img", "testdata/relabel_config_original.yaml", nil)
	assert.NoError(t, err)

	t.Run("should not modify config when TargetAllocator is disabled", func(t *testing.T) {
		expectedConfigBytes, err := os.ReadFile("testdata/relabel_config_original.yaml")
		assert.NoError(t, err)
		expectedConfig := string(expectedConfigBytes)

		actualConfig, err := ReplaceConfig(param.OtelCol, nil)
		assert.NoError(t, err)

		assert.YAMLEq(t, expectedConfig, actualConfig)
	})

	t.Run("should remove scrape configs if TargetAllocator is enabled", func(t *testing.T) {
		expectedConfigBytes, err := os.ReadFile("testdata/config_expected_targetallocator.yaml")
		assert.NoError(t, err)
		expectedConfig := string(expectedConfigBytes)

		actualConfig, err := ReplaceConfig(param.OtelCol, param.TargetAllocator)
		assert.NoError(t, err)

		assert.YAMLEq(t, expectedConfig, actualConfig)
	})

	t.Run("should update collectorTargetReloadInterval if specified", func(t *testing.T) {
		customInterval := &metav1.Duration{Duration: 10 * time.Second}

		param.OtelCol.Spec.TargetAllocator.CollectorTargetReloadInterval = customInterval

		actualConfig, err := ReplaceConfig(param.OtelCol, param.TargetAllocator, ta.WithCollectorTargetReloadInterval(customInterval.Duration.String()))
		assert.NoError(t, err)

		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		assert.Equal(t, customInterval.Duration.String(), promCfgMap["target_allocator"].(map[any]any)["interval"])
	})
}

// TestReplaceConfigPreservesScalarTypes checks that the target allocator rewrite leaves every value with the
// type it has in the CR, both inside the rewritten Prometheus receiver and elsewhere. The values are read back
// the way the collector reads them.
func TestReplaceConfigPreservesScalarTypes(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{
					"prometheus": map[string]any{
						"config": map[string]any{
							"global": map[string]any{"external_labels": map[string]any{"cluster": "0e12", "on": "yes", "1e5": "true"}},
							"scrape_configs": []any{map[string]any{
								"job_name":       "1e10",
								"scrape_timeout": "10s",
								"static_configs": []any{map[string]any{"targets": []any{"0.0.0.0:9090"}}},
							}},
						},
					},
				}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{
					"metricstransform": map[string]any{"new_value": "0e12", "limit": float64(1000000), "ratio": float64(0.5), "enabled": true, "none": nil},
				}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": nil}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {Receivers: []string{"prometheus"}, Processors: []string{"metricstransform"}, Exporters: []string{"debug"}},
					},
				},
			},
		},
	}
	targetAllocator := &v1alpha1.TargetAllocator{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}

	doc, err := ReplaceConfig(otelcol, targetAllocator)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(doc), &decoded))

	prometheus := decoded["receivers"].(map[string]any)["prometheus"].(map[string]any)
	assert.Equal(t, map[string]any{
		"global": map[string]any{"external_labels": map[string]any{"cluster": "0e12", "on": "yes", "1e5": "true"}},
	}, prometheus["config"], "scrape_configs are removed, the rest of the receiver config is kept as is")
	assert.Equal(t, map[string]any{
		"endpoint":     "http://test-targetallocator.default.svc:80",
		"interval":     "30s",
		"collector_id": "${POD_NAME}",
	}, prometheus["target_allocator"])
	assert.Equal(t, map[string]any{
		"new_value": "0e12", "limit": 1000000, "ratio": 0.5, "enabled": true, "none": nil,
	}, decoded["processors"].(map[string]any)["metricstransform"])

	// The rewrite works on a copy: the CR's config is not modified.
	scrapeConfigs := otelcol.Spec.Config.Receivers.Object["prometheus"].(map[string]any)["config"].(map[string]any)["scrape_configs"]
	assert.Len(t, scrapeConfigs, 1)
	assert.NotContains(t, otelcol.Spec.Config.Receivers.Object["prometheus"].(map[string]any), "target_allocator")
}

func TestReplaceConfigRejectsMissingReceivers(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": nil}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {Exporters: []string{"debug"}},
					},
				},
			},
		},
	}
	targetAllocator := &v1alpha1.TargetAllocator{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	_, err := ReplaceConfig(otelcol, targetAllocator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receivers")
}

func TestReplaceConfigRejectsMissingPrometheus(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": nil}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": nil}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {Receivers: []string{"otlp"}, Exporters: []string{"debug"}},
					},
				},
			},
		},
	}
	targetAllocator := &v1alpha1.TargetAllocator{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	_, err := ReplaceConfig(otelcol, targetAllocator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus")
}

func TestReplaceConfigRejectsInvalidPrometheusType(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"prometheus": "string"}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": nil}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {Receivers: []string{"prometheus"}, Exporters: []string{"debug"}},
					},
				},
			},
		},
	}
	targetAllocator := &v1alpha1.TargetAllocator{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	_, err := ReplaceConfig(otelcol, targetAllocator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus")
}

func TestReplaceConfigRejectsInvalidPrometheusConfig(t *testing.T) {
	otelcol := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{
					"prometheus": map[string]any{"config": "string"},
				}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"debug": nil}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {Receivers: []string{"prometheus"}, Exporters: []string{"debug"}},
					},
				},
			},
		},
	}
	targetAllocator := &v1alpha1.TargetAllocator{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
	_, err := ReplaceConfig(otelcol, targetAllocator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prometheusConfig")
}
