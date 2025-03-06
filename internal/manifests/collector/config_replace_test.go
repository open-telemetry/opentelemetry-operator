// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
)

func TestPrometheusParser(t *testing.T) {
	param, err := newParams("test/test-img", "testdata/http_sd_config_test.yaml")
	assert.NoError(t, err)

	t.Run("should update config with targetAllocator block if block not present", func(t *testing.T) {
		// Set up the test scenario
		actualConfig, err := ReplaceConfig(param.OtelCol, param.TargetAllocator)
		assert.NoError(t, err)

		// Verify the expected changes in the config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		prometheusConfig := promCfgMap["config"].(map[interface{}]interface{})

		assert.NotContains(t, prometheusConfig, "scrape_configs")

		expectedTAConfig := map[interface{}]interface{}{
			"endpoint":     "http://test-targetallocator:80",
			"interval":     "30s",
			"collector_id": "${POD_NAME}",
		}
		assert.Equal(t, expectedTAConfig, promCfgMap["target_allocator"])
		assert.NoError(t, err)
	})

	t.Run("should update config with targetAllocator block if block already present", func(t *testing.T) {
		// Set up the test scenario
		paramTa, err := newParams("test/test-img", "testdata/http_sd_config_ta_test.yaml")
		require.NoError(t, err)

		actualConfig, err := ReplaceConfig(paramTa.OtelCol, param.TargetAllocator)
		assert.NoError(t, err)

		// Verify the expected changes in the config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		prometheusConfig := promCfgMap["config"].(map[interface{}]interface{})

		assert.NotContains(t, prometheusConfig, "scrape_configs")

		expectedTAConfig := map[interface{}]interface{}{
			"endpoint":     "http://test-targetallocator:80",
			"interval":     "30s",
			"collector_id": "${POD_NAME}",
		}
		assert.Equal(t, expectedTAConfig, promCfgMap["target_allocator"])
		assert.NoError(t, err)
	})

	t.Run("should not update config with http_sd_config", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(param.OtelCol, nil)
		assert.NoError(t, err)

		// prepare
		var cfg Config
		promCfgMap, err := ta.ConfigToPromConfig(actualConfig)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(promCfgMap)
		assert.NoError(t, err)

		err = yaml.UnmarshalStrict(promCfg, &cfg)
		assert.NoError(t, err)

		// test
		expectedMap := map[string]bool{
			"prometheus": false,
			"service-x":  false,
		}
		for _, scrapeConfig := range cfg.PromConfig.ScrapeConfigs {
			assert.Len(t, scrapeConfig.ServiceDiscoveryConfigs, 2)
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].Name(), "file")
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[1].Name(), "static")
			expectedMap[scrapeConfig.JobName] = true
		}
		for k := range expectedMap {
			assert.True(t, expectedMap[k], k)
		}
		assert.True(t, cfg.TargetAllocConfig == nil)
	})

}

func TestReplaceConfig(t *testing.T) {
	param, err := newParams("test/test-img", "testdata/relabel_config_original.yaml")
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
}
