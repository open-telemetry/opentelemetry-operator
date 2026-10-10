// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestLoadAgentCollectorConfigOpenShiftTLSOverride(t *testing.T) {
	loader := NewConfigLoader()

	base, err := loader.LoadCollectorConfig(AgentCollectorType, DistroProvider(""), v1alpha1.ClusterObservabilitySpec{})
	require.NoError(t, err)
	openshift, err := loader.LoadCollectorConfig(AgentCollectorType, OpenShift, v1alpha1.ClusterObservabilitySpec{})
	require.NoError(t, err)

	assert.Equal(t, true, requireMap(t, base.Receivers.Object, "kubelet_stats")["insecure_skip_verify"])

	kubeletstats := requireMap(t, openshift.Receivers.Object, "kubelet_stats")
	assert.Equal(t, "/etc/kubelet-serving-ca/ca-bundle.crt", kubeletstats["ca_file"])
	assert.Equal(t, false, kubeletstats["insecure_skip_verify"])
	assert.Contains(t, openshift.Receivers.Object, "host_metrics")
	assert.Contains(t, openshift.Receivers.Object, "file_log")
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         v1beta1.Config
		expectedErr string
	}{
		{
			name: "no receivers configured",
			cfg: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {
							Receivers: []string{"otlp"},
							Exporters: []string{"otlp"},
						},
					},
				},
			},
			expectedErr: "no receivers configured",
		},
		{
			name: "no exporters configured",
			cfg: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"metrics": {
							Receivers: []string{"otlp"},
							Exporters: []string{"otlp"},
						},
					},
				},
			},
			expectedErr: "no exporters configured",
		},
		{
			name: "no pipelines configured",
			cfg: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{},
				},
			},
			expectedErr: "no pipelines configured",
		},
		{
			name: "pipeline references non-existent receiver",
			cfg: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"traces": {
							Receivers: []string{"prometheus"},
							Exporters: []string{"otlp"},
						},
					},
				},
			},
			expectedErr: "pipeline traces references non-existent receiver prometheus",
		},
		{
			name: "pipeline references non-existent processor",
			cfg: v1beta1.Config{
				Receivers:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{"batch": map[string]any{}}},
				Exporters:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"traces": {
							Receivers:  []string{"otlp"},
							Processors: []string{"memory_limiter"},
							Exporters:  []string{"otlp"},
						},
					},
				},
			},
			expectedErr: "pipeline traces references non-existent processor memory_limiter",
		},
		{
			name: "pipeline references non-existent exporter",
			cfg: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Exporters: v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"traces": {
							Receivers: []string{"otlp"},
							Exporters: []string{"debug"},
						},
					},
				},
			},
			expectedErr: "pipeline traces references non-existent exporter debug",
		},
		{
			name: "valid pipeline configuration with nil pipeline skipped",
			cfg: v1beta1.Config{
				Receivers:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Processors: &v1beta1.AnyConfig{Object: map[string]any{"batch": map[string]any{}}},
				Exporters:  v1beta1.AnyConfig{Object: map[string]any{"otlp": map[string]any{}}},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"empty": nil,
						"traces": {
							Receivers:  []string{"otlp"},
							Processors: []string{"batch"},
							Exporters:  []string{"otlp"},
						},
					},
				},
			},
			expectedErr: "",
		},
	}

	loader := NewConfigLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.ValidateConfig(tt.cfg)
			if tt.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			}
		})
	}
}

func TestDetectDistroProvider(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.Config
		expected DistroProvider
	}{
		{
			name: "openshift routes available",
			cfg: config.Config{
				OpenShiftRoutesAvailability: openshift.RoutesAvailable,
			},
			expected: OpenShift,
		},
		{
			name: "openshift routes not available",
			cfg: config.Config{
				OpenShiftRoutesAvailability: openshift.RoutesNotAvailable,
			},
			expected: DistroProvider(""),
		},
	}

	loader := NewConfigLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, loader.DetectDistroProvider(tt.cfg))
		})
	}
}

func TestCompareConfigVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{
			name:     "identical versions",
			v1:       "a1b2c3",
			v2:       "a1b2c3",
			expected: false,
		},
		{
			name:     "different versions",
			v1:       "a1b2c3",
			v2:       "d4e5f6",
			expected: true,
		},
	}

	loader := NewConfigLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, loader.CompareConfigVersions(tt.v1, tt.v2))
		})
	}
}

