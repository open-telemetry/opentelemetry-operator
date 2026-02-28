// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func TestUpgrade0_145_0(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *v1beta1.OpenTelemetryCollector
		expectedConfig *v1beta1.OpenTelemetryCollector
	}{
		{
			name: "should rename otlp to otlp_grpc",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should rename otlphttp to otlp_http",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlphttp": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"metrics": {
									Receivers:  []string{"prometheus"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlphttp"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_http": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"metrics": {
									Receivers:  []string{"prometheus"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_http"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should rename named exporters otlp/production to otlp_grpc/production",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp/production": map[string]interface{}{
									"endpoint": "prod:4317",
								},
								"otlp/staging": map[string]interface{}{
									"endpoint": "staging:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp/production", "otlp/staging"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc/production": map[string]interface{}{
									"endpoint": "prod:4317",
								},
								"otlp_grpc/staging": map[string]interface{}{
									"endpoint": "staging:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc/production", "otlp_grpc/staging"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should handle mixed pipelines with both otlp and otlphttp",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
								"otlphttp": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
								"logging": map[string]interface{}{},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp", "otlphttp", "logging"},
								},
								"metrics": {
									Receivers:  []string{"prometheus"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlphttp"},
								},
								"logs": {
									Receivers:  []string{"filelog"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
								"otlp_http": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
								"logging": map[string]interface{}{},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc", "otlp_http", "logging"},
								},
								"metrics": {
									Receivers:  []string{"prometheus"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_http"},
								},
								"logs": {
									Receivers:  []string{"filelog"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should preserve complex TLS configuration",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp": map[string]interface{}{
									"endpoint": "localhost:4317",
									"tls": map[string]interface{}{
										"insecure":  false,
										"cert_file": "/certs/cert.pem",
										"key_file":  "/certs/key.pem",
										"ca_file":   "/certs/ca.pem",
									},
									"compression": "gzip",
									"timeout":     "30s",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
									"tls": map[string]interface{}{
										"insecure":  false,
										"cert_file": "/certs/cert.pem",
										"key_file":  "/certs/key.pem",
										"ca_file":   "/certs/ca.pem",
									},
									"compression": "gzip",
									"timeout":     "30s",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should not modify config with no exporters section",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: nil,
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: nil,
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{},
						},
					},
				},
			},
		},
		{
			name: "should not modify already migrated config",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
								"otlp_http": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc", "otlp_http"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
								"otlp_http": map[string]interface{}{
									"endpoint": "http://localhost:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_grpc", "otlp_http"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "should handle config with no service pipelines",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: nil,
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_grpc": map[string]interface{}{
									"endpoint": "localhost:4317",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: nil,
						},
					},
				},
			},
		},
		{
			name: "should rename named otlphttp exporters",
			initialConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlphttp/prod": map[string]interface{}{
									"endpoint": "http://prod:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlphttp/prod"},
								},
							},
						},
					},
				},
			},
			expectedConfig: &v1beta1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "OpenTelemetryCollector",
					APIVersion: "v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{
						Exporters: v1beta1.AnyConfig{
							Object: map[string]interface{}{
								"otlp_http/prod": map[string]interface{}{
									"endpoint": "http://prod:4318",
								},
							},
						},
						Service: v1beta1.Service{
							Pipelines: map[string]*v1beta1.Pipeline{
								"traces": {
									Receivers:  []string{"otlp"},
									Processors: []string{"batch"},
									Exporters:  []string{"otlp_http/prod"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := VersionUpgrade{
				Log: logger,
			}

			result, err := upgrade0_145_0(up, tt.initialConfig)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify exporters were renamed correctly
			assert.Equal(t, tt.expectedConfig.Spec.Config.Exporters.Object, result.Spec.Config.Exporters.Object)

			// Verify pipeline references were updated
			if tt.expectedConfig.Spec.Config.Service.Pipelines != nil {
				require.NotNil(t, result.Spec.Config.Service.Pipelines)
				for pipelineName, expectedPipeline := range tt.expectedConfig.Spec.Config.Service.Pipelines {
					resultPipeline, exists := result.Spec.Config.Service.Pipelines[pipelineName]
					require.True(t, exists, "pipeline %s should exist", pipelineName)
					assert.Equal(t, expectedPipeline.Exporters, resultPipeline.Exporters)
				}
			}
		})
	}
}
