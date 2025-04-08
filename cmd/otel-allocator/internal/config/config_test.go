// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoad(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "file sd load",
			args: args{
				file: "./testdata/config_test.yaml",
			},
			want: Config{
				AllocationStrategy: DefaultAllocationStrategy,
				CollectorNamespace: "default",
				CollectorSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/instance":   "default.test",
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				FilterStrategy: DefaultFilterStrategy,
				PrometheusCR: PrometheusCRConfig{
					Enabled:                         true,
					ScrapeInterval:                  model.Duration(time.Second * 60),
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
					PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
					ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
					ProbeNamespaceSelector:          &metav1.LabelSelector{},
				},
				CollectorNotReadyGracePeriod: 30 * time.Second,
				HTTPS: HTTPSServerConfig{
					Enabled:         true,
					ListenAddr:      ":8443",
					CAFilePath:      "/path/to/ca.pem",
					TLSCertFilePath: "/path/to/cert.pem",
					TLSKeyFilePath:  "/path/to/key.pem",
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    DefaultScrapeProtocols,
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					Runtime: promconfig.DefaultRuntimeConfig,
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:           "prometheus",
							EnableCompression: true,
							HonorTimestamps:   true,
							ScrapeInterval:    model.Duration(60 * time.Second),
							ScrapeProtocols:   DefaultScrapeProtocols,
							ScrapeTimeout:     model.Duration(10 * time.Second),
							MetricsPath:       "/metrics",
							Scheme:            "http",
							HTTPClientConfig: commonconfig.HTTPClientConfig{
								FollowRedirects: true,
								EnableHTTP2:     true,
							},
							ServiceDiscoveryConfigs: []discovery.Config{
								&file.SDConfig{
									Files:           []string{"./file_sd_test.json"},
									RefreshInterval: model.Duration(5 * time.Minute),
								},
								discovery.StaticConfig{
									{
										Targets: []model.LabelSet{
											{model.AddressLabel: "prom.domain:9001"},
											{model.AddressLabel: "prom.domain:9002"},
											{model.AddressLabel: "prom.domain:9003"},
										},
										Labels: model.LabelSet{
											"my": "label",
										},
										Source: "0",
									},
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no config",
			args: args{
				file: "./testdata/no_config.yaml",
			},
			want:    CreateDefaultConfig(),
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector",
			args: args{
				file: "./testdata/pod_service_selector_test.yaml",
			},
			want: Config{
				AllocationStrategy: DefaultAllocationStrategy,
				CollectorNamespace: "default",
				CollectorSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/instance":   "default.test",
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				FilterStrategy: DefaultFilterStrategy,
				PrometheusCR: PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "test",
						},
					},
					ServiceMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "test",
						},
					},
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
					PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
					ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
					ProbeNamespaceSelector:          &metav1.LabelSelector{},
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    DefaultScrapeProtocols,
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					Runtime: promconfig.DefaultRuntimeConfig,
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:           "prometheus",
							EnableCompression: true,
							HonorTimestamps:   true,
							ScrapeInterval:    model.Duration(60 * time.Second),
							ScrapeProtocols:   DefaultScrapeProtocols,
							ScrapeTimeout:     model.Duration(10 * time.Second),
							MetricsPath:       "/metrics",
							Scheme:            "http",
							HTTPClientConfig: commonconfig.HTTPClientConfig{
								FollowRedirects: true,
								EnableHTTP2:     true,
							},
							ServiceDiscoveryConfigs: []discovery.Config{
								discovery.StaticConfig{
									{
										Targets: []model.LabelSet{
											{model.AddressLabel: "prom.domain:9001"},
											{model.AddressLabel: "prom.domain:9002"},
											{model.AddressLabel: "prom.domain:9003"},
										},
										Labels: model.LabelSet{
											"my": "label",
										},
										Source: "0",
									},
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with camelcase",
			args: args{
				file: "./testdata/pod_service_selector_camelcase_test.yaml",
			},
			want: Config{
				AllocationStrategy: DefaultAllocationStrategy,
				CollectorNamespace: "default",
				CollectorSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app.kubernetes.io/instance":   "default.test",
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				FilterStrategy: DefaultFilterStrategy,
				PrometheusCR: PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "test",
						},
					},
					ServiceMonitorSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"release": "test",
						},
					},
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
					PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
					ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
					ProbeNamespaceSelector:          &metav1.LabelSelector{},
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    DefaultScrapeProtocols,
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					Runtime: promconfig.DefaultRuntimeConfig,
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:           "prometheus",
							EnableCompression: true,
							HonorTimestamps:   true,
							ScrapeInterval:    model.Duration(60 * time.Second),
							ScrapeProtocols:   DefaultScrapeProtocols,
							ScrapeTimeout:     model.Duration(10 * time.Second),
							MetricsPath:       "/metrics",
							Scheme:            "http",
							HTTPClientConfig: commonconfig.HTTPClientConfig{
								FollowRedirects: true,
								EnableHTTP2:     true,
							},
							ServiceDiscoveryConfigs: []discovery.Config{
								discovery.StaticConfig{
									{
										Targets: []model.LabelSet{
											{model.AddressLabel: "prom.domain:9001"},
											{model.AddressLabel: "prom.domain:9002"},
											{model.AddressLabel: "prom.domain:9003"},
										},
										Labels: model.LabelSet{
											"my": "label",
										},
										Source: "0",
									},
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with matchexpressions",
			args: args{
				file: "./testdata/pod_service_selector_expressions_test.yaml",
			},
			want: Config{
				AllocationStrategy: DefaultAllocationStrategy,
				CollectorNamespace: "default",
				CollectorSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app.kubernetes.io/instance",
							Operator: metav1.LabelSelectorOpIn,
							Values: []string{
								"default.test",
							},
						},
						{
							Key:      "app.kubernetes.io/managed-by",
							Operator: metav1.LabelSelectorOpIn,
							Values: []string{
								"opentelemetry-operator",
							},
						},
					},
				},
				FilterStrategy: DefaultFilterStrategy,
				PrometheusCR: PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "release",
								Operator: metav1.LabelSelectorOpIn,
								Values: []string{
									"test",
								},
							},
						},
					},
					ServiceMonitorSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "release",
								Operator: metav1.LabelSelectorOpIn,
								Values: []string{
									"test",
								},
							},
						},
					},
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
					PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
					ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
					ProbeNamespaceSelector:          &metav1.LabelSelector{},
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    DefaultScrapeProtocols,
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					Runtime: promconfig.DefaultRuntimeConfig,
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:           "prometheus",
							EnableCompression: true,
							HonorTimestamps:   true,
							ScrapeInterval:    model.Duration(60 * time.Second),
							ScrapeProtocols:   DefaultScrapeProtocols,
							ScrapeTimeout:     model.Duration(10 * time.Second),
							MetricsPath:       "/metrics",
							Scheme:            "http",
							HTTPClientConfig: commonconfig.HTTPClientConfig{
								FollowRedirects: true,
								EnableHTTP2:     true,
							},
							ServiceDiscoveryConfigs: []discovery.Config{
								discovery.StaticConfig{
									{
										Targets: []model.LabelSet{
											{model.AddressLabel: "prom.domain:9001"},
											{model.AddressLabel: "prom.domain:9002"},
											{model.AddressLabel: "prom.domain:9003"},
										},
										Labels: model.LabelSet{
											"my": "label",
										},
										Source: "0",
									},
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with camelcase matchexpressions",
			args: args{
				file: "./testdata/pod_service_selector_camelcase_expressions_test.yaml",
			},
			want: Config{
				AllocationStrategy: DefaultAllocationStrategy,
				CollectorNamespace: "default",
				CollectorSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app.kubernetes.io/instance",
							Operator: metav1.LabelSelectorOpIn,
							Values: []string{
								"default.test",
							},
						},
						{
							Key:      "app.kubernetes.io/managed-by",
							Operator: metav1.LabelSelectorOpIn,
							Values: []string{
								"opentelemetry-operator",
							},
						},
					},
				},
				FilterStrategy: DefaultFilterStrategy,
				PrometheusCR: PrometheusCRConfig{
					PodMonitorSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "release",
								Operator: metav1.LabelSelectorOpIn,
								Values: []string{
									"test",
								},
							},
						},
					},
					ServiceMonitorSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "release",
								Operator: metav1.LabelSelectorOpIn,
								Values: []string{
									"test",
								},
							},
						},
					},
					ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
					PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
					ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
					ProbeNamespaceSelector:          &metav1.LabelSelector{},
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    DefaultScrapeProtocols,
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					Runtime: promconfig.DefaultRuntimeConfig,
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:           "prometheus",
							EnableCompression: true,
							HonorTimestamps:   true,
							ScrapeInterval:    model.Duration(60 * time.Second),
							ScrapeProtocols:   DefaultScrapeProtocols,
							ScrapeTimeout:     model.Duration(10 * time.Second),
							MetricsPath:       "/metrics",
							Scheme:            "http",
							HTTPClientConfig: commonconfig.HTTPClientConfig{
								FollowRedirects: true,
								EnableHTTP2:     true,
							},
							ServiceDiscoveryConfigs: []discovery.Config{
								discovery.StaticConfig{
									{
										Targets: []model.LabelSet{
											{model.AddressLabel: "prom.domain:9001"},
											{model.AddressLabel: "prom.domain:9002"},
											{model.AddressLabel: "prom.domain:9003"},
										},
										Labels: model.LabelSet{
											"my": "label",
										},
										Source: "0",
									},
								},
							},
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateDefaultConfig()
			err := LoadFromFile(tt.args.file, &got)
			if !tt.wantErr(t, err, fmt.Sprintf("Load(%v)", tt.args.file)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Load(%v)", tt.args.file)
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	current := os.Getenv("OTELCOL_NAMESPACE")
	t.Cleanup(func() {
		err := os.Setenv("OTELCOL_NAMESPACE", current)
		assert.NoError(t, err)
	})
	namespace := "default"
	os.Setenv("OTELCOL_NAMESPACE", namespace)
	cfg := &Config{}
	err := LoadFromEnv(cfg)
	require.NoError(t, err)
	assert.Equal(t, namespace, cfg.CollectorNamespace)
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		fileConfig  Config
		expectedErr error
	}{
		{
			name:        "no namespace",
			fileConfig:  Config{PrometheusCR: PrometheusCRConfig{Enabled: true}},
			expectedErr: fmt.Errorf("collector namespace must be set"),
		},
		{
			name:        "promCR enabled, no Prometheus config",
			fileConfig:  Config{PromConfig: nil, PrometheusCR: PrometheusCRConfig{Enabled: true}, CollectorNamespace: "default"},
			expectedErr: nil,
		},
		{
			name:        "promCR disabled, no Prometheus config",
			fileConfig:  Config{PromConfig: nil},
			expectedErr: fmt.Errorf("at least one scrape config must be defined, or Prometheus CR watching must be enabled"),
		},
		{
			name:        "promCR disabled, Prometheus config present, no scrapeConfigs",
			fileConfig:  Config{PromConfig: &promconfig.Config{}},
			expectedErr: fmt.Errorf("at least one scrape config must be defined, or Prometheus CR watching must be enabled"),
		},
		{
			name: "promCR disabled, Prometheus config present, scrapeConfigs present",
			fileConfig: Config{
				PromConfig:         &promconfig.Config{ScrapeConfigs: []*promconfig.ScrapeConfig{{}}},
				CollectorNamespace: "default",
			},
			expectedErr: nil,
		},
		{
			name: "promCR enabled, Prometheus config present, scrapeConfigs present",
			fileConfig: Config{
				PromConfig:         &promconfig.Config{ScrapeConfigs: []*promconfig.ScrapeConfig{{}}},
				PrometheusCR:       PrometheusCRConfig{Enabled: true},
				CollectorNamespace: "default",
			},
			expectedErr: nil,
		},
		{
			name: "both allowNamespaces and denyNamespaces set",
			fileConfig: Config{
				PrometheusCR: PrometheusCRConfig{
					Enabled:         true,
					AllowNamespaces: []string{"ns1"},
					DenyNamespaces:  []string{"ns2"},
				},
				CollectorNamespace: "default",
			},
			expectedErr: fmt.Errorf("only one of allowNamespaces or denyNamespaces can be set"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(&tc.fileConfig)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestGetAllowDenyLists(t *testing.T) {
	testCases := []struct {
		name              string
		promCRConfig      PrometheusCRConfig
		expectedAllowList map[string]struct{}
		expectedDenyList  map[string]struct{}
	}{
		{
			name:              "no allow or deny namespaces",
			promCRConfig:      PrometheusCRConfig{Enabled: true},
			expectedAllowList: map[string]struct{}{v1.NamespaceAll: {}},
			expectedDenyList:  map[string]struct{}{},
		},
		{
			name:              "allow namespaces",
			promCRConfig:      PrometheusCRConfig{Enabled: true, AllowNamespaces: []string{"ns1"}},
			expectedAllowList: map[string]struct{}{"ns1": {}},
			expectedDenyList:  map[string]struct{}{},
		},
		{
			name:              "deny namespaces",
			promCRConfig:      PrometheusCRConfig{Enabled: true, DenyNamespaces: []string{"ns2"}},
			expectedAllowList: map[string]struct{}{v1.NamespaceAll: {}},
			expectedDenyList:  map[string]struct{}{"ns2": {}},
		},
		{
			name:              "both allow and deny namespaces",
			promCRConfig:      PrometheusCRConfig{Enabled: true, AllowNamespaces: []string{"ns1"}, DenyNamespaces: []string{"ns2"}},
			expectedAllowList: map[string]struct{}{"ns1": {}},
			expectedDenyList:  map[string]struct{}{"ns2": {}},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			allowList, denyList := tc.promCRConfig.GetAllowDenyLists()
			assert.Equal(t, tc.expectedAllowList, allowList)
			assert.Equal(t, tc.expectedDenyList, denyList)
		})
	}
}
