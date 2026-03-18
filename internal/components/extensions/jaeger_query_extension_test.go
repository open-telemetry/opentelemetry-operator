// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

func TestJaegerQueryExtensionParser(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	assert.Equal(t, "jaeger_query", genericBuilder.ParserType())
	assert.Equal(t, "__jaeger_query", genericBuilder.ParserName())

	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), nil)
	require.NoError(t, err)

	ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", defaultCfg)
	require.NoError(t, err)
	assert.Equal(t, []corev1.ServicePort{{
		Name:       "jaeger-query",
		Port:       16686,
		TargetPort: intstr.FromInt32(16686),
	}}, ports)
}

func TestJaegerQueryExtensionParser_config(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	tests := []struct {
		name   string
		config any
		want   any
	}{
		{
			name:   "valid config",
			config: map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}},
			want:   map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}},
		},
		{
			name: "missing config",
			want: map[string]any{"http": map[string]any{"endpoint": "0.0.0.0:16686"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, errCfg := genericBuilder.GetDefaultConfig(logr.Discard(), test.config)
			assert.Equal(t, test.want, cfg)
			require.NoError(t, errCfg)
		})
	}
}

func TestJaegerQueryExtensionTLSProfile(t *testing.T) {
	tests := []struct {
		name       string
		config     any
		tlsProfile components.TLSProfile
		want       map[string]any
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name: "TLS profile injected when tls block exists",
			config: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls":      map[string]any{},
				},
			},
			tlsProfile: components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}),
			want: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"min_version":   "1.2",
						"cipher_suites": []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "TLS profile not injected when tls block is absent",
			config: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
				},
			},
			tlsProfile: components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}),
			want: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "TLS profile does not override existing min_version",
			config: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"min_version": "1.3",
					},
				},
			},
			tlsProfile: components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}),
			want: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"min_version":   "1.3",
						"cipher_suites": []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "TLS profile does not override existing cipher_suites",
			config: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"cipher_suites": []string{"TLS_AES_256_GCM_SHA384"},
					},
				},
			},
			tlsProfile: components.NewStaticTLSProfile(tls.VersionTLS12, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}),
			want: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"min_version":   "1.2",
						"cipher_suites": []string{"TLS_AES_256_GCM_SHA384"},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "TLS 1.3 profile does not inject cipher suites",
			config: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls":      map[string]any{},
				},
			},
			tlsProfile: components.NewStaticTLSProfile(tls.VersionTLS13, []uint16{tls.TLS_AES_128_GCM_SHA256}),
			want: map[string]any{
				"http": map[string]any{
					"endpoint": "127.0.0.1:16686",
					"tls": map[string]any{
						"min_version": "1.3",
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := ParserFor("jaeger_query")
			got, err := parser.GetDefaultConfig(logr.Discard(), tt.config, components.WithTLSProfile(tt.tlsProfile))
			if !tt.wantErr(t, err, fmt.Sprintf("GetDefaultConfig(%v)", tt.config)) {
				return
			}

			gotMap, ok := got.(map[string]any)
			if ok {
				assert.Equalf(t, tt.want, gotMap, "GetDefaultConfig(%v)", tt.config)
			} else if tt.want != nil {
				t.Errorf("Expected map[string]interface{}, got %T", got)
			}
		})
	}
}
