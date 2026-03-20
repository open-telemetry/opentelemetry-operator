// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers_test

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

func TestPrometheusParser(t *testing.T) {
	parser := receivers.ReceiverFor("prometheus")
	assert.Equal(t, "__prometheus", parser.ParserName())
}

func TestPrometheusParserPorts(t *testing.T) {
	parser := receivers.ReceiverFor("prometheus")
	ports, err := parser.Ports(logger, "prometheus", map[string]any{})
	assert.NoError(t, err)
	assert.Len(t, ports, 0)
}

func TestPrometheusParserTLSProfile(t *testing.T) {
	tlsProfile := components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	})

	tests := []struct {
		name          string
		config        map[string]any
		expectVersion string
		description   string
	}{
		{
			name: "applies min_version to tls_config without one",
			config: map[string]any{
				"config": map[string]any{
					"scrape_configs": []any{
						map[string]any{
							"job_name": "test",
							"tls_config": map[string]any{
								"ca_file": "/etc/prom/ca.crt",
							},
						},
					},
				},
			},
			expectVersion: "TLS12",
		},
		{
			name: "does not override existing min_version",
			config: map[string]any{
				"config": map[string]any{
					"scrape_configs": []any{
						map[string]any{
							"job_name": "test",
							"tls_config": map[string]any{
								"ca_file":     "/etc/prom/ca.crt",
								"min_version": "TLS13",
							},
						},
					},
				},
			},
			expectVersion: "TLS13",
		},
		{
			name: "does not add tls_config when not present",
			config: map[string]any{
				"config": map[string]any{
					"scrape_configs": []any{
						map[string]any{
							"job_name": "test",
						},
					},
				},
			},
			expectVersion: "",
		},
		{
			name:          "handles nil config",
			config:        nil,
			expectVersion: "",
		},
		{
			name: "handles multiple scrape configs",
			config: map[string]any{
				"config": map[string]any{
					"scrape_configs": []any{
						map[string]any{
							"job_name": "job1",
							"tls_config": map[string]any{
								"ca_file": "/ca1.crt",
							},
						},
						map[string]any{
							"job_name": "job2",
							"tls_config": map[string]any{
								"ca_file": "/ca2.crt",
							},
						},
						map[string]any{
							"job_name": "job3-no-tls",
						},
					},
				},
			},
			expectVersion: "TLS12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := receivers.ReceiverFor("prometheus")

			result, err := parser.GetDefaultConfig(logger, tt.config, components.WithTLSProfile(tlsProfile))
			require.NoError(t, err)

			if tt.config == nil {
				assert.Nil(t, result)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				return
			}

			cfgMap, ok := resultMap["config"].(map[string]any)
			if !ok {
				return
			}

			scrapeConfigs, ok := cfgMap["scrape_configs"].([]any)
			if !ok {
				return
			}

			for _, sc := range scrapeConfigs {
				scMap := sc.(map[string]any)
				tlsCfg, hasTLS := scMap["tls_config"]
				if !hasTLS {
					continue
				}
				tlsMap := tlsCfg.(map[string]any)
				if tt.expectVersion != "" {
					assert.Equal(t, tt.expectVersion, tlsMap["min_version"],
						"scrape config %s", scMap["job_name"])
				}
			}
		})
	}
}

func TestPrometheusParserNoTLSProfile(t *testing.T) {
	config := map[string]any{
		"config": map[string]any{
			"scrape_configs": []any{
				map[string]any{
					"job_name": "test",
					"tls_config": map[string]any{
						"ca_file": "/etc/prom/ca.crt",
					},
				},
			},
		},
	}

	parser := receivers.ReceiverFor("prometheus")
	result, err := parser.GetDefaultConfig(logger, config)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	cfgMap := resultMap["config"].(map[string]any)
	scrapeConfigs := cfgMap["scrape_configs"].([]any)
	tlsMap := scrapeConfigs[0].(map[string]any)["tls_config"].(map[string]any)

	_, hasMinVersion := tlsMap["min_version"]
	assert.False(t, hasMinVersion, "should not set min_version when no TLS profile is provided")
}

func TestPrometheusParserTLS13(t *testing.T) {
	tlsProfile := components.NewStaticTLSProfile(tls.VersionTLS13, nil)

	config := map[string]any{
		"config": map[string]any{
			"scrape_configs": []any{
				map[string]any{
					"job_name": "test",
					"tls_config": map[string]any{
						"ca_file": "/etc/prom/ca.crt",
					},
				},
			},
		},
	}

	parser := receivers.ReceiverFor("prometheus")
	result, err := parser.GetDefaultConfig(logger, config, components.WithTLSProfile(tlsProfile))
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	cfgMap := resultMap["config"].(map[string]any)
	scrapeConfigs := cfgMap["scrape_configs"].([]any)
	tlsMap := scrapeConfigs[0].(map[string]any)["tls_config"].(map[string]any)

	assert.Equal(t, "TLS13", tlsMap["min_version"])
}
