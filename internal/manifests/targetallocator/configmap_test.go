// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfg "go.opentelemetry.io/collector/featuregate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func TestDesiredConfigMap(t *testing.T) {
	expectedLabels := map[string]string{
		"app.kubernetes.io/name":       "my-instance-targetallocator",
		"app.kubernetes.io/component":  "opentelemetry-targetallocator",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "default.my-instance",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.47.0",
	}
	collector := collectorInstance()
	targetAllocator := targetAllocatorInstance()
	cfg := config.New()
	params := Params{
		Collector:       collector,
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logr.Discard(),
	}

	t.Run("should return expected target allocator config map", func(t *testing.T) {
		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
filter_strategy: relabel-config
`,
		}

		actual, err := ConfigMap(params)
		require.NoError(t, err)

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData[targetAllocatorFilename], actual.Data[targetAllocatorFilename])

	})
	t.Run("should return target allocator config map without collector", func(t *testing.T) {
		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector: null
filter_strategy: relabel-config
`,
		}
		targetAllocator = targetAllocatorInstance()
		targetAllocator.Spec.ScrapeConfigs = []v1beta1.AnyConfig{}
		params.TargetAllocator = targetAllocator
		testParams := Params{
			Collector:       nil,
			TargetAllocator: targetAllocator,
		}
		actual, err := ConfigMap(testParams)
		require.NoError(t, err)
		params.Collector = collector

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData[targetAllocatorFilename], actual.Data[targetAllocatorFilename])

	})
	t.Run("should return target allocator config map without scrape configs", func(t *testing.T) {
		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
filter_strategy: relabel-config
`,
		}
		targetAllocator = targetAllocatorInstance()
		targetAllocator.Spec.ScrapeConfigs = []v1beta1.AnyConfig{}
		params.TargetAllocator = targetAllocator
		collectorWithoutPrometheusReceiver := collectorInstance()
		collectorWithoutPrometheusReceiver.Spec.Config.Receivers.Object["prometheus"] = map[string]any{
			"config": map[string]any{
				"scrape_configs": []any{},
			},
		}
		testParams := Params{
			Collector:       collectorWithoutPrometheusReceiver,
			TargetAllocator: targetAllocator,
		}
		actual, err := ConfigMap(testParams)
		require.NoError(t, err)
		params.Collector = collector

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData[targetAllocatorFilename], actual.Data[targetAllocatorFilename])

	})
	t.Run("should return expected target allocator config map with label selectors", func(t *testing.T) {
		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  global:
    scrape_interval: 30s
    scrape_protocols:
    - PrometheusProto
    - OpenMetricsText1.0.0
    - OpenMetricsText0.0.1
    - PrometheusText0.0.4
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
filter_strategy: relabel-config
prometheus_cr:
  enabled: true
  pod_monitor_selector:
    matchlabels:
      release: my-instance
    matchexpressions: []
  probe_selector:
    matchlabels:
      release: my-instance
    matchexpressions: []
  scrape_config_selector:
    matchlabels:
      release: my-instance
    matchexpressions: []
  service_monitor_selector:
    matchlabels:
      release: my-instance
    matchexpressions: []
`,
		}
		targetAllocator = targetAllocatorInstance()
		targetAllocator.Spec.PrometheusCR.Enabled = true
		targetAllocator.Spec.PrometheusCR.PodMonitorSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"release": "my-instance",
			},
		}
		targetAllocator.Spec.PrometheusCR.ServiceMonitorSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"release": "my-instance",
			}}
		targetAllocator.Spec.PrometheusCR.ScrapeConfigSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"release": "my-instance",
			}}
		targetAllocator.Spec.PrometheusCR.ProbeSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"release": "my-instance",
			}}
		targetAllocator.Spec.GlobalConfig = v1beta1.AnyConfig{
			Object: map[string]interface{}{
				"scrape_interval":  "30s",
				"scrape_protocols": []string{"PrometheusProto", "OpenMetricsText1.0.0", "OpenMetricsText0.0.1", "PrometheusText0.0.4"},
			},
		}
		params.TargetAllocator = targetAllocator
		actual, err := ConfigMap(params)
		assert.NoError(t, err)
		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})
	t.Run("should return expected target allocator config map with scrape interval set", func(t *testing.T) {
		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
filter_strategy: relabel-config
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  probe_selector: null
  scrape_config_selector: null
  scrape_interval: 30s
  service_monitor_selector: null
`,
		}

		targetAllocator = targetAllocatorInstance()
		targetAllocator.Spec.PrometheusCR.Enabled = true
		targetAllocator.Spec.PrometheusCR.ScrapeInterval = &metav1.Duration{Duration: time.Second * 30}
		params.TargetAllocator = targetAllocator
		actual, err := ConfigMap(params)
		assert.NoError(t, err)

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)

	})

	t.Run("should return expected target allocator config map with HTTPS configuration", func(t *testing.T) {
		expectedLabels["app.kubernetes.io/component"] = "opentelemetry-targetallocator"
		expectedLabels["app.kubernetes.io/name"] = "my-instance-targetallocator"

		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
		require.NoError(t, err)

		testParams := Params{
			Collector:       collector,
			TargetAllocator: targetAllocator,
			Config:          cfg,
		}

		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
filter_strategy: relabel-config
https:
  ca_file_path: /tls/ca.crt
  enabled: true
  listen_addr: :8443
  tls_cert_file_path: /tls/tls.crt
  tls_key_file_path: /tls/tls.key
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  probe_selector: null
  scrape_config_selector: null
  scrape_interval: 30s
  service_monitor_selector: null
`,
		}

		actual, err := ConfigMap(testParams)
		assert.NoError(t, err)

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)
	})

	t.Run("should return expected target allocator config map allocation fallback strategy", func(t *testing.T) {
		expectedLabels["app.kubernetes.io/component"] = "opentelemetry-targetallocator"
		expectedLabels["app.kubernetes.io/name"] = "my-instance-targetallocator"

		cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

		flgs := featuregate.Flags(colfg.GlobalRegistry())
		err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.fallbackstrategy"})
		require.NoError(t, err)

		testParams := Params{
			Collector:       collector,
			TargetAllocator: targetAllocator,
			Config:          cfg,
		}

		expectedData := map[string]string{
			targetAllocatorFilename: `allocation_fallback_strategy: consistent-hashing
allocation_strategy: consistent-hashing
collector_selector:
  matchlabels:
    app.kubernetes.io/component: opentelemetry-collector
    app.kubernetes.io/instance: default.my-instance
    app.kubernetes.io/managed-by: opentelemetry-operator
    app.kubernetes.io/part-of: opentelemetry
  matchexpressions: []
config:
  scrape_configs:
  - job_name: otel-collector
    scrape_interval: 10s
    static_configs:
    - targets:
      - 0.0.0.0:8888
      - 0.0.0.0:9999
filter_strategy: relabel-config
https:
  ca_file_path: /tls/ca.crt
  enabled: true
  listen_addr: :8443
  tls_cert_file_path: /tls/tls.crt
  tls_key_file_path: /tls/tls.key
prometheus_cr:
  enabled: true
  pod_monitor_selector: null
  probe_selector: null
  scrape_config_selector: null
  scrape_interval: 30s
  service_monitor_selector: null
`,
		}

		actual, err := ConfigMap(testParams)
		assert.NoError(t, err)

		assert.Equal(t, "my-instance-targetallocator", actual.Name)
		assert.Equal(t, expectedLabels, actual.Labels)
		assert.Equal(t, expectedData, actual.Data)
	})
}

