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

	ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", defaultCfg)
	require.NoError(t, err)
	assert.Equal(t, []corev1.ServicePort{{
		Name:       "jaeger-query",
		Port:       16686,
		TargetPort: intstr.FromInt32(16686),
	}}, ports)
}

func TestJaegerQueryExtensionParser_httpAndGrpc(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	// Config with both HTTP and gRPC endpoints
	cfg := map[string]any{
		"http": map[string]any{"endpoint": "0.0.0.0:16686"},
		"grpc": map[string]any{"endpoint": "0.0.0.0:16685"},
	}
	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), cfg)
	require.NoError(t, err)

	ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", defaultCfg)
	require.NoError(t, err)
	assert.Len(t, ports, 2)
	assert.Equal(t, corev1.ServicePort{
		Name:       "jaeger-query",
		Port:       16686,
		TargetPort: intstr.FromInt32(16686),
	}, ports[0])
	assert.Equal(t, corev1.ServicePort{
		Name:       "port-16685",
		Port:       16685,
		TargetPort: intstr.FromInt32(16685),
	}, ports[1])
}

func TestJaegerQueryExtensionParser_grpcOnly(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	// Config with only gRPC endpoint (HTTP will get default)
	cfg := map[string]any{
		"grpc": map[string]any{"endpoint": "0.0.0.0:16685"},
	}
	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), cfg)
	require.NoError(t, err)

	ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", defaultCfg)
	require.NoError(t, err)
	assert.Len(t, ports, 2)
	assert.Equal(t, corev1.ServicePort{
		Name:       "jaeger-query",
		Port:       16686,
		TargetPort: intstr.FromInt32(16686),
	}, ports[0])
	assert.Equal(t, corev1.ServicePort{
		Name:       "port-16685",
		Port:       16685,
		TargetPort: intstr.FromInt32(16685),
	}, ports[1])
}

func TestJaegerQueryExtensionParser_samePort(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	// Config where gRPC uses the same port as HTTP — should only return one port
	cfg := map[string]any{
		"http": map[string]any{"endpoint": "0.0.0.0:16686"},
		"grpc": map[string]any{"endpoint": "0.0.0.0:16686"},
	}
	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), cfg)
	require.NoError(t, err)

	ports, err := genericBuilder.Ports(logr.Discard(), "jaeger_query", defaultCfg)
	require.NoError(t, err)
	assert.Equal(t, []corev1.ServicePort{{
		Name:       "jaeger-query",
		Port:       16686,
		TargetPort: intstr.FromInt32(16686),
	}}, ports)
}

func TestJaegerQueryExtensionParser_invalidGrpcEndpoint(t *testing.T) {
	jaegerBuilder := NewJaegerQueryExtensionParserBuilder()
	genericBuilder, err := jaegerBuilder.Build()
	require.NoError(t, err)

	// Config with a malformed gRPC endpoint that cannot be parsed — should gracefully
	// return only the HTTP port without error
	cfg := map[string]any{
		"http": map[string]any{"endpoint": "0.0.0.0:16686"},
		"grpc": map[string]any{"endpoint": "invalid-no-port"},
	}
	defaultCfg, err := genericBuilder.GetDefaultConfig(logr.Discard(), cfg)
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
			name:   "valid http config",
			config: map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}},
			want:   map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}},
		},
		{
			name:   "valid http and grpc config",
			config: map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}, "grpc": map[string]any{"endpoint": "127.0.0.0:16685"}},
			want:   map[string]any{"http": map[string]any{"endpoint": "127.0.0.0:16686"}, "grpc": map[string]any{"endpoint": "127.0.0.0:16685"}},
		},
		{
			name: "grpc with missing host gets default",
			config: map[string]any{
				"http": map[string]any{"endpoint": "127.0.0.0:16686"},
				"grpc": map[string]any{"endpoint": ":16685"},
			},
			want: map[string]any{
				"http": map[string]any{"endpoint": "127.0.0.0:16686"},
				"grpc": map[string]any{"endpoint": "0.0.0.0:16685"},
			},
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
