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

func TestScraperParsers(t *testing.T) {
	for _, tt := range []struct {
		receiverName string
		parserName   string
		defaultPort  int
	}{
		{"kubeletstats", "__kubeletstats", 0},
		{"sshcheck", "__sshcheck", 0},
		{"cloudfoundry", "__cloudfoundry", 0},
		{"vcenter", "__vcenter", 0},
		{"oracledb", "__oracledb", 0},
		{"snmp", "__snmp", 0},
		{"googlecloudpubsub", "__googlecloudpubsub", 0},
		{"chrony", "__chrony", 0},
		{"jmx", "__jmx", 0},
		{"podman_stats", "__podman_stats", 0},
		{"pulsar", "__pulsar", 0},
		{"docker_stats", "__docker_stats", 0},
		{"aerospike", "__aerospike", 0},
		{"zookeeper", "__zookeeper", 0},
		{"prometheus_simple", "__prometheus_simple", 0},
		{"saphana", "__saphana", 0},
		{"riak", "__riak", 0},
		{"redis", "__redis", 0},
		{"rabbitmq", "__rabbitmq", 0},
		{"purefb", "__purefb", 0},
		{"postgresql", "__postgresql", 0},
		{"nsxt", "__nsxt", 0},
		{"nginx", "__nginx", 0},
		{"mysql", "__mysql", 0},
		{"memcached", "__memcached", 0},
		{"httpcheck", "__httpcheck", 0},
		{"haproxy", "__haproxy", 0},
		{"flinkmetrics", "__flinkmetrics", 0},
		{"couchdb", "__couchdb", 0},
	} {
		t.Run(tt.receiverName, func(t *testing.T) {
			t.Run("builds successfully", func(t *testing.T) {
				// test
				parser := receivers.ReceiverFor(tt.receiverName)

				// verify
				assert.Equal(t, tt.parserName, parser.ParserName())
			})

			t.Run("default is nothing", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				ports, err := parser.Ports(logger, tt.receiverName, map[string]any{})

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 0)
			})

			t.Run("always returns nothing", func(t *testing.T) {
				// prepare
				parser := receivers.ReceiverFor(tt.receiverName)

				// test
				ports, err := parser.Ports(logger, tt.receiverName, map[string]any{
					"endpoint": "0.0.0.0:65535",
				})

				// verify
				assert.NoError(t, err)
				assert.Len(t, ports, 0)
			})
		})
	}
}

func TestScraperParserTLSProfile(t *testing.T) {
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
				"endpoint": "localhost:6379",
				"tls":      map[string]any{},
			},
			expectMinVersion: "1.2",
			expectCiphers:    []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		},
		{
			name: "does not override existing min_version",
			config: map[string]any{
				"endpoint": "localhost:6379",
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
				"endpoint": "localhost:6379",
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
				"endpoint": "localhost:6379",
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
			parser := receivers.ReceiverFor("redis")

			var config any
			if tt.config != nil {
				config = tt.config
			}

			result, err := parser.GetDefaultConfig(logger, config, components.WithTLSProfile(tlsProfile))
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

func TestScraperParserNoTLSProfile(t *testing.T) {
	config := map[string]any{
		"endpoint": "localhost:6379",
		"tls":      map[string]any{},
	}

	parser := receivers.ReceiverFor("redis")
	result, err := parser.GetDefaultConfig(logger, config)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	tlsMap := resultMap["tls"].(map[string]any)

	_, hasMinVersion := tlsMap["min_version"]
	_, hasCiphers := tlsMap["cipher_suites"]
	assert.False(t, hasMinVersion, "should not set min_version when no TLS profile is provided")
	assert.False(t, hasCiphers, "should not set cipher_suites when no TLS profile is provided")
}

func TestScraperParserTLS13NoCiphers(t *testing.T) {
	tlsProfile := components.NewStaticTLSProfile(tls.VersionTLS13, nil)

	config := map[string]any{
		"endpoint": "localhost:6379",
		"tls":      map[string]any{},
	}

	parser := receivers.ReceiverFor("redis")
	result, err := parser.GetDefaultConfig(logger, config, components.WithTLSProfile(tlsProfile))
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	tlsMap := resultMap["tls"].(map[string]any)

	assert.Equal(t, "1.3", tlsMap["min_version"])
	_, hasCiphers := tlsMap["cipher_suites"]
	assert.False(t, hasCiphers, "TLS 1.3 should not inject cipher_suites")
}
