// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAddressEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		address       string
		expectedHost  string
		expectedPort  int32
		expectedError bool
	}{
		{
			name:          "valid address with port",
			address:       "localhost:8080",
			expectedHost:  "localhost",
			expectedPort:  8080,
			expectedError: false,
		},
		{
			name:          "valid address with port and path",
			address:       "localhost:8080/metrics",
			expectedHost:  "localhost:8080/metrics",
			expectedPort:  defaultServicePort,
			expectedError: false,
		},
		{
			name:          "address without port",
			address:       "localhost",
			expectedHost:  "localhost",
			expectedPort:  defaultServicePort,
			expectedError: false,
		},
		{
			name:          "address with environment variable port",
			address:       "localhost:${POD_IP}",
			expectedHost:  "",
			expectedPort:  0,
			expectedError: true,
		},
		{
			name:          "address with environment variable port with env prefix",
			address:       "localhost:${env:POD_IP}",
			expectedHost:  "",
			expectedPort:  0,
			expectedError: true,
		},
		{
			name:          "empty address",
			address:       "",
			expectedHost:  "",
			expectedPort:  defaultServicePort,
			expectedError: false,
		},
		{
			name:          "invalid port",
			address:       "localhost:invalid",
			expectedHost:  "localhost:invalid",
			expectedPort:  defaultServicePort,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseAddressEndpoint(tt.address)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, host, tt.expectedHost)
				require.Equal(t, port, tt.expectedPort)
			}
		})
	}
}

func TestAddPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		arr      []string
		expected []string
	}{
		{
			name:     "empty array",
			prefix:   "prefix-",
			arr:      []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			prefix:   "prefix-",
			arr:      []string{"item"},
			expected: []string{"prefix-item"},
		},
		{
			name:     "multiple elements",
			prefix:   "prefix-",
			arr:      []string{"item1", "item2", "item3"},
			expected: []string{"prefix-item1", "prefix-item2", "prefix-item3"},
		},
		{
			name:     "empty prefix",
			prefix:   "",
			arr:      []string{"item1", "item2"},
			expected: []string{"item1", "item2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addPrefix(tt.prefix, tt.arr)
			require.Equal(t, result, tt.expected)
		})
	}
}

func TestGetNullValue(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		expected []string
	}{
		{
			name:     "empty map",
			cfg:      map[string]any{},
			expected: []string{},
		},
		{
			name: "single null value",
			cfg: map[string]any{
				"key": nil,
			},
			expected: []string{"key:"},
		},
		{
			name: "multiple null values",
			cfg: map[string]any{
				"key1": nil,
				"key2": nil,
			},
			expected: []string{"key1:", "key2:"},
		},
		{
			name: "nested null values",
			cfg: map[string]any{
				"parent": map[string]any{
					"child": nil,
				},
			},
			expected: []string{"parent.child:"},
		},
		{
			name: "deeply nested null values",
			cfg: map[string]any{
				"parent": map[string]any{
					"child": map[string]any{
						"grandchild": nil,
					},
				},
			},
			expected: []string{"parent.child.grandchild:"},
		},
		{
			name: "mixed null and non-null values",
			cfg: map[string]any{
				"key1": nil,
				"key2": "value",
				"key3": map[string]any{
					"child1": nil,
					"child2": "value",
				},
			},
			expected: []string{"key1:", "key3.child1:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNullValuedKeys(tt.cfg)
			require.Equal(t, len(result), len(tt.expected))

			for _, expected := range tt.expected {
				found := slices.Contains(result, expected)
				require.True(t, found, "getNullValuedKeys() missing expected value: %s", expected)
			}
		})
	}
}

func TestNormalizeConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "remove nil values",
			input: map[string]any{
				"key1": "value1",
				"key2": nil,
			},
			expected: map[string]any{
				"key1": "value1",
			},
		},
		{
			name: "convert port float64 to int32",
			input: map[string]any{
				"port": float64(8080),
			},
			expected: map[string]any{
				"port": int32(8080),
			},
		},
		{
			name: "normalize nested map",
			input: map[string]any{
				"parent": map[string]any{
					"child": nil,
					"port":  float64(8080),
				},
			},
			expected: map[string]any{
				"parent": map[string]any{
					"port": int32(8080),
				},
			},
		},
		{
			name: "normalize array with nil values",
			input: map[string]any{
				"items": []any{
					nil,
					map[string]any{
						"key": "value",
					},
				},
			},
			expected: map[string]any{
				"items": []any{
					map[string]any{},
					map[string]any{
						"key": "value",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeConfig(tt.input)
			require.Equal(t, tt.input, tt.expected)
		})
	}
}
