// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporters

import (
	"crypto/tls"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestExporterParserTLSProfile(t *testing.T) {
	tlsProfile := components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	})

	tests := []struct {
		name             string
		config           map[string]any
		expectMinVersion string
		expectCiphers    []string
	}{
		{
			name: "applies min_version and cipher_suites to tls block",
			config: map[string]any{
				"endpoint": "tempo.example.com:4317",
				"tls":      map[string]any{},
			},
			expectMinVersion: "1.2",
			expectCiphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		},
		{
			name: "does not override existing min_version",
			config: map[string]any{
				"endpoint": "tempo.example.com:4317",
				"tls": map[string]any{
					"min_version": "1.3",
				},
			},
			expectMinVersion: "1.3",
			expectCiphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		},
		{
			name: "does not override existing cipher_suites",
			config: map[string]any{
				"endpoint": "tempo.example.com:4317",
				"tls": map[string]any{
					"cipher_suites": []string{"TLS_AES_256_GCM_SHA384"},
				},
			},
			expectMinVersion: "1.2",
			expectCiphers:    []string{"TLS_AES_256_GCM_SHA384"},
		},
		{
			name: "does not add tls block when not present",
			config: map[string]any{
				"endpoint": "tempo.example.com:4317",
			},
			expectMinVersion: "",
			expectCiphers:    nil,
		},
		{
			name:             "handles nil config",
			config:           nil,
			expectMinVersion: "",
			expectCiphers:    nil,
		},
		{
			name:             "handles empty config",
			config:           map[string]any{},
			expectMinVersion: "",
			expectCiphers:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("otlp")

			var config any
			if tt.config != nil {
				config = tt.config
			}

			result, err := parser.GetDefaultConfig(logr.Discard(), config, components.WithTLSProfile(tlsProfile))
			require.NoError(t, err)

			if tt.config == nil {
				assert.Nil(t, result)
				return
			}

			resultMap := result.(map[string]any)
			tlsCfg, hasTLS := resultMap["tls"]
			if tt.expectMinVersion == "" && tt.expectCiphers == nil {
				if hasTLS {
					tlsMap := tlsCfg.(map[string]any)
					_, hasMin := tlsMap["min_version"]
					_, hasCipher := tlsMap["cipher_suites"]
					assert.False(t, hasMin, "should not have min_version")
					assert.False(t, hasCipher, "should not have cipher_suites")
				}
				return
			}

			require.True(t, hasTLS)
			tlsMap := tlsCfg.(map[string]any)
			assert.Equal(t, tt.expectMinVersion, tlsMap["min_version"])
			assert.Equal(t, tt.expectCiphers, tlsMap["cipher_suites"])
		})
	}
}

func TestExporterParserNoTLSProfile(t *testing.T) {
	config := map[string]any{
		"endpoint": "tempo.example.com:4317",
		"tls":      map[string]any{},
	}

	parser := ParserFor("otlp")
	result, err := parser.GetDefaultConfig(logr.Discard(), config)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	tlsMap := resultMap["tls"].(map[string]any)

	_, hasMinVersion := tlsMap["min_version"]
	_, hasCiphers := tlsMap["cipher_suites"]
	assert.False(t, hasMinVersion, "should not set min_version when no TLS profile is provided")
	assert.False(t, hasCiphers, "should not set cipher_suites when no TLS profile is provided")
}

func TestExporterParserTLS13NoCiphers(t *testing.T) {
	tlsProfile := components.NewStaticTLSProfile(tls.VersionTLS13, nil)

	config := map[string]any{
		"endpoint": "tempo.example.com:4317",
		"tls":      map[string]any{},
	}

	parser := ParserFor("otlp")
	result, err := parser.GetDefaultConfig(logr.Discard(), config, components.WithTLSProfile(tlsProfile))
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	tlsMap := resultMap["tls"].(map[string]any)

	assert.Equal(t, "1.3", tlsMap["min_version"])
	_, hasCiphers := tlsMap["cipher_suites"]
	assert.False(t, hasCiphers, "TLS 1.3 should not inject cipher_suites")
}
