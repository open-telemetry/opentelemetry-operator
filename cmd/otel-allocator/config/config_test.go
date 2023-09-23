// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"testing"
	"time"

	commonconfig "github.com/prometheus/common/config"
	promconfig "github.com/prometheus/prometheus/config"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/file"
	"github.com/stretchr/testify/assert"
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
				LabelSelector: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
				PrometheusCR: PrometheusCRConfig{
					ScrapeInterval: model.Duration(time.Second * 60),
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:         "prometheus",
							HonorTimestamps: true,
							ScrapeInterval:  model.Duration(60 * time.Second),
							ScrapeTimeout:   model.Duration(10 * time.Second),
							MetricsPath:     "/metrics",
							Scheme:          "http",
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
				LabelSelector: map[string]string{
					"app.kubernetes.io/instance":   "default.test",
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
				PrometheusCR: PrometheusCRConfig{
					ScrapeInterval: DefaultCRScrapeInterval,
				},
				PromConfig: &promconfig.Config{
					GlobalConfig: promconfig.GlobalConfig{
						ScrapeInterval:     model.Duration(60 * time.Second),
						ScrapeTimeout:      model.Duration(10 * time.Second),
						EvaluationInterval: model.Duration(60 * time.Second),
					},
					ScrapeConfigs: []*promconfig.ScrapeConfig{
						{
							JobName:         "prometheus",
							HonorTimestamps: true,
							ScrapeInterval:  model.Duration(60 * time.Second),
							ScrapeTimeout:   model.Duration(10 * time.Second),
							MetricsPath:     "/metrics",
							Scheme:          "http",
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
				PodMonitorSelector: map[string]string{
					"release": "test",
				},
				ServiceMonitorSelector: map[string]string{
					"release": "test",
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

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name        string
		fileConfig  Config
		expectedErr error
	}{
		{
			name:        "promCR enabled, no Prometheus config",
			fileConfig:  Config{PromConfig: nil, PrometheusCR: PrometheusCRConfig{Enabled: true}},
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
				PromConfig: &promconfig.Config{ScrapeConfigs: []*promconfig.ScrapeConfig{{}}},
			},
			expectedErr: nil,
		},
		{
			name: "promCR enabled, Prometheus config present, scrapeConfigs present",
			fileConfig: Config{
				PromConfig:   &promconfig.Config{ScrapeConfigs: []*promconfig.ScrapeConfig{{}}},
				PrometheusCR: PrometheusCRConfig{Enabled: true},
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfig(&tc.fileConfig)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
