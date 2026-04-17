// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apihelpers

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// ConfigYAML / Telemetry / MetricsEndpoint / ApplyDefaults tests
// (originally in apis/v1beta1/config_test.go, moved here because the functions moved to apihelpers)

func TestConfigYaml(t *testing.T) {
	cfg := &v1beta1.Config{
		Receivers: v1beta1.AnyConfig{
			Object: map[string]any{
				"otlp": nil,
			},
		},
		Processors: &v1beta1.AnyConfig{
			Object: map[string]any{
				"modify_2000": "enabled",
			},
		},
		Exporters: v1beta1.AnyConfig{
			Object: map[string]any{
				"otlp/exporter": nil,
			},
		},
		Connectors: &v1beta1.AnyConfig{
			Object: map[string]any{
				"con": "magic",
			},
		},
		Extensions: &v1beta1.AnyConfig{
			Object: map[string]any{
				"addon": "option1",
			},
		},
		Service: v1beta1.Service{
			Extensions: []string{"addon"},
			Telemetry: &v1beta1.AnyConfig{
				Object: map[string]any{
					"insights": "yeah!",
				},
			},
			Pipelines: map[string]*v1beta1.Pipeline{
				"traces": {
					Receivers:  []string{"otlp"},
					Processors: []string{"modify_2000"},
					Exporters:  []string{"otlp/exporter", "con"},
				},
			},
		},
	}
	yamlCollector, err := ConfigYAML(cfg)
	require.NoError(t, err)

	const expected = `receivers:
  otlp: null
exporters:
  otlp/exporter: null
processors:
  modify_2000: enabled
connectors:
  con: magic
extensions:
  addon: option1
service:
  extensions:
    - addon
  telemetry:
    insights: yeah!
  pipelines:
    traces:
      exporters:
        - otlp/exporter
        - con
      processors:
        - modify_2000
      receivers:
        - otlp
`

	assert.Equal(t, expected, yamlCollector)
}

func TestGetTelemetryFromYAML(t *testing.T) {
	collectorYaml, err := os.ReadFile(testdataDir + "/otelcol-demo.yaml")
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	telemetry := &Telemetry{
		Metrics: MetricsConfig{
			Level:   "detailed",
			Address: "0.0.0.0:8888",
		},
	}
	logger := logr.Discard()
	assert.Equal(t, telemetry, GetServiceTelemetry(&cfg.Service, logger))
}

func TestGetTelemetryFromYAMLIsNil(t *testing.T) {
	collectorYaml, err := os.ReadFile(testdataDir + "/otelcol-couchbase.yaml")
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	logger := logr.Discard()
	assert.Nil(t, GetServiceTelemetry(&cfg.Service, logger))
}

