// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"os"
	"testing"

	"github.com/prometheus/prometheus/discovery/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"gopkg.in/yaml.v2"

	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestPrometheusParser(t *testing.T) {
	param, err := newParams("test/test-img", "testdata/http_sd_config_test.yaml")
	assert.NoError(t, err)

	t.Run("should update config with http_sd_config", func(t *testing.T) {
		err := colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), false)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), true)
		})
		actualConfig, err := ReplaceConfig(param.OtelCol)
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
			assert.Len(t, scrapeConfig.ServiceDiscoveryConfigs, 1)
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].Name(), "http")
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].(*http.SDConfig).URL, "http://test-targetallocator:80/jobs/"+scrapeConfig.JobName+"/targets?collector_id=$POD_NAME")
			expectedMap[scrapeConfig.JobName] = true
		}
		for k := range expectedMap {
			assert.True(t, expectedMap[k], k)
		}
		assert.True(t, cfg.TargetAllocConfig == nil)
	})

	t.Run("should update config with targetAllocator block if block not present", func(t *testing.T) {
		// Set up the test scenario
		param.OtelCol.Spec.TargetAllocator.Enabled = true
		actualConfig, err := ReplaceConfig(param.OtelCol)
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
		paramTa.OtelCol.Spec.TargetAllocator.Enabled = true

		actualConfig, err := ReplaceConfig(paramTa.OtelCol)
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
		param.OtelCol.Spec.TargetAllocator.Enabled = false
		actualConfig, err := ReplaceConfig(param.OtelCol)
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
		param.OtelCol.Spec.TargetAllocator.Enabled = false
		expectedConfigBytes, err := os.ReadFile("testdata/relabel_config_original.yaml")
		assert.NoError(t, err)
		expectedConfig := string(expectedConfigBytes)

		actualConfig, err := ReplaceConfig(param.OtelCol)
		assert.NoError(t, err)

		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should rewrite scrape configs with SD config when TargetAllocator is enabled and feature flag is not set", func(t *testing.T) {
		err := colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), false)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = colfeaturegate.GlobalRegistry().Set(featuregate.EnableTargetAllocatorRewrite.ID(), true)
		})

		param.OtelCol.Spec.TargetAllocator.Enabled = true

		expectedConfigBytes, err := os.ReadFile("testdata/relabel_config_expected_with_sd_config.yaml")
		assert.NoError(t, err)
		expectedConfig := string(expectedConfigBytes)

		actualConfig, err := ReplaceConfig(param.OtelCol)
		assert.NoError(t, err)

		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should remove scrape configs if TargetAllocator is enabled and feature flag is set", func(t *testing.T) {
		param.OtelCol.Spec.TargetAllocator.Enabled = true

		expectedConfigBytes, err := os.ReadFile("testdata/config_expected_targetallocator.yaml")
		assert.NoError(t, err)
		expectedConfig := string(expectedConfigBytes)

		actualConfig, err := ReplaceConfig(param.OtelCol)
		assert.NoError(t, err)

		assert.Equal(t, expectedConfig, actualConfig)
	})
}
