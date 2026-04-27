// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apihelpers

import (
	"encoding/json"
	"os"
	"path"
	"strings"
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

// testdataPath returns the path to the testdata directory, which is shared from apis/v1beta1.
const testdataPath = "../../apis/v1beta1/testdata"

func TestConfigFiles(t *testing.T) {
	files, err := os.ReadDir(testdataPath)
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join(testdataPath, file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			collectorJson, err := go_yaml.YAMLToJSON(collectorYaml)
			require.NoError(t, err)

			cfg := &v1beta1.Config{}
			err = json.Unmarshal(collectorJson, cfg)
			require.NoError(t, err)
			jsonCfg, err := json.Marshal(cfg)
			require.NoError(t, err)

			assert.JSONEq(t, string(collectorJson), string(jsonCfg))
			yamlCfg, err := go_yaml.JSONToYAML(jsonCfg)
			require.NoError(t, err)
			assert.YAMLEq(t, string(collectorYaml), string(yamlCfg))
		})
	}
}

func TestNullObjects(t *testing.T) {
	collectorYaml, err := os.ReadFile(path.Join(testdataPath, "otelcol-null-values.yaml"))
	require.NoError(t, err)

	collectorJson, err := go_yaml.YAMLToJSON(collectorYaml)
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = json.Unmarshal(collectorJson, cfg)
	require.NoError(t, err)

	nullObjects := NullObjects(cfg)
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

func TestNullObjects_issue_3445(t *testing.T) {
	collectorYaml, err := os.ReadFile(path.Join(testdataPath, "issue-3452.yaml"))
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

func TestConfigFiles_go_yaml(t *testing.T) {
	files, err := os.ReadDir(testdataPath)
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join(testdataPath, file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			cfg := &v1beta1.Config{}
			err = go_yaml.Unmarshal(collectorYaml, cfg)
			require.NoError(t, err)
			yamlCfg, err := go_yaml.Marshal(cfg)
			require.NoError(t, err)

			require.NoError(t, err)
			assert.YAMLEq(t, string(collectorYaml), string(yamlCfg))
		})
	}
}

func TestNullObjects_go_yaml(t *testing.T) {
	collectorYaml, err := os.ReadFile(path.Join(testdataPath, "otelcol-null-values.yaml"))
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)

	nullObjects := NullObjects(cfg)
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

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
	yamlCollector, err := Yaml(cfg)
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
	collectorYaml, err := os.ReadFile(path.Join(testdataPath, "otelcol-demo.yaml"))
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	telemetry := &v1beta1.Telemetry{
		Metrics: v1beta1.MetricsConfig{
			Level:   "detailed",
			Address: "0.0.0.0:8888",
		},
	}
	logger := logr.Discard()
	assert.Equal(t, telemetry, GetTelemetry(&cfg.Service, logger))
}