func TestConfigVersions(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("GetConfigVersion valid collector type", func(t *testing.T) {
		version, err := loader.GetConfigVersion(AgentCollectorType, OpenShift)
		require.NoError(t, err)
		assert.NotEmpty(t, version)

		baseVersion, err := loader.GetConfigVersion(AgentCollectorType, DistroProvider(""))
		require.NoError(t, err)
		assert.NotEmpty(t, baseVersion)
		assert.NotEqual(t, version, baseVersion)
	})

	t.Run("GetConfigVersion non-existent base config", func(t *testing.T) {
		_, err := loader.GetConfigVersion(CollectorType("unknown"), DistroProvider(""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read base config for version")
	})

	t.Run("GetAllConfigVersions returns map of all supported variants", func(t *testing.T) {
		versions, err := loader.GetAllConfigVersions()
		require.NoError(t, err)
		assert.NotEmpty(t, versions)
		assert.Contains(t, versions, "agent-collector-openshift")
	})
}

func TestBuildExportersConfig(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("empty spec returns empty otlp_http config", func(t *testing.T) {
		exporters := loader.buildExportersConfig(v1alpha1.ClusterObservabilitySpec{})
		require.Contains(t, exporters, "otlp_http")
		otlpHTTP := requireMap(t, exporters, "otlp_http")
		assert.Empty(t, otlpHTTP)
	})

	t.Run("full exporter configuration mapped accurately", func(t *testing.T) {
		trueVal := true
		intVal := 1024
		consumersVal := 4

		spec := v1alpha1.ClusterObservabilitySpec{
			Exporter: v1alpha1.OTLPHTTPExporter{
				Endpoint:         "https://collector.example.com:4318",
				TracesEndpoint:   "https://traces.example.com",
				MetricsEndpoint:  "https://metrics.example.com",
				LogsEndpoint:     "https://logs.example.com",
				ProfilesEndpoint: "https://profiles.example.com",
				TLS: &v1alpha1.TLSConfig{
					CAFile:     "/etc/ssl/ca.crt",
					CertFile:   "/etc/ssl/cert.crt",
					KeyFile:    "/etc/ssl/key.pem",
					Insecure:   true,
					ServerName: "collector.example.com",
				},
				Timeout:         "30s",
				ReadBufferSize:  &intVal,
				WriteBufferSize: &intVal,
				Encoding:        "proto",
				Compression:     "gzip",
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
				SendingQueue: &v1alpha1.SendingQueueConfig{
					Enabled:      &trueVal,
					NumConsumers: &consumersVal,
					QueueSize:    &intVal,
				},
				RetryOnFailure: &v1alpha1.RetryConfig{
					Enabled:             &trueVal,
					InitialInterval:     "5s",
					RandomizationFactor: "0.1",
					Multiplier:          "1.5",
					MaxInterval:         "30s",
					MaxElapsedTime:      "5m",
				},
			},
		}

		exporters := loader.buildExportersConfig(spec)
		otlpHTTP := requireMap(t, exporters, "otlp_http")

		assert.Equal(t, "https://collector.example.com:4318", otlpHTTP["endpoint"])
		assert.Equal(t, "https://traces.example.com", otlpHTTP["traces_endpoint"])
		assert.Equal(t, "https://metrics.example.com", otlpHTTP["metrics_endpoint"])
		assert.Equal(t, "https://logs.example.com", otlpHTTP["logs_endpoint"])
		assert.Equal(t, "https://profiles.example.com", otlpHTTP["profiles_endpoint"])
		assert.Equal(t, "30s", otlpHTTP["timeout"])
		assert.Equal(t, 1024, otlpHTTP["read_buffer_size"])
		assert.Equal(t, 1024, otlpHTTP["write_buffer_size"])
		assert.Equal(t, "proto", otlpHTTP["encoding"])
		assert.Equal(t, "gzip", otlpHTTP["compression"])
		assert.Equal(t, map[string]string{"X-Custom-Header": "custom-value"}, otlpHTTP["headers"])

		tlsConfig := requireMap(t, otlpHTTP, "tls")
		assert.Equal(t, "/etc/ssl/ca.crt", tlsConfig["ca_file"])
		assert.Equal(t, "/etc/ssl/cert.crt", tlsConfig["cert_file"])
		assert.Equal(t, "/etc/ssl/key.pem", tlsConfig["key_file"])
		assert.Equal(t, true, tlsConfig["insecure"])
		assert.Equal(t, "collector.example.com", tlsConfig["server_name"])

		queueConfig := requireMap(t, otlpHTTP, "sending_queue")
		assert.Equal(t, true, queueConfig["enabled"])
		assert.Equal(t, 4, queueConfig["num_consumers"])
		assert.Equal(t, 1024, queueConfig["queue_size"])

		retryConfig := requireMap(t, otlpHTTP, "retry_on_failure")
		assert.Equal(t, true, retryConfig["enabled"])
		assert.Equal(t, "5s", retryConfig["initial_interval"])
		assert.Equal(t, "0.1", retryConfig["randomization_factor"])
		assert.Equal(t, "1.5", retryConfig["multiplier"])
		assert.Equal(t, "30s", retryConfig["max_interval"])
		assert.Equal(t, "5m", retryConfig["max_elapsed_time"])
	})
}

func TestBuildPipelinesWithExporters(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("AgentCollectorType pipelines configuration", func(t *testing.T) {
		pipelines := loader.buildPipelinesWithExporters(AgentCollectorType)
		assert.Contains(t, pipelines, "metrics")
		assert.Contains(t, pipelines, "logs")
		assert.Contains(t, pipelines, "traces")

		assert.Equal(t, []string{"otlp", "host_metrics", "kubelet_stats"}, pipelines["metrics"].Receivers)
		assert.Equal(t, []string{"resource_detection", "k8s_attributes", "batch"}, pipelines["metrics"].Processors)
		assert.Equal(t, []string{"otlp_http"}, pipelines["metrics"].Exporters)
	})

	t.Run("ClusterCollectorType pipelines configuration", func(t *testing.T) {
		pipelines := loader.buildPipelinesWithExporters(ClusterCollectorType)
		assert.Contains(t, pipelines, "metrics")
		assert.Contains(t, pipelines, "metrics/prometheus")
		assert.Contains(t, pipelines, "logs")

		assert.Equal(t, []string{"k8s_cluster"}, pipelines["metrics"].Receivers)
		assert.Equal(t, []string{"prometheus"}, pipelines["metrics/prometheus"].Receivers)
		assert.Equal(t, []string{"k8s_events"}, pipelines["logs"].Receivers)
		assert.Equal(t, []string{"otlp_http"}, pipelines["logs"].Exporters)
	})
}

func requireMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key]
	require.Truef(t, ok, "key %q not found", key)
	result, ok := value.(map[string]any)
	require.Truef(t, ok, "key %q has type %T", key, value)
	return result
}
