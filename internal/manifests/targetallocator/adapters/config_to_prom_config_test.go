// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adapters_test

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
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
	expectedData := map[any]any{
		"config": map[any]any{
			"scrape_config": map[any]any{
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
	expectedData := map[any]any{
		"config": map[any]any{
			"scrape_config": map[any]any{
				"job_name":        "otel-collector",
				"scrape_interval": "10s",
			},
		},
		"target_allocator": map[any]any{
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
	assert.Equal(t, err, errors.New("no prometheus available as part of the configuration"))

	// verify
	assert.True(t, reflect.ValueOf(promConfig).IsNil())
}

func TestUnescapeDollarSignsInPromConfig(t *testing.T) {
	testCases := []struct {
		description string
		input       string
		expected    string
	}{
		{
			description: "no scrape configs",
			input: `
receivers:
  prometheus:
    config:
      scrape_configs: []
`,
			expected: `
receivers:
  prometheus:
    config:
      scrape_configs: []
`,
		},
		{
			description: "only metric relabellings",
			input: `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$$1_$2'
`,
			expected: `
receivers:
  prometheus:
    config:
      scrape_configs:
      - job_name: 'example'
        metric_relabel_configs:
        - source_labels: ['job']
          target_label: 'job'
          replacement: '$1_$2'
`,
		},
		{
			description: "only target relabellings",
			input: `
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
`,
			expected: `
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
`,
		},
		{
			description: "full",
			input: `
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
`,
			expected: `
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
`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			config, err := ta.UnescapeDollarSignsInPromConfig(testCase.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			expectedConfig, err := ta.UnescapeDollarSignsInPromConfig(testCase.expected)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(config, expectedConfig) {
				t.Errorf("unexpected config: got %v, want %v", config, expectedConfig)
			}
		})
	}
}

func TestAddHTTPSDConfigToPromConfig(t *testing.T) {
	t.Run("ValidConfiguration, add http_sd_config", func(t *testing.T) {
		cfg := map[any]any{
			"config": map[any]any{
				"scrape_configs": []any{
					map[any]any{
						"job_name": "test_job",
						"static_configs": []any{
							map[any]any{
								"targets": []any{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}
		taServiceName := "test-service"
		expectedCfg := map[any]any{
			"config": map[any]any{
				"scrape_configs": []any{
					map[any]any{
						"job_name": "test_job",
						"http_sd_configs": []any{
							map[string]any{
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
		cfg := map[any]any{
			"config": map[any]any{
				"job_name": "test_job",
				"static_configs": []any{
					map[any]any{
						"targets": []any{
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
		cfg := map[any]any{
			"config": map[any]any{
				"scrape_configs": []any{
					map[any]any{
						"job_name": "test_job",
						"static_configs": []any{
							map[any]any{
								"targets": []any{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}

		taServiceName := "test-targetallocator"

		expectedResult := map[any]any{
			"config": map[any]any{},
			"target_allocator": map[any]any{
				"endpoint":     "http://test-targetallocator:80",
				"interval":     "30s",
				"collector_id": "${POD_NAME}",
			},
		}

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName)

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})

	t.Run("invalid prometheusConfig property, returns error", func(t *testing.T) {
		testCases := []struct {
			name    string
			cfg     map[any]any
			errText string
		}{
			{
				name: "invalid config property",
				cfg: map[any]any{
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

	t.Run("TA-only mode: no config block, only target_allocator set", func(t *testing.T) {
		// Regression test for #2998. The user supplied only a `target_allocator:` block
		// on the prometheus receiver (no `config:`). Reconciliation must not fail, and
		// no spurious `config:` block should be inserted — the prometheus receiver itself
		// does not require one.
		cfg := map[any]any{
			"target_allocator": map[any]any{
				"endpoint": "user-supplied-endpoint",
			},
		}
		taServiceName := "test-targetallocator"

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		taSection, ok := result["target_allocator"].(map[any]any)
		assert.True(t, ok)
		// Operator-managed endpoint replaces the user-supplied one.
		assert.Equal(t, "http://test-targetallocator:80", taSection["endpoint"])
		// No `config:` block should be added when the user didn't supply one.
		_, hasConfig := result["config"]
		assert.False(t, hasConfig)
	})

	t.Run("TA-only mode: no config block, no target_allocator block", func(t *testing.T) {
		// Edge case for #2998: an entirely empty receiver succeeds once TA is enabled.
		// The operator inserts the target_allocator block but leaves `config:` absent.
		cfg := map[any]any{}
		taServiceName := "test-targetallocator"

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		_, hasConfig := result["config"]
		assert.False(t, hasConfig)
		_, hasTA := result["target_allocator"]
		assert.True(t, hasTA)
	})
}

func TestValidatePromConfig(t *testing.T) {
	testCases := []struct {
		description            string
		config                 map[any]any
		targetAllocatorEnabled bool
		expectedError          error
	}{
		{
			description:            "target_allocator enabled",
			config:                 map[any]any{},
			targetAllocatorEnabled: true,
			expectedError:          nil,
		},
		{
			description: "target_allocator enabled, target_allocator section present",
			config: map[any]any{
				"target_allocator": map[any]any{},
			},
			targetAllocatorEnabled: true,
			expectedError:          nil,
		},
		{
			description: "target_allocator enabled, config section present",
			config: map[any]any{
				"config": map[any]any{},
			},
			targetAllocatorEnabled: true,
			expectedError:          nil,
		},
		{
			description: "target_allocator disabled, config section present",
			config: map[any]any{
				"config": map[any]any{},
			},
			targetAllocatorEnabled: false,
			expectedError:          nil,
		},
		{
			description:            "target_allocator disabled, config section not present",
			config:                 map[any]any{},
			targetAllocatorEnabled: false,
			expectedError:          fmt.Errorf("no %s available as part of the configuration", "prometheusConfig"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			err := ta.ValidatePromConfig(testCase.config, testCase.targetAllocatorEnabled)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

func TestValidateTargetAllocatorConfig(t *testing.T) {
	testCases := []struct {
		description                 string
		config                      map[any]any
		targetAllocatorPrometheusCR bool
		expectedError               error
	}{
		{
			description: "scrape configs present and PrometheusCR enabled",
			config: map[any]any{
				"config": map[any]any{
					"scrape_configs": []any{
						map[any]any{
							"job_name": "test_job",
							"static_configs": []any{
								map[any]any{
									"targets": []any{
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
			config: map[any]any{
				"config": map[any]any{
					"scrape_configs": []any{
						map[any]any{
							"job_name": "test_job",
							"static_configs": []any{
								map[any]any{
									"targets": []any{
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
			config:                      map[any]any{},
			targetAllocatorPrometheusCR: true,
			expectedError:               nil,
		},
		{
			// TA-only mode (#2998): receiver has no `config:` block, so the user is
			// delegating scrape configuration to the target allocator. Validator permits.
			description:                 "receiver config empty and PrometheusCR disabled",
			config:                      map[any]any{},
			targetAllocatorPrometheusCR: false,
			expectedError:               nil,
		},
		{
			// TA-only mode with explicit target_allocator block — same as the empty case,
			// permitted because no `config:` means scrape configs come from the TA itself.
			description: "no config block but target_allocator set, PrometheusCR disabled",
			config: map[any]any{
				"target_allocator": map[any]any{
					"endpoint": "http://my-ta:80",
				},
			},
			targetAllocatorPrometheusCR: false,
			expectedError:               nil,
		},
		{
			description: "scrape configs empty and PrometheusCR disabled",
			config: map[any]any{
				"config": map[any]any{
					"scrape_configs": []any{},
				},
			},
			targetAllocatorPrometheusCR: false,
			expectedError:               errors.New("either at least one scrape config needs to be defined or PrometheusCR needs to be enabled"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			err := ta.ValidateTargetAllocatorConfig(testCase.targetAllocatorPrometheusCR, testCase.config)
			assert.Equal(t, testCase.expectedError, err)
		})
	}
}

func TestAddTAConfigToPromConfigWithTLSConfig(t *testing.T) {
	t.Run("should return expected prom config map with TA config and TLS config", func(t *testing.T) {
		cfg := map[any]any{
			"config": map[any]any{
				"scrape_configs": []any{
					map[any]any{
						"job_name": "test_job",
						"static_configs": []any{
							map[any]any{
								"targets": []any{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}

		taServiceName := "test-targetallocator"

		expectedResult := map[any]any{
			"config": map[any]any{},
			"target_allocator": map[any]any{
				"endpoint":     "https://test-targetallocator:443",
				"interval":     "30s",
				"collector_id": "${POD_NAME}",
				"tls": map[any]any{
					"ca_file":         "ca.crt",
					"cert_file":       "tls.crt",
					"key_file":        "tls.key",
					"reload_interval": "5m",
				},
			},
		}

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName, ta.WithTLSConfig("ca.crt", "tls.crt", "tls.key", taServiceName))

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}

func TestAddTAConfigToPromConfigWithCollectorTargetReloadInterval(t *testing.T) {
	t.Run("should return expected prom config map with TA config and collector target reload interval", func(t *testing.T) {
		cfg := map[any]any{
			"config": map[any]any{
				"scrape_configs": []any{
					map[any]any{
						"job_name": "test_job",
						"static_configs": []any{
							map[any]any{
								"targets": []any{
									"localhost:9090",
								},
							},
						},
					},
				},
			},
		}

		taServiceName := "test-targetallocator"

		expectedResult := map[any]any{
			"config": map[any]any{},
			"target_allocator": map[any]any{
				"endpoint":     "http://test-targetallocator:80",
				"interval":     "10s",
				"collector_id": "${POD_NAME}",
			},
		}

		result, err := ta.AddTAConfigToPromConfig(cfg, taServiceName, ta.WithCollectorTargetReloadInterval("10s"))

		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
	})
}