func TestGetTelemetryFromYAMLIsNil(t *testing.T) {
	collectorYaml, err := os.ReadFile(path.Join(testdataPath, "otelcol-couchbase.yaml"))
	require.NoError(t, err)

	cfg := &v1beta1.Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	logger := logr.Discard()
	assert.Nil(t, GetTelemetry(&cfg.Service, logger))
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
			addr, port, err := MetricsEndpoint(&tt.config, logr.Discard())
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

func TestConfig_GetEnabledComponents(t *testing.T) {
	tests := []struct {
		name string
		file string
		want map[ComponentKind]map[string]any
	}{
		{
			name: "connectors",
			file: path.Join(testdataPath, "otelcol-connectors.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver: {
					"foo":   struct{}{},
					"count": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"bar":   struct{}{},
					"count": struct{}{},
				},
				KindExtension: {},
			},
		},
		{
			name: "couchbase",
			file: path.Join(testdataPath, "otelcol-couchbase.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver: {
					"prometheus/couchbase": struct{}{},
				},
				KindProcessor: {
					"filter/couchbase":           struct{}{},
					"metricstransform/couchbase": struct{}{},
					"transform/couchbase":        struct{}{},
				},
				KindExporter: {
					"prometheus": struct{}{},
				},
				KindExtension: {},
			},
		},
		{
			name: "demo",
			file: path.Join(testdataPath, "otelcol-demo.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver: {
					"otlp": struct{}{},
				},
				KindProcessor: {
					"memory_limiter": struct{}{},
				},
				KindExporter: {
					"debug":      struct{}{},
					"zipkin":     struct{}{},
					"otlp":       struct{}{},
					"prometheus": struct{}{},
				},
				KindExtension: {
					"health_check": struct{}{},
					"pprof":        struct{}{},
					"zpages":       struct{}{},
				},
			},
		},
		{
			name: "extensions",
			file: path.Join(testdataPath, "otelcol-extensions.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver: {
					"otlp": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"otlp/auth": struct{}{},
				},
				KindExtension: {
					"oauth2client": struct{}{},
				},
			},
		},
		{
			name: "filelog",
			file: path.Join(testdataPath, "otelcol-filelog.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver: {
					"filelog": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"debug": struct{}{},
				},
				KindExtension: {},
			},
		},
		{
			name: "null",
			file: path.Join(testdataPath, "otelcol-null-values.yaml"),
			want: map[ComponentKind]map[string]any{
				KindReceiver:  {},
				KindProcessor: {},
				KindExporter:  {},
				KindExtension: {},
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

func TestConfig_getEnvironmentVariablesForComponentKinds(t *testing.T) {
	tests := []struct {
		name           string
		config         *v1beta1.Config
		componentKinds []ComponentKind
		envVarsLen     int
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
			componentKinds: []ComponentKind{KindReceiver},
			envVarsLen:     0,
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
			componentKinds: []ComponentKind{KindReceiver},
			envVarsLen:     1,
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
			file: path.Join(testdataPath, "otelcol-k8sevents.yaml"),
			want: []v1.ServicePort{
				{
					Name:        "otlp-http",
					Protocol:    "",
					AppProtocol: ptr.To("http"),
					Port:        4318,
					TargetPort:  intstr.FromInt32(4318),
				},
			},
			wantErr: false,
		},
		{
			name:    "connectors",
			file:    path.Join(testdataPath, "otelcol-connectors.yaml"),
			want:    nil,
			wantErr: false,
		},
		{
			name: "couchbase",
			file: path.Join(testdataPath, "otelcol-couchbase.yaml"),
			want: nil,
		},
		{
			name: "demo",
			file: path.Join(testdataPath, "otelcol-demo.yaml"),
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
			file: path.Join(testdataPath, "otelcol-extensions.yaml"),
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
			file: path.Join(testdataPath, "otelcol-filelog.yaml"),
			want: nil,
		},
		{
			name: "null",
			file: path.Join(testdataPath, "otelcol-null-values.yaml"),
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
			file:    path.Join(testdataPath, "otelcol-connectors.yaml"),
			want:    nil,
			wantErr: false,
		},
		{
			name: "couchbase",
			file: path.Join(testdataPath, "otelcol-couchbase.yaml"),
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 9123,
				},
			},
		},
		{
			name: "demo",
			file: path.Join(testdataPath, "otelcol-demo.yaml"),
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 8889,
				},
			},
		},
		{
			name: "extensions",
			file: path.Join(testdataPath, "otelcol-extensions.yaml"),
			want: nil,
		},
		{
			name: "filelog",
			file: path.Join(testdataPath, "otelcol-filelog.yaml"),
			want: nil,
		},
		{
			name: "null",
			file: path.Join(testdataPath, "otelcol-null-values.yaml"),
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
			assert.ElementsMatchf(t, tt.want, ports, "GetReceiverPorts()")
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

	_, err := ServiceApplyDefaults(&cfg.Service, logr.Discard())
	require.NoError(t, err)

	logger := logr.Discard()
	telemetry := GetTelemetry(&cfg.Service, logger)
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

	_, err := ServiceApplyDefaults(&cfg.Service, logr.Discard())
	require.NoError(t, err)

	logger := logr.Discard()
	telemetry := GetTelemetry(&cfg.Service, logger)
	require.NotNil(t, telemetry)

	require.Len(t, telemetry.Metrics.Readers, 1)

	require.NotNil(t, telemetry.Metrics.Readers[0].Pull)
	require.NotNil(t, telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus)
	require.Equal(t, "0.0.0.0", *telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus.Host)
	require.Equal(t, 8888, *telemetry.Metrics.Readers[0].Pull.Exporter.Prometheus.Port)
}

func TestAnyConfigDeepCopyInto_NestedMapIndependence(t *testing.T) {
	src := v1beta1.AnyConfig{Object: map[string]any{
		"prometheus": map[string]any{
			"config": map[string]any{
				"scrape_configs": []any{
					map[string]any{
						"job_name": "kubelet",
						"tls_config": map[string]any{
							"ca_file":              "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
							"insecure_skip_verify": true,
						},
					},
				},
			},
		},
	}}

	dst := src.DeepCopy()

	scrapeConfigs := dst.Object["prometheus"].(map[string]any)["config"].(map[string]any)["scrape_configs"].([]any)
	tlsConfig := scrapeConfigs[0].(map[string]any)["tls_config"].(map[string]any)
	tlsConfig["min_version"] = "TLS12"

	srcTLS := src.Object["prometheus"].(map[string]any)["config"].(map[string]any)["scrape_configs"].([]any)[0].(map[string]any)["tls_config"].(map[string]any)
	assert.NotContains(t, srcTLS, "min_version", "DeepCopy must produce independent nested maps; source was mutated through the copy")
}

func TestAnyConfigDeepCopyInto_NilObject(t *testing.T) {
	src := v1beta1.AnyConfig{Object: nil}
	dst := src.DeepCopy()
	assert.Nil(t, dst.Object)
}

func TestAnyConfigDeepCopyInto_EmptyObject(t *testing.T) {
	src := v1beta1.AnyConfig{Object: map[string]any{}}
	dst := src.DeepCopy()
	assert.NotNil(t, dst.Object)
	assert.Empty(t, dst.Object)
	dst.Object["key"] = "value"
	assert.Empty(t, src.Object)
}

func TestAnyConfigDeepCopyInto_PreservesValues(t *testing.T) {
	src := v1beta1.AnyConfig{Object: map[string]any{
		"string_val": "hello",
		"number_val": float64(42),
		"bool_val":   true,
		"nested": map[string]any{
			"inner": "value",
			"list":  []any{"a", "b"},
		},
	}}

	dst := src.DeepCopy()

	assert.Equal(t, "hello", dst.Object["string_val"])
	assert.Equal(t, float64(42), dst.Object["number_val"])
	assert.Equal(t, true, dst.Object["bool_val"])
	nested := dst.Object["nested"].(map[string]any)
	assert.Equal(t, "value", nested["inner"])
	assert.Equal(t, []any{"a", "b"}, nested["list"])
}