func TestConfigMetricsEndpoint(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		expectedAddr string
		expectedPort int32
		expectedErr  bool
		config       v1beta1.Service
	}{
		{
			desc:         "custom port",
			expectedAddr: "localhost",
			expectedPort: 9090,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "localhost:9090",
						},
					},
				},
			},
		},
		{
			desc:         "custom port ipv6",
			expectedAddr: "[::]",
			expectedPort: 9090,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "[::]:9090",
						},
					},
				},
			},
		},
		{
			desc:         "missing port",
			expectedAddr: "localhost",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "localhost",
						},
					},
				},
			},
		},
		{
			desc:         "missing port ipv6",
			expectedAddr: "[::]",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "[::]",
						},
					},
				},
			},
		},
		{
			desc:         "env var and missing port",
			expectedAddr: "${env:POD_IP}",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "${env:POD_IP}",
						},
					},
				},
			},
		},
		{
			desc:         "env var and missing port ipv6",
			expectedAddr: "[${env:POD_IP}]",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "[${env:POD_IP}]",
						},
					},
				},
			},
		},
		{
			desc:         "env var and with port",
			expectedAddr: "${POD_IP}",
			expectedPort: 1234,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "${POD_IP}:1234",
						},
					},
				},
			},
		},
		{
			desc:         "env var and with port ipv6",
			expectedAddr: "[${POD_IP}]",
			expectedPort: 1234,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "[${POD_IP}]:1234",
						},
					},
				},
			},
		},
		{
			desc:        "port is env var",
			expectedErr: true,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "localhost:${env:POD_PORT}",
						},
					},
				},
			},
		},
		{
			desc:        "port is env var ipv6",
			expectedErr: true,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "[::]:${env:POD_PORT}",
						},
					},
				},
			},
		},
		{
			desc:         "missing address",
			expectedAddr: "0.0.0.0",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"level": "detailed",
						},
					},
				},
			},
		},
		{
			desc:         "missing metrics",
			expectedAddr: "0.0.0.0",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{},
			},
		},
		{
			desc:         "missing telemetry",
			expectedAddr: "0.0.0.0",
			expectedPort: 8888,
		},
		{
			desc:         "configured telemetry",
			expectedAddr: "1.2.3.4",
			expectedPort: 4567,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "1.2.3.4:4567",
						},
					},
				},
			},
		},
		{
			desc:         "derive from readers prometheus host+port",
			expectedAddr: "0.0.0.0",
			expectedPort: 8889,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"level": "detailed",
							"readers": []any{
								map[string]any{
									"pull": map[string]any{
										"exporter": map[string]any{
											"prometheus": map[string]any{
												"host": "0.0.0.0",
												"port": 8889,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:         "derive from readers prometheus port only (default host)",
			expectedAddr: "0.0.0.0",
			expectedPort: 8899,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"readers": []any{
								map[string]any{
									"pull": map[string]any{
										"exporter": map[string]any{
											"prometheus": map[string]any{
												"port": 8899,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:         "derive from readers prometheus host only (default port)",
			expectedAddr: "127.0.0.1",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"readers": []any{
								map[string]any{
									"pull": map[string]any{
										"exporter": map[string]any{
											"prometheus": map[string]any{
												"host": "127.0.0.1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:         "readers takes precedence over address",
			expectedAddr: "0.0.0.0",
			expectedPort: 8889,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"address": "1.2.3.4:4567",
							"readers": []any{
								map[string]any{
									"pull": map[string]any{
										"exporter": map[string]any{
											"prometheus": map[string]any{
												"host": "0.0.0.0",
												"port": 8889,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc:         "readers present but no prometheus -> defaults",
			expectedAddr: "0.0.0.0",
			expectedPort: 8888,
			config: v1beta1.Service{
				Telemetry: &v1beta1.AnyConfig{
					Object: map[string]any{
						"metrics": map[string]any{
							"readers": []any{
								map[string]any{
									"pull": map[string]any{
										"exporter": map[string]any{
											"otlp": map[string]any{
												"protocols": map[string]any{
													"http": map[string]any{
														"endpoint": "0.0.0.0:19001",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			addr, port, err := ServiceMetricsEndpoint(&tt.config, logr.Discard())
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedAddr, addr)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestTelemetryLogsPreservedWithMetrics(t *testing.T) {
	cfg := &v1beta1.Config{
		Service: v1beta1.Service{
			Telemetry: &v1beta1.AnyConfig{
				Object: map[string]any{
					"logs": map[string]any{
						"level": "debug",
					},
				},
			},
		},
	}

	expected := &v1beta1.Config{
		Service: v1beta1.Service{
			Telemetry: &v1beta1.AnyConfig{
				Object: map[string]any{
					"logs": map[string]any{
						"level": "debug",
					},
					"metrics": map[string]any{
						"readers": []any{
							map[string]any{
								"pull": map[string]any{
									"exporter": map[string]any{
										"prometheus": map[string]any{
											"host": "0.0.0.0",
											"port": int32(8888),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ApplyDefaults(cfg, logr.Discard())
	require.NoError(t, err)

	logger := logr.Discard()
	telemetry := GetServiceTelemetry(&cfg.Service, logger)
	require.NotNil(t, telemetry)
	require.Equal(t, expected, cfg)
}

func TestTelemetryIncompleteConfigAppliesDefaults(t *testing.T) {
	cfg := &v1beta1.Config{
		Service: v1beta1.Service{
			Telemetry: &v1beta1.AnyConfig{
				Object: map[string]any{
					"metrics": map[string]any{
						"level": "basic",
						"readers": []any{
							map[string]any{
								"periodic": map[string]any{
									"exporter": map[string]any{
										"otlp": map[string]any{
											"endpoint": "otlp_host:4317",
											// Missing protocol - makes this invalid
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := ApplyDefaults(cfg, logr.Discard())
	require.NoError(t, err)

	logger := logr.Discard()
	telemetry := GetServiceTelemetry(&cfg.Service, logger)
	require.NotNil(t, telemetry)

	require.Len(t, telemetry.Metrics.Readers, 1)

	require.NotNil(t, telemetry.Metrics.Readers[0].Pull)
	require.NotNil(t, telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus)
	require.Equal(t, "0.0.0.0", *telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
	require.Equal(t, 8888, *telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
}

const testdataDir = "../apis/v1beta1/testdata"

func TestNullObjects_issue_3445(t *testing.T) {
	collectorYaml, err := os.ReadFile(testdataDir + "/issue-3452.yaml")
	require.NoError(t, err)

	collectorJson, err := go_yaml.YAMLToJSON(collectorYaml)
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = json.Unmarshal(collectorJson, cfg)
	require.NoError(t, err)

	_, err = ApplyDefaults(cfg, logr.Discard())
	require.NoError(t, err)
	assert.Empty(t, NullObjects(cfg))
}

func TestConfig_getEnvironmentVariablesForComponentKinds(t *testing.T) {
	tests := []struct {
		name       string
		config     *v1beta1.Config
		envVarsLen int
	}{
		{
			name: "no env vars",
			config: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]any{
						"myreceiver": map[string]any{
							"env": "test",
						},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"test": {
							Receivers: []string{"myreceiver"},
						},
					},
				},
			},
			envVarsLen: 0,
		},
		{
			name: "kubeletstats env vars",
			config: &v1beta1.Config{
				Receivers: v1beta1.AnyConfig{
					Object: map[string]any{
						"kubeletstats": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Pipelines: map[string]*v1beta1.Pipeline{
						"test": {
							Receivers: []string{"kubeletstats"},
						},
					},
				},
			},
			envVarsLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logr.Discard()
			envVars, err := GetEnvironmentVariables(tt.config, logger)

			assert.NoError(t, err)
			assert.Len(t, envVars, tt.envVarsLen)
		})
	}
}

func TestConfig_GetReceiverPorts(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    []v1.ServicePort
		wantErr bool
	}{
		{
			name: "k8sevents",
			file: testdataDir + "/otelcol-k8sevents.yaml",
			want: []v1.ServicePort{
				{
					Name:        "otlp-http",
					Protocol:    "",
					AppProtocol: ptr.To("http"),
					Port:        4318,
					TargetPort:  intstr.FromInt32(4318),
				},
			},
			wantErr: false, // Silently fail
		},
		{
			name:    "connectors",
			file:    testdataDir + "/otelcol-connectors.yaml",
			want:    nil,
			wantErr: false, // Silently fail
		},
		{
			name: "couchbase",
			file: testdataDir + "/otelcol-couchbase.yaml",
			want: nil, // Couchbase uses a prometheus scraper, no ports should be opened
		},
		{
			name: "demo",
			file: testdataDir + "/otelcol-demo.yaml",
			want: []v1.ServicePort{
				{
					Name:        "otlp-grpc",
					Protocol:    "",
					AppProtocol: ptr.To("grpc"),
					Port:        4317,
					TargetPort:  intstr.FromInt32(4317),
				},
			},
		},
		{
			name: "extensions",
			file: testdataDir + "/otelcol-extensions.yaml",
			want: []v1.ServicePort{
				{
					Name:        "otlp-grpc",
					Protocol:    "",
					AppProtocol: ptr.To("grpc"),
					Port:        4317,
					TargetPort:  intstr.FromInt32(4317),
				},
			},
		},
		{
			name: "filelog",
			file: testdataDir + "/otelcol-filelog.yaml",
			want: nil,
		},
		{
			name: "null",
			file: testdataDir + "/otelcol-null-values.yaml",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &v1beta1.Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			ports, err := GetReceiverPorts(c, logr.Discard())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equalf(t, tt.want, ports, "GetReceiverPorts()")
		})
	}
}

func TestConfig_GetExporterPorts(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    []v1.ServicePort
		wantErr bool
	}{
		{
			name:    "connectors",
			file:    testdataDir + "/otelcol-connectors.yaml",
			want:    nil,
			wantErr: false,
		},
		{
			name: "couchbase",
			file: testdataDir + "/otelcol-couchbase.yaml",
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 9123,
				},
			},
		},
		{
			name: "demo",
			file: testdataDir + "/otelcol-demo.yaml",
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 8889,
				},
			},
		},
		{
			name: "extensions",
			file: testdataDir + "/otelcol-extensions.yaml",
			want: nil,
		},
		{
			name: "filelog",
			file: testdataDir + "/otelcol-filelog.yaml",
			want: nil,
		},
		{
			name: "null",
			file: testdataDir + "/otelcol-null-values.yaml",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &v1beta1.Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			ports, err := GetExporterPorts(c, logr.Discard())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatchf(t, tt.want, ports, "GetExporterPorts()")
		})
	}
}

func TestConfig_GetLivenessProbe(t *testing.T) {
	tests := []struct {
		name      string
		config    *v1beta1.Config
		wantProbe *v1.Probe
		wantErr   bool
	}{
		{
			name: "nil extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "nil extensions with health_check in service extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions with health_check in service extensions should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension enabled should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom path",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"path": "/healthz",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom endpoint port",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"endpoint": "0.0.0.0:8080",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(8080),
					},
				},
			},
		},
		{
			name: "extension without liveness probe should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"jaeger_query": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"jaeger_query"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "invalid health_check config should return error",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": func() {},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetLivenessProbe(tt.config, logr.Discard())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLivenessProbe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantProbe, got); diff != "" {
				t.Errorf("GetLivenessProbe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfig_GetReadinessProbe(t *testing.T) {
	tests := []struct {
		name      string
		config    *v1beta1.Config
		wantProbe *v1.Probe
		wantErr   bool
	}{
		{
			name: "nil extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "nil extensions with health_check in service extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions with health_check in service extensions should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension enabled should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom path",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"path": "/healthz",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom endpoint port",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"endpoint": "0.0.0.0:8080",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(8080),
					},
				},
			},
		},
		{
			name: "extension without readiness probe should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"jaeger_query": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"jaeger_query"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "invalid health_check config should return error",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": func() {},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetReadinessProbe(tt.config, logr.Discard())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetReadinessProbe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantProbe, got); diff != "" {
				t.Errorf("GetReadinessProbe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfig_GetStartupProbe(t *testing.T) {
	tests := []struct {
		name      string
		config    *v1beta1.Config
		wantProbe *v1.Probe
		wantErr   bool
	}{
		{
			name: "nil extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "nil extensions with health_check in service extensions should return nil",
			config: &v1beta1.Config{
				Extensions: nil,
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{},
				},
			},
			wantProbe: nil,
		},
		{
			name: "empty extensions with health_check in service extensions should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension enabled should return probe",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom path",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"path": "/healthz",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(13133),
					},
				},
			},
		},
		{
			name: "health_check extension with custom endpoint port",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": map[string]any{
							"endpoint": "0.0.0.0:8080",
						},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantProbe: &v1.Probe{
				ProbeHandler: v1.ProbeHandler{
					HTTPGet: &v1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromInt32(8080),
					},
				},
			},
		},
		{
			name: "extension without startup probe should return nil",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"jaeger_query": map[string]any{},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"jaeger_query"},
				},
			},
			wantProbe: nil,
		},
		{
			name: "invalid health_check config should return error",
			config: &v1beta1.Config{
				Extensions: &v1beta1.AnyConfig{
					Object: map[string]any{
						"health_check": func() {},
					},
				},
				Service: v1beta1.Service{
					Extensions: []string{"health_check"},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStartupProbe(tt.config, logr.Discard())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStartupProbe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.wantProbe, got); diff != "" {
				t.Errorf("GetStartupProbe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNullObjects(t *testing.T) {
	collectorYaml, err := os.ReadFile(testdataDir + "/otelcol-null-values.yaml")
	require.NoError(t, err)

	collectorJson, err := go_yaml.YAMLToJSON(collectorYaml)
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = json.Unmarshal(collectorJson, cfg)
	require.NoError(t, err)

	nullObjects := NullObjects(cfg)
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

func TestNullObjects_go_yaml(t *testing.T) {
	collectorYaml, err := os.ReadFile(testdataDir + "/otelcol-null-values.yaml")
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)

	nullObjects := NullObjects(cfg)
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

func TestConfig_GetEnabledComponents(t *testing.T) {
	tests := []struct {
		name string
		file string
		want map[v1beta1.ComponentKind]map[string]any
	}{
		{
			name: "connectors",
			file: testdataDir + "/otelcol-connectors.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver: {
					"foo":   struct{}{},
					"count": struct{}{},
				},
				v1beta1.KindProcessor: {},
				v1beta1.KindExporter: {
					"bar":   struct{}{},
					"count": struct{}{},
				},
				v1beta1.KindExtension: {},
			},
		},
		{
			name: "couchbase",
			file: testdataDir + "/otelcol-couchbase.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver: {
					"prometheus/couchbase": struct{}{},
				},
				v1beta1.KindProcessor: {
					"filter/couchbase":           struct{}{},
					"metricstransform/couchbase": struct{}{},
					"transform/couchbase":        struct{}{},
				},
				v1beta1.KindExporter: {
					"prometheus": struct{}{},
				},
				v1beta1.KindExtension: {},
			},
		},
		{
			name: "demo",
			file: testdataDir + "/otelcol-demo.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver: {
					"otlp": struct{}{},
				},
				v1beta1.KindProcessor: {
					"memory_limiter": struct{}{},
				},
				v1beta1.KindExporter: {
					"debug":      struct{}{},
					"zipkin":     struct{}{},
					"otlp":       struct{}{},
					"prometheus": struct{}{},
				},
				v1beta1.KindExtension: {
					"health_check": struct{}{},
					"pprof":        struct{}{},
					"zpages":       struct{}{},
				},
			},
		},
		{
			name: "extensions",
			file: testdataDir + "/otelcol-extensions.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver: {
					"otlp": struct{}{},
				},
				v1beta1.KindProcessor: {},
				v1beta1.KindExporter: {
					"otlp/auth": struct{}{},
				},
				v1beta1.KindExtension: {
					"oauth2client": struct{}{},
				},
			},
		},
		{
			name: "filelog",
			file: testdataDir + "/otelcol-filelog.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver: {
					"filelog": struct{}{},
				},
				v1beta1.KindProcessor: {},
				v1beta1.KindExporter: {
					"debug": struct{}{},
				},
				v1beta1.KindExtension: {},
			},
		},
		{
			name: "null",
			file: testdataDir + "/otelcol-null-values.yaml",
			want: map[v1beta1.ComponentKind]map[string]any{
				v1beta1.KindReceiver:  {},
				v1beta1.KindProcessor: {},
				v1beta1.KindExporter:  {},
				v1beta1.KindExtension: {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &v1beta1.Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, GetEnabledComponents(c), "GetEnabledComponents()")
		})
	}
}
