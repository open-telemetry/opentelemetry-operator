// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestLoadFromFile(t *testing.T) {
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
				file: filepath.Join("testdata", "config_test.yaml"),
			},
			want: Config{
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
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
						ScrapeProtocols:    promconfig.DefaultScrapeProtocols,
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
							ScrapeProtocols:   promconfig.DefaultScrapeProtocols,
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
			name: "file sd load with global",
			args: args{
				file: filepath.Join("testdata", "global_config_test.yaml"),
			},
			want: Config{
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
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
						ScrapeProtocols:    []promconfig.ScrapeProtocol{promconfig.PrometheusProto},
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
							ScrapeProtocols:   []promconfig.ScrapeProtocol{promconfig.PrometheusProto},
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
				file: filepath.Join("testdata", "no_config.yaml"),
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
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				HTTPS: HTTPSServerConfig{
					ListenAddr: ":8443",
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    promconfig.DefaultScrapeProtocols,
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
							ScrapeProtocols:   promconfig.DefaultScrapeProtocols,
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
				CollectorNotReadyGracePeriod: 30 * time.Second,
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with camelcase",
			args: args{
				file: filepath.Join("testdata", "pod_service_selector_camelcase_test.yaml"),
			},
			want: Config{
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				HTTPS: HTTPSServerConfig{
					ListenAddr: ":8443",
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    promconfig.DefaultScrapeProtocols,
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
							ScrapeProtocols:   promconfig.DefaultScrapeProtocols,
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
				CollectorNotReadyGracePeriod: 30 * time.Second,
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with matchexpressions",
			args: args{
				file: filepath.Join("testdata", "pod_service_selector_expressions_test.yaml"),
			},
			want: Config{
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				HTTPS: HTTPSServerConfig{
					ListenAddr: ":8443",
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    promconfig.DefaultScrapeProtocols,
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
							ScrapeProtocols:   promconfig.DefaultScrapeProtocols,
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
				CollectorNotReadyGracePeriod: 30 * time.Second,
			},
			wantErr: assert.NoError,
		},
		{
			name: "service monitor pod monitor selector with camelcase matchexpressions",
			args: args{
				file: filepath.Join("testdata", "pod_service_selector_camelcase_expressions_test.yaml"),
			},
			want: Config{
				ListenAddr:         DefaultListenAddr,
				KubeConfigFilePath: DefaultKubeConfigFilePath,
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
					ScrapeProtocols:                 defaultScrapeProtocolsCR,
					ScrapeInterval:                  DefaultCRScrapeInterval,
				},
				HTTPS: HTTPSServerConfig{
					ListenAddr: ":8443",
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeProtocols:    promconfig.DefaultScrapeProtocols,
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
							ScrapeProtocols:   promconfig.DefaultScrapeProtocols,
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
				CollectorNotReadyGracePeriod: 30 * time.Second,
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
	namespace := "default"
	t.Setenv("OTELCOL_NAMESPACE", namespace)
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

func TestConfigLoadPriority(t *testing.T) {
	// Helper function to create a dummy kube config for tests
	createDummyKubeConfig := func(t *testing.T, dir string) string {
		kubeConfigPath := filepath.Join(dir, "kube.config")
		kubeConfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.com
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: dummy-token
`
		err := os.WriteFile(kubeConfigPath, []byte(kubeConfigContent), 0600)
		require.NoError(t, err)
		return kubeConfigPath
	}

	t.Run("default values when nothing is set", func(t *testing.T) {
		// Setup: create empty config file and dummy kube config
		tempDir := t.TempDir()
		emptyConfigPath := filepath.Join(tempDir, "empty.yaml")
		err := os.WriteFile(emptyConfigPath, []byte("{}"), 0600)
		require.NoError(t, err)

		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		// Prepare args for Load function
		args := []string{
			"--" + configFilePathFlagName + "=" + emptyConfigPath,
			"--" + kubeConfigPathFlagName + "=" + kubeConfigPath,
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert defaults are used
		assert.Equal(t, DefaultListenAddr, config.ListenAddr)
		assert.Equal(t, kubeConfigPath, config.KubeConfigFilePath)
		assert.Equal(t, DefaultHttpsListenAddr, config.HTTPS.ListenAddr)
		assert.Equal(t, DefaultAllocationStrategy, config.AllocationStrategy)
		assert.Equal(t, DefaultFilterStrategy, config.FilterStrategy)
		assert.False(t, config.PrometheusCR.Enabled)
		assert.False(t, config.HTTPS.Enabled)
	})

	t.Run("command-line has priority over config file for boolean values", func(t *testing.T) {
		// Setup: create config file with values and dummy kube config
		tempDir := t.TempDir()
		configContent := `
prometheus_cr:
  enabled: false
https:
  enabled: false
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		// Prepare args for Load function that override config file
		args := []string{
			"--" + configFilePathFlagName + "=" + configPath,
			"--" + prometheusCREnabledFlagName + "=true",
			"--" + httpsEnabledFlagName + "=true",
			"--" + kubeConfigPathFlagName + "=" + kubeConfigPath,
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert CLI values override config file
		assert.True(t, config.PrometheusCR.Enabled, "CLI should override config file for prometheus CR enabled")
		assert.True(t, config.HTTPS.Enabled, "CLI should override config file for HTTPS enabled")
	})

	t.Run("command-line has priority over config file for string values", func(t *testing.T) {
		// Setup: create config file with values and dummy kube config
		tempDir := t.TempDir()
		configContent := `
listen_addr: ":9090"
https:
  listen_addr: ":9443"
  ca_file_path: "/config/ca.pem"
  tls_cert_file_path: "/config/cert.pem"
  tls_key_file_path: "/config/key.pem"
kube_config_file_path: "/config/kube.config"
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		// CLI values different from config file
		cliListenAddr := ":8888"
		cliHttpsListenAddr := ":8443"
		cliCAPath := "/cli/ca.pem"
		cliCertPath := "/cli/cert.pem"
		cliKeyPath := "/cli/key.pem"

		// Prepare args for Load function that override config file
		args := []string{
			"--" + configFilePathFlagName + "=" + configPath,
			"--" + listenAddrFlagName + "=" + cliListenAddr,
			"--" + listenAddrHttpsFlagName + "=" + cliHttpsListenAddr,
			"--" + httpsCAFilePathFlagName + "=" + cliCAPath,
			"--" + httpsTLSCertFilePathFlagName + "=" + cliCertPath,
			"--" + httpsTLSKeyFilePathFlagName + "=" + cliKeyPath,
			"--" + kubeConfigPathFlagName + "=" + kubeConfigPath,
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert CLI values override config file
		assert.Equal(t, cliListenAddr, config.ListenAddr, "CLI should override config file for listen address")
		assert.Equal(t, cliHttpsListenAddr, config.HTTPS.ListenAddr, "CLI should override config file for HTTPS listen address")
		assert.Equal(t, cliCAPath, config.HTTPS.CAFilePath, "CLI should override config file for CA file path")
		assert.Equal(t, cliCertPath, config.HTTPS.TLSCertFilePath, "CLI should override config file for TLS cert file path")
		assert.Equal(t, cliKeyPath, config.HTTPS.TLSKeyFilePath, "CLI should override config file for TLS key file path")
		assert.Equal(t, kubeConfigPath, config.KubeConfigFilePath, "CLI should override config file for kube config path")
	})

	t.Run("config file overrides defaults when CLI not specified", func(t *testing.T) {
		// Setup: create config file with values and dummy kube config
		tempDir := t.TempDir()
		configListenAddr := ":7070"
		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		configContent := `
collector_namespace: config-file-namespace
listen_addr: "` + configListenAddr + `"
prometheus_cr:
  enabled: true
https:
  enabled: true
  listen_addr: ":7443"
kube_config_file_path: "` + kubeConfigPath + `"
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		// Prepare args for Load function with only config file path
		args := []string{
			"--" + configFilePathFlagName + "=" + configPath,
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert config file values override defaults
		assert.Equal(t, configListenAddr, config.ListenAddr, "Config file should override defaults for listen address")
		assert.True(t, config.PrometheusCR.Enabled, "Config file should override defaults for prometheus CR enabled")
		assert.True(t, config.HTTPS.Enabled, "Config file should override defaults for HTTPS enabled")
		assert.Equal(t, ":7443", config.HTTPS.ListenAddr, "Config file should override defaults for HTTPS listen address")
		assert.Equal(t, kubeConfigPath, config.KubeConfigFilePath, "Config file should set kube config path")
		assert.Equal(t, "config-file-namespace", config.CollectorNamespace, "Config file should set collector namespace")
	})

	t.Run("environment variables are applied", func(t *testing.T) {
		// Setup: create empty config file and dummy kube config
		tempDir := t.TempDir()
		emptyConfigPath := filepath.Join(tempDir, "empty.yaml")
		err := os.WriteFile(emptyConfigPath, []byte("{}"), 0600)
		require.NoError(t, err)

		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		// Set environment variable
		testNamespace := "test-namespace"
		t.Setenv("OTELCOL_NAMESPACE", testNamespace)

		// Prepare args for Load function
		args := []string{
			"--" + configFilePathFlagName + "=" + emptyConfigPath,
			"--" + kubeConfigPathFlagName + "=" + kubeConfigPath,
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert environment variable is applied
		assert.Equal(t, testNamespace, config.CollectorNamespace, "Environment variable should be applied")
	})

	t.Run("loading priority order: defaults <- config file <- env vars <- CLI", func(t *testing.T) {
		// Setup: create config file with values and dummy kube config
		tempDir := t.TempDir()
		kubeConfigPath := createDummyKubeConfig(t, tempDir)

		// Config file sets values
		configContent := `
collector_namespace: "config-file-namespace"
listen_addr: ":9090"
prometheus_cr:
  enabled: false
kube_config_file_path: "` + kubeConfigPath + `"
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		// Environment variable sets value
		testNamespace := "env-var-namespace"
		t.Setenv("OTELCOL_NAMESPACE", testNamespace)

		// Prepare args for Load function with CLI values
		cliListenAddr := ":8888"
		args := []string{
			"--" + configFilePathFlagName + "=" + configPath,
			"--" + listenAddrFlagName + "=" + cliListenAddr,
			"--" + prometheusCREnabledFlagName + "=true",
		}

		// Load config using the full Load function
		config, err := Load(args)
		require.NoError(t, err)

		// Assert correct priority: CLI over env vars over config file over defaults
		assert.Equal(t, testNamespace, config.CollectorNamespace, "Env var should override config file for namespace")
		assert.Equal(t, cliListenAddr, config.ListenAddr, "CLI should override config file for listen address")
		assert.True(t, config.PrometheusCR.Enabled, "CLI should override config file for prometheus CR enabled")
	})
}
