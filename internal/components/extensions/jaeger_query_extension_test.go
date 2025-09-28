// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestJaegerQueryExtensionParser(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	assert.Equal(t, "jaeger_query", genericBuilder.ParserType())
	assert.Equal(t, "__jaeger_query", genericBuilder.ParserName())

	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		config   interface{}
		expected []corev1.ServicePort
	}{
		{
			name:   "default http only",
			config: defaultCfg,
			expected: []corev1.ServicePort{{
				Name: "jaeger-query", Port: 16686, TargetPort: intstr.FromInt32(16686),
			}},
		},
		{
			name:   "grpc only configured",
			config: map[string]interface{}{"grpc": map[string]interface{}{"endpoint": "0.0.0.0:16685"}},
			expected: []corev1.ServicePort{
				{Name: "jaeger-query", Port: 16686, TargetPort: intstr.FromInt32(16686)},
				{Name: "jq-grpc", Port: 16685, TargetPort: intstr.FromInt32(16685)},
			},
		},
		{
			name: "http and grpc configured (non-default ports)",
			config: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "0.0.0.0:17686"},
				"grpc": map[string]interface{}{"endpoint": "0.0.0.0:17685"},
			},
			expected: []corev1.ServicePort{
				{Name: "jaeger-query", Port: 17686, TargetPort: intstr.FromInt32(17686)},
				{Name: "jq-grpc", Port: 17685, TargetPort: intstr.FromInt32(17685)},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var cfg = tc.config
			ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", cfg)
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, ports)
		})
	}
}

func TestJaegerQueryExtensionParser_GetDefaultConfig(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	tests := []struct {
		name   string
		config interface{}
		want   interface{}
	}{
		{
			name:   "http: preserves provided endpoint",
			config: map[string]interface{}{"http": map[string]interface{}{"endpoint": "127.0.0.0:17686"}},
			want:   map[string]interface{}{"http": map[string]interface{}{"endpoint": "127.0.0.0:17686"}},
		},
		{
			name:   "http: defaults host when missing",
			config: map[string]interface{}{"http": map[string]interface{}{"endpoint": ":17686"}},
			want: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "0.0.0.0:17686"},
			},
		},
		{
			name: "http: defaults when missing",
			want: map[string]interface{}{"http": map[string]interface{}{"endpoint": "0.0.0.0:16686"}},
		},
		{
			name:   "grpc: preserves provided endpoint; http defaults",
			config: map[string]interface{}{"grpc": map[string]interface{}{"endpoint": "127.0.0.0:17685"}},
			want: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "0.0.0.0:16686"},
				"grpc": map[string]interface{}{"endpoint": "127.0.0.0:17685"},
			},
		},
		{
			name:   "grpc: defaults host when missing",
			config: map[string]interface{}{"grpc": map[string]interface{}{"endpoint": ":17685"}},
			want: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "0.0.0.0:16686"},
				"grpc": map[string]interface{}{"endpoint": "0.0.0.0:17685"},
			},
		},
		{
			name:   "grpc: empty section removed",
			config: map[string]interface{}{"grpc": map[string]interface{}{}},
			want:   map[string]interface{}{"http": map[string]interface{}{"endpoint": "0.0.0.0:16686"}},
		},
		{
			name: "http+grpc: preserves provided endpoints",
			config: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "127.0.0.1:17686"},
				"grpc": map[string]interface{}{"endpoint": "127.0.0.1:17685"},
			},
			want: map[string]interface{}{
				"http": map[string]interface{}{"endpoint": "127.0.0.1:17686"},
				"grpc": map[string]interface{}{"endpoint": "127.0.0.1:17685"},
			},
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
