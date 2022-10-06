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

package config

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/file"
	"github.com/stretchr/testify/assert"
)

const testFile = "./testdata/config_test.yaml"

func TestConfigLoad(t *testing.T) {
	expectedFileSDConfig := &file.SDConfig{
		Files:           []string{"./file_sd_test.json"},
		RefreshInterval: model.Duration(300000000000),
	}
	expectedStaticSDConfig := discovery.StaticConfig{
		{
			Targets: []model.LabelSet{
				{model.AddressLabel: "prom.domain:9001"},
				{model.AddressLabel: "prom.domain:9002"},
				{model.AddressLabel: "prom.domain:9003"},
			},
			Labels: model.LabelSet{
				"my": "label",
			},
			Source: "0",
		},
	}

	cfg := Config{}
	err := unmarshal(&cfg, testFile)
	assert.NoError(t, err)

	scrapeConfig := *cfg.Config.ScrapeConfigs[0]
	actualFileSDConfig := scrapeConfig.ServiceDiscoveryConfigs[0]
	actulaStaticSDConfig := scrapeConfig.ServiceDiscoveryConfigs[1]
	t.Log(actulaStaticSDConfig)

	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/instance"], "default.test")
	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/managed-by"], "opentelemetry-operator")
	assert.Equal(t, scrapeConfig.JobName, "prometheus")
	assert.Equal(t, expectedFileSDConfig, actualFileSDConfig)
	assert.Equal(t, expectedStaticSDConfig, actulaStaticSDConfig)
}
