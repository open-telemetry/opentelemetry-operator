// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

func TestPrometheusParser(t *testing.T) {
	parser := receivers.ReceiverFor("prometheus")
	assert.Equal(t, "__prometheus", parser.ParserName())
}

func TestPrometheusParserPorts(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]any
		expectedPort int32
	}{
		{
			name:   "no config returns no ports",
			config: map[string]any{},
		},
		{
			name: "api_server disabled returns no ports",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": false,
					"server_config": map[string]any{
						"endpoint": "0.0.0.0:9091",
					},
				},
			},
		},
		{
			name: "api_server enabled without server_config returns no ports",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": true,
				},
			},
		},
		{
			name: "api_server enabled with empty endpoint returns no ports",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": true,
					"server_config": map[string]any{
						"endpoint": "",
					},
				},
			},
		},
		{
			name: "api_server enabled with endpoint returns port",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": true,
					"server_config": map[string]any{
						"endpoint": "0.0.0.0:9091",
					},
				},
			},
			expectedPort: 9091,
		},
		{
			name: "api_server enabled with localhost endpoint returns port",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": true,
					"server_config": map[string]any{
						"endpoint": "localhost:9090",
					},
				},
			},
			expectedPort: 9090,
		},
		{
			name: "api_server with scrape_configs still parses port",
			config: map[string]any{
				"api_server": map[string]any{
					"enabled": true,
					"server_config": map[string]any{
						"endpoint": "0.0.0.0:9091",
					},
				},
				"config": map[string]any{
					"scrape_configs": []any{
						map[string]any{
							"job_name": "test",
						},
					},
				},
			},
			expectedPort: 9091,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := receivers.ReceiverFor("prometheus")
			ports, err := parser.Ports(logger, "prometheus", tt.config)
			require.NoError(t, err)
			if tt.expectedPort == 0 {
				assert.Empty(t, ports)
			} else {
				require.Len(t, ports, 1)
				assert.Equal(t, tt.expectedPort, ports[0].Port)
			}
		})
	}
}
