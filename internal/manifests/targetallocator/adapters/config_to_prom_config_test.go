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
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"

	"github.com/stretchr/testify/assert"
)

func TestExtractPromConfigFromConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  prometheus:
    config:
      scrape_config:
        job_name: otel-collector
        scrape_interval: 10s
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`
	expectedData := map[interface{}]interface{}{
		"config": map[interface{}]interface{}{
			"scrape_config": map[interface{}]interface{}{
				"job_name":        "otel-collector",
				"scrape_interval": "10s",
			},
		},
	}

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, expectedData, promConfig)
}

func TestExtractPromConfigWithTAConfigFromConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  prometheus:
    config:
      scrape_config:
        job_name: otel-collector
        scrape_interval: 10s
    target_allocator:
      endpoint: "test:80"
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`
	expectedData := map[interface{}]interface{}{
		"config": map[interface{}]interface{}{
			"scrape_config": map[interface{}]interface{}{
				"job_name":        "otel-collector",
				"scrape_interval": "10s",
			},
		},
		"target_allocator": map[interface{}]interface{}{
			"endpoint": "test:80",
		},
	}

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.NoError(t, err)

	// verify
	assert.Equal(t, expectedData, promConfig)
}

func TestExtractPromConfigFromNullConfig(t *testing.T) {
	configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`

	// test
	promConfig, err := ta.ConfigToPromConfig(configStr)
	assert.Equal(t, err, fmt.Errorf("no prometheus available as part of the configuration"))

	// verify
	assert.True(t, reflect.ValueOf(promConfig).IsNil())
}

func TestUnescapeDollarSignsInPromConfig(t *testing.T) {
	actual := `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        relabel_configs:
        - source_labels: ['__meta_service_id']
          target_label: 'job'
          replacement: 'my_service_$$1'
        - source_labels: ['__meta_service_name']
          target_label: 'instance'
          replacement: '$1'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$$1_$2'
`
	expected := `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        relabel_configs:
        - source_labels: ['__meta_service_id']
          target_label: 'job'
          replacement: 'my_service_$1'
        - source_labels: ['__meta_service_name']
          target_label: 'instance'
          replacement: '$1'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$1_$2'
`

	config, err := ta.UnescapeDollarSignsInPromConfig(actual)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expectedConfig, err := ta.UnescapeDollarSignsInPromConfig(expected)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Errorf("unexpected config: got %v, want %v", config, expectedConfig)
	}
}

func TestAddHTTPSDConfigToPromConfig(t *testing.T) {
	t.Run("ValidConfiguration, add http_sd_config", func(t *testing.T) {
		cfg := map[interface{}]interface{}{
			"config": map[interface{}]interface{}{
				"scrape_configs": []interface{}{
					map[interface{}]interface{}{
						"job_name": "test_job",
						"static_configs": []interface{}{
							map[interface{}]interface{}{
								"targets": []interface{}{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}
		taServiceName := "test-service"
		expectedCfg := map[interface{}]interface{}{
			"config": map[interface{}]interface{}{
				"scrape_configs": []interface{}{
					map[interface{}]interface{}{
						"job_name": "test_job",
						"http_sd_configs": []interface{}{
							map[string]interface{}{
								"url": fmt.Sprintf("http://%s:80/jobs/%s/targets?collector_id=$POD_NAME", taServiceName, url.QueryEscape("test_job")),
							},
						},
					},
				},
			},
		}

		actualCfg, err := ta.AddHTTPSDConfigToPromConfig(cfg, taServiceName)
		assert.NoError(t, err)
		assert.Equal(t, expectedCfg, actualCfg)
	})

	t.Run("invalid config property, returns error", func(t *testing.T) {
		cfg := map[interface{}]interface{}{
			"config": map[interface{}]interface{}{
				"job_name": "test_job",
				"static_configs": []interface{}{
					map[interface{}]interface{}{
						"targets": []interface{}{
							"localhost:9090",
						},
					},
				},
			},
		}

		taServiceName := "test-service"

		_, err := ta.AddHTTPSDConfigToPromConfig(cfg, taServiceName)
		assert.Error(t, err)
		assert.EqualError(t, err, "no scrape_configs available as part of the configuration")
	})
}

func TestAddTAConfigToPromConfig(t *testing.T) {
	t.Run("should return expected prom config map with TA config", func(t *testing.T) {
		cfg := map[interface{}]interface{}{
			"config": map[interface{}]interface{}{
				"scrape_configs": []interface{}{
					map[interface{}]interface{}{
						"job_name": "test_job",
						"static_configs": []interface{}{
							map[interface{}]interface{}{
								"targets": []interface{}{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}

		taServiceName := "test-targetallocator"

		expectedResult := map[interface{}]interface{}{
			"config": map[interface{}]interface{}{},
			"target_allocator": map[interface{}]interface{}{
				"endpoint":     "http://test-targetallocator:80",
				"interval":     "30s",
				"collector_id": "${POD_NAME}",
			},
		}

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("missing or invalid prometheusConfig property, returns error", func(t *testing.T) {
		testCases := []struct {
			name    string
			cfg     map[interface{}]interface{}
			errText string
		}{
			{
				name:    "missing config property",
				cfg:     map[interface{}]interface{}{},
				errText: "no prometheusConfig available as part of the configuration",
			},
			{
				name: "invalid config property",
				cfg: map[interface{}]interface{}{
					"config": "invalid",
				},
				errText: "prometheusConfig property in the configuration doesn't contain valid prometheusConfig",
			},
		}

		taServiceName := "test-targetallocator"

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := ta.AddTAConfigToPromConfig(tc.cfg, taServiceName)

				assert.Error(t, err)
				assert.EqualError(t, err, tc.errText)
			})
		}
	})
}

func TestValidatePromConfig(t *testing.T) {
	testCases := []struct {
		description                   string
		config                        map[interface{}]interface{}
		targetAllocatorEnabled        bool
		targetAllocatorRewriteEnabled bool
		expectedError                 error
	}{
		{
			description:                   "target_allocator and rewrite enabled",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: true,
			expectedError:                 nil,
		},
		{
			description: "target_allocator enabled, target_allocator section present",
			config: map[interface{}]interface{}{
				"target_allocator": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description: "target_allocator enabled, config section present",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description:                   "target_allocator enabled, neither section present",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        true,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 errors.New("either target allocator or prometheus config needs to be present"),
		},
		{
			description: "target_allocator disabled, config section present",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{},
			},
			targetAllocatorEnabled:        false,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 nil,
		},
		{
			description:                   "target_allocator disabled, config section not present",
			config:                        map[interface{}]interface{}{},
			targetAllocatorEnabled:        false,
			targetAllocatorRewriteEnabled: false,
			expectedError:                 fmt.Errorf("no %s available as part of the configuration", "prometheusConfig"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.description, func(t *testing.T) {
			err := ta.ValidatePromConfig(testCase.config, testCase.targetAllocatorEnabled, testCase.targetAllocatorRewriteEnabled)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

func TestValidateTargetAllocatorConfig(t *testing.T) {
	testCases := []struct {
		description                 string
		config                      map[interface{}]interface{}
		targetAllocatorPrometheusCR bool
		expectedError               error
	}{
		{
			description: "scrape configs present and PrometheusCR enabled",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{
					"scrape_configs": []interface{}{
						map[interface{}]interface{}{
							"job_name": "test_job",
							"static_configs": []interface{}{
								map[interface{}]interface{}{
									"targets": []interface{}{
										"localhost:9090",
									},
								},
							},
						},
					},
				},
			},
			targetAllocatorPrometheusCR: true,
			expectedError:               nil,
		},
		{
			description: "scrape configs present and PrometheusCR disabled",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{
					"scrape_configs": []interface{}{
						map[interface{}]interface{}{
							"job_name": "test_job",
							"static_configs": []interface{}{
								map[interface{}]interface{}{
									"targets": []interface{}{
										"localhost:9090",
									},
								},
							},
						},
					},
				},
			},
			targetAllocatorPrometheusCR: false,
			expectedError:               nil,
		},
		{
			description:                 "receiver config empty and PrometheusCR enabled",
			config:                      map[interface{}]interface{}{},
			targetAllocatorPrometheusCR: true,
			expectedError:               nil,
		},
		{
			description:                 "receiver config empty and PrometheusCR disabled",
			config:                      map[interface{}]interface{}{},
			targetAllocatorPrometheusCR: false,
			expectedError:               fmt.Errorf("no %s available as part of the configuration", "prometheusConfig"),
		},
		{
			description: "scrape configs empty and PrometheusCR disabled",
			config: map[interface{}]interface{}{
				"config": map[interface{}]interface{}{
					"scrape_configs": []interface{}{},
				},
			},
			targetAllocatorPrometheusCR: false,
			expectedError:               fmt.Errorf("either at least one scrape config needs to be defined or PrometheusCR needs to be enabled"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.description, func(t *testing.T) {
			err := ta.ValidateTargetAllocatorConfig(testCase.targetAllocatorPrometheusCR, testCase.config)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}
