// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
			"endpoint":     "http://test-targetallocator:80",
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
			"endpoint":     "http://test-targetallocator:80",
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
