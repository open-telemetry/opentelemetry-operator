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

package adapters_test

import (
	"io/ioutil"
	"testing"

	"github.com/prometheus/prometheus/discovery/http"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/reconcile"
	taa "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

type Params struct {
	Instance v1alpha1.OpenTelemetryCollector
}

type PromConfig struct {
	Instance v1alpha1.OpenTelemetryCollector
}

const defaultTestFile = "test.yaml"

var instanceUID = uuid.NewUUID()

func TestPrometheusParser(t *testing.T) {
	replicas := int32(1)
	configYAML, err := ioutil.ReadFile(defaultTestFile)
	assert.NoError(t, err)

	param := reconcile.Params{
		Instance: v1alpha1.OpenTelemetryCollector{
			TypeMeta: metav1.TypeMeta{
				Kind:       "opentelemetry.io",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       instanceUID,
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Mode: v1alpha1.ModeStatefulSet,
				Ports: []v1.ServicePort{{
					Name: "web",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					NodePort: 0,
				}},
				TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
					Enabled: true,
				},
				Replicas: &replicas,
				Config:   string(configYAML),
			},
		},
	}

	t.Run("should update config with http_sd_config", func(t *testing.T) {
		actualConfig, err := taa.ReplaceConfig(param.Instance)

		// prepare
		var cfg taa.Config
		config, err := adapters.ConfigFromString(actualConfig)
		assert.NoError(t, err)

		promCfgMap, err := taa.ConfigToPromConfig(config)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(map[string]interface{}{
			"config": promCfgMap,
		})
		assert.NoError(t, err)

		yaml.UnmarshalStrict(promCfg, &cfg)

		// test
		expectedMap := map[string]bool{
			"prometheus": false,
			"service-x":  false,
		}
		for _, scrapeConfig := range cfg.PromConfig.ScrapeConfigs {
			assert.Len(t, scrapeConfig.ServiceDiscoveryConfigs, 1)
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].Name(), "http")
			assert.Equal(t, scrapeConfig.ServiceDiscoveryConfigs[0].(*http.SDConfig).URL, "https://test-targetallocator:443/jobs/"+scrapeConfig.JobName+"/targets?collector_id=$POD_NAME")
			expectedMap[scrapeConfig.JobName] = true
		}
		for k := range expectedMap {
			assert.True(t, expectedMap[k], k)
		}
	})

	t.Run("should not update config with http_sd_config", func(t *testing.T) {
		param.Instance.Spec.TargetAllocator.Enabled = false
		actualConfig, err := taa.ReplaceConfig(param.Instance)
		assert.NoError(t, err)

		// prepare
		var cfg taa.Config
		config, err := adapters.ConfigFromString(actualConfig)
		assert.NoError(t, err)

		promCfgMap, err := taa.ConfigToPromConfig(config)
		assert.NoError(t, err)

		promCfg, err := yaml.Marshal(map[string]interface{}{
			"config": promCfgMap,
		})
		assert.NoError(t, err)

		yaml.UnmarshalStrict(promCfg, &cfg)

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
