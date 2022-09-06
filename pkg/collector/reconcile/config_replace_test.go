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

package reconcile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

func TestPrometheusParser(t *testing.T) {
	param, err := newParams("test/test-img", "../testdata/http_sd_config_test.yaml")
	require.NoError(t, err)
	paramTA, err := newParams("test/test-img", "../testdata/collector_with_TA.yaml")
	require.NoError(t, err)

	t.Run("should add target_allocator if settings are missing", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(param)
		require.NoError(t, err)

		// prepare
		taColCfg, configErr := ta.ConfigToCollectorTAConfig(actualConfig)
		require.NoError(t, configErr)
		assert.NotNil(t, taColCfg)
		assert.Len(t, taColCfg, 2)
		assert.Contains(t, taColCfg, "endpoint")
		assert.Contains(t, taColCfg, "collector_id")
	})

	t.Run("should merge target_allocator settings with managed endpoints", func(t *testing.T) {
		actualConfig, err := ReplaceConfig(paramTA)
		require.NoError(t, err)

		// prepare
		taColCfg, configErr := ta.ConfigToCollectorTAConfig(actualConfig)
		require.NoError(t, configErr)
		assert.Len(t, taColCfg, 3)
		assert.Contains(t, taColCfg, "endpoint")
		assert.Contains(t, taColCfg, "collector_id")
		assert.Contains(t, taColCfg, "interval")
	})

	t.Run("should not update config if TA is not enabled", func(t *testing.T) {
		param.Instance.Spec.TargetAllocator.Enabled = false
		actualConfig, err := ReplaceConfig(param)
		assert.NoError(t, err)

		// prepare
		var cfg Config
		promCfgMap, configErr := ta.ConfigToPromConfig(actualConfig)
		require.NoError(t, configErr)

		promCfg, marshalErr := yaml.Marshal(map[string]interface{}{
			"config": promCfgMap,
		})
		assert.NoError(t, marshalErr)

		unmarshalErr := yaml.UnmarshalStrict(promCfg, &cfg)
		assert.NoError(t, unmarshalErr)

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
	})

}