func TestGetScrapeConfigsFromOtelConfig(t *testing.T) {
	testCases := []struct {
		name    string
		input   v1beta1.Config
		want    []v1beta1.AnyConfig
		wantErr error
	}{
		{
			name: "empty scrape configs list",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{
								"scrape_configs": []any{},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{},
		},
		{
			name: "no scrape configs key",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{},
						},
					},
				},
			},
			wantErr: fmt.Errorf("no scrape_configs available as part of the configuration"),
		},
		{
			name: "one scrape config",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{
								"scrape_configs": []any{
									map[string]any{
										"job": "somejob",
									},
								},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{
				{Object: map[string]interface{}{"job": "somejob"}},
			},
		},
		{
			name: "regex substitution",
			input: v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"prometheus": map[string]any{
							"config": map[string]any{
								"scrape_configs": []any{
									map[string]any{
										"job": "somejob",
										"metric_relabel_configs": []map[string]any{
											{
												"action":      "labelmap",
												"regex":       "label_(.+)",
												"replacement": "$$1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{
				{Object: map[string]interface{}{
					"job": "somejob",
					"metric_relabel_configs": []any{
						map[any]any{
							"action":      "labelmap",
							"regex":       "label_(.+)",
							"replacement": "$1",
						},
					},
				}},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			configStr, err := testCase.input.Yaml()
			require.NoError(t, err)
			actual, err := getScrapeConfigsFromOtelConfig(configStr)
			assert.Equal(t, testCase.wantErr, err)
			assert.Equal(t, testCase.want, actual)
		})
	}
}

func TestGetGlobalConfigFromOtelConfig(t *testing.T) {
	type args struct {
		otelConfig v1beta1.Config
	}
	tests := []struct {
		name    string
		args    args
		want    v1beta1.AnyConfig
		wantErr error
	}{
		{
			name: "Valid Global Config",
			args: args{
				otelConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]interface{}{
								"config": map[string]interface{}{
									"global": map[string]interface{}{
										"scrape_interval":  "15s",
										"scrape_protocols": []string{"PrometheusProto", "OpenMetricsText1.0.0", "OpenMetricsText0.0.1", "PrometheusText0.0.4"},
									},
								},
							},
						},
					},
				},
			},
			want: v1beta1.AnyConfig{
				Object: map[string]interface{}{
					"scrape_interval":  "15s",
					"scrape_protocols": []string{"PrometheusProto", "OpenMetricsText1.0.0", "OpenMetricsText0.0.1", "PrometheusText0.0.4"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Invalid Global Config - Missing Global",
			args: args{
				otelConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]interface{}{
								"config": map[string]interface{}{},
							},
						},
					},
				},
			},
			want:    v1beta1.AnyConfig{},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGlobalConfigFromOtelConfig(tt.args.otelConfig)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetScrapeConfigs(t *testing.T) {
	type args struct {
		taScrapeConfigs []v1beta1.AnyConfig
		collectorConfig v1beta1.Config
	}
	testCases := []struct {
		name    string
		args    args
		want    []v1beta1.AnyConfig
		wantErr error
	}{
		{
			name: "no scrape configs",
			args: args{
				taScrapeConfigs: []v1beta1.AnyConfig{},
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]any{
								"config": map[string]any{
									"scrape_configs": []any{},
								},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{},
		},
		{
			name: "scrape configs in both ta and collector",
			args: args{
				taScrapeConfigs: []v1beta1.AnyConfig{
					{
						Object: map[string]any{
							"job": "ta",
						},
					},
				},
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]any{
								"config": map[string]any{
									"scrape_configs": []any{
										map[string]any{
											"job": "collector",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []v1beta1.AnyConfig{
				{Object: map[string]any{"job": "ta"}},
				{Object: map[string]any{"job": "collector"}},
			},
		},
		{
			name: "no scrape configs key",
			args: args{
				taScrapeConfigs: []v1beta1.AnyConfig{},
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]any{
								"config": map[string]any{},
							},
						},
					},
				},
			},
			wantErr: fmt.Errorf("no scrape_configs available as part of the configuration"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := getScrapeConfigs(testCase.args.taScrapeConfigs, testCase.args.collectorConfig)
			assert.Equal(t, testCase.wantErr, err)
			assert.Equal(t, testCase.want, actual)
		})
	}
}

func TestGetGlobalConfig(t *testing.T) {
	type args struct {
		taGlobalConfig  v1beta1.AnyConfig
		collectorConfig v1beta1.Config
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr error
	}{
		{
			name: "Valid Global Config in both TA and Collector, TA wins",
			args: args{
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]interface{}{
								"config": map[string]interface{}{
									"global": map[string]interface{}{
										"scrape_interval": "15s",
									},
								},
							},
						},
					},
				},
				taGlobalConfig: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"scrape_protocols": []string{"PrometheusProto"},
					},
				},
			},
			want: map[string]interface{}{
				"scrape_protocols": []string{"PrometheusProto"},
			},
		},
		{
			name: "Valid Global Config in TA, not in Collector",
			args: args{
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]interface{}{
								"config": map[string]interface{}{},
							},
						},
					},
				},
				taGlobalConfig: v1beta1.AnyConfig{
					Object: map[string]interface{}{
						"scrape_protocols": []string{"PrometheusProto"},
					},
				},
			},
			want: map[string]interface{}{
				"scrape_protocols": []string{"PrometheusProto"},
			},
		},
		{
			name: "Valid Global Config in Collector, not in TA",
			args: args{
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": map[string]interface{}{
								"config": map[string]interface{}{
									"global": map[string]interface{}{
										"scrape_interval": "15s",
									},
								},
							},
						},
					},
				},
				taGlobalConfig: v1beta1.AnyConfig{},
			},
			want: map[string]interface{}{
				"scrape_interval": "15s",
			},
		},
		{
			name: "Invalid Global Config in Collector, not in TA",
			args: args{
				collectorConfig: v1beta1.Config{
					Receivers: v1beta1.AnyConfig{
						Object: map[string]interface{}{
							"prometheus": "invalid_value",
						},
					},
				},
				taGlobalConfig: v1beta1.AnyConfig{},
			},
			wantErr: &mapstructure.Error{Errors: []string{"'prometheus' expected a map, got 'string'"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGlobalConfig(tt.args.taGlobalConfig, tt.args.collectorConfig)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
