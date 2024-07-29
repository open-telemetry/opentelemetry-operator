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

package v1beta1

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_yaml "gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

func TestConfigFiles(t *testing.T) {
	files, err := os.ReadDir("./testdata")
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join("./testdata", file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			collectorJson, err := yaml.YAMLToJSON(collectorYaml)
			require.NoError(t, err)

			cfg := &Config{}
			err = json.Unmarshal(collectorJson, cfg)
			require.NoError(t, err)
			jsonCfg, err := json.Marshal(cfg)
			require.NoError(t, err)

			assert.JSONEq(t, string(collectorJson), string(jsonCfg))
			yamlCfg, err := yaml.JSONToYAML(jsonCfg)
			require.NoError(t, err)
			assert.YAMLEq(t, string(collectorYaml), string(yamlCfg))
		})
	}
}

func TestNullObjects(t *testing.T) {
	collectorYaml, err := os.ReadFile("./testdata/otelcol-null-values.yaml")
	require.NoError(t, err)

	collectorJson, err := yaml.YAMLToJSON(collectorYaml)
	require.NoError(t, err)

	cfg := &Config{}
	err = json.Unmarshal(collectorJson, cfg)
	require.NoError(t, err)

	nullObjects := cfg.nullObjects()
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

func TestConfigFiles_go_yaml(t *testing.T) {
	files, err := os.ReadDir("./testdata")
	require.NoError(t, err)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "otelcol-") {
			continue
		}

		testFile := path.Join("./testdata", file.Name())
		t.Run(testFile, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(testFile)
			require.NoError(t, err)

			cfg := &Config{}
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
	collectorYaml, err := os.ReadFile("./testdata/otelcol-null-values.yaml")
	require.NoError(t, err)

	cfg := &Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)

	nullObjects := cfg.nullObjects()
	assert.Equal(t, []string{"connectors.spanmetrics:", "exporters.otlp.endpoint:", "extensions.health_check:", "processors.batch:", "receivers.otlp.protocols.grpc:", "receivers.otlp.protocols.http:"}, nullObjects)
}

func TestConfigYaml(t *testing.T) {
	cfg := &Config{
		Receivers: AnyConfig{
			Object: map[string]interface{}{
				"otlp": nil,
			},
		},
		Processors: &AnyConfig{
			Object: map[string]interface{}{
				"modify_2000": "enabled",
			},
		},
		Exporters: AnyConfig{
			Object: map[string]interface{}{
				"otlp/exporter": nil,
			},
		},
		Connectors: &AnyConfig{
			Object: map[string]interface{}{
				"con": "magic",
			},
		},
		Extensions: &AnyConfig{
			Object: map[string]interface{}{
				"addon": "option1",
			},
		},
		Service: Service{
			Extensions: &[]string{"addon"},
			Telemetry: &AnyConfig{
				Object: map[string]interface{}{
					"insights": "yeah!",
				},
			},
			Pipelines: map[string]*Pipeline{
				"traces": {
					Receivers:  []string{"otlp"},
					Processors: []string{"modify_2000"},
					Exporters:  []string{"otlp/exporter", "con"},
				},
			},
		},
	}
	yamlCollector, err := cfg.Yaml()
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
	collectorYaml, err := os.ReadFile("./testdata/otelcol-demo.yaml")
	require.NoError(t, err)

	cfg := &Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	telemetry := &Telemetry{
		Metrics: MetricsConfig{
			Level:   "detailed",
			Address: "0.0.0.0:8888",
		},
	}
	assert.Equal(t, telemetry, cfg.Service.GetTelemetry())
}

func TestGetTelemetryFromYAMLIsNil(t *testing.T) {
	collectorYaml, err := os.ReadFile("./testdata/otelcol-couchbase.yaml")
	require.NoError(t, err)

	cfg := &Config{}
	err = go_yaml.Unmarshal(collectorYaml, cfg)
	require.NoError(t, err)
	assert.Nil(t, cfg.Service.GetTelemetry())
}

func TestConfigToMetricsPort(t *testing.T) {

	for _, tt := range []struct {
		desc         string
		expectedPort int32
		config       Service
	}{
		{
			"custom port",
			9090,
			Service{
				Telemetry: &AnyConfig{
					Object: map[string]interface{}{
						"metrics": map[string]interface{}{
							"address": "0.0.0.0:9090",
						},
					},
				},
			},
		},
		{
			"bad address",
			8888,
			Service{
				Telemetry: &AnyConfig{
					Object: map[string]interface{}{
						"metrics": map[string]interface{}{
							"address": "0.0.0.0",
						},
					},
				},
			},
		},
		{
			"missing address",
			8888,
			Service{
				Telemetry: &AnyConfig{
					Object: map[string]interface{}{
						"metrics": map[string]interface{}{
							"level": "detailed",
						},
					},
				},
			},
		},
		{
			"missing metrics",
			8888,
			Service{
				Telemetry: &AnyConfig{},
			},
		},
		{
			"missing telemetry",
			8888,
			Service{},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// these are acceptable failures, we return to the collector's default metric port
			port, err := tt.config.MetricsPort()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestConfig_GetEnabledComponents(t *testing.T) {
	tests := []struct {
		name string
		file string
		want map[ComponentKind]map[string]interface{}
	}{

		{
			name: "connectors",
			file: "testdata/otelcol-connectors.yaml",
			want: map[ComponentKind]map[string]interface{}{
				KindReceiver: {
					"foo":   struct{}{},
					"count": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"bar":   struct{}{},
					"count": struct{}{},
				},
			},
		},
		{
			name: "couchbase",
			file: "testdata/otelcol-couchbase.yaml",
			want: map[ComponentKind]map[string]interface{}{
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
			},
		},
		{
			name: "demo",
			file: "testdata/otelcol-demo.yaml",
			want: map[ComponentKind]map[string]interface{}{
				KindReceiver: {
					"otlp": struct{}{},
				},
				KindProcessor: {
					"batch": struct{}{},
				},
				KindExporter: {
					"debug":      struct{}{},
					"zipkin":     struct{}{},
					"otlp":       struct{}{},
					"prometheus": struct{}{},
				},
			},
		},
		{
			name: "extensions",
			file: "testdata/otelcol-extensions.yaml",
			want: map[ComponentKind]map[string]interface{}{
				KindReceiver: {
					"otlp": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"otlp/auth": struct{}{},
				},
			},
		},
		{
			name: "filelog",
			file: "testdata/otelcol-filelog.yaml",
			want: map[ComponentKind]map[string]interface{}{
				KindReceiver: {
					"filelog": struct{}{},
				},
				KindProcessor: {},
				KindExporter: {
					"debug": struct{}{},
				},
			},
		},
		{
			name: "null",
			file: "testdata/otelcol-null-values.yaml",
			want: map[ComponentKind]map[string]interface{}{
				KindReceiver:  {},
				KindProcessor: {},
				KindExporter:  {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, c.GetEnabledComponents(), "GetEnabledComponents()")
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
			file: "testdata/otelcol-k8sevents.yaml",
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
			file:    "testdata/otelcol-connectors.yaml",
			want:    nil,
			wantErr: false, // Silently fail
		},
		{
			name: "couchbase",
			file: "testdata/otelcol-couchbase.yaml",
			want: nil, // Couchbase uses a prometheus scraper, no ports should be opened
		},
		{
			name: "demo",
			file: "testdata/otelcol-demo.yaml",
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
			file: "testdata/otelcol-extensions.yaml",
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
			file: "testdata/otelcol-filelog.yaml",
			want: nil,
		},
		{
			name: "null",
			file: "testdata/otelcol-null-values.yaml",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			ports, err := c.GetReceiverPorts(logr.Discard())
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
			file:    "testdata/otelcol-connectors.yaml",
			want:    nil,
			wantErr: false,
		},
		{
			name: "couchbase",
			file: "testdata/otelcol-couchbase.yaml",
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 9123,
				},
			},
		},
		{
			name: "demo",
			file: "testdata/otelcol-demo.yaml",
			want: []v1.ServicePort{
				{
					Name: "prometheus",
					Port: 8889,
				},
			},
		},
		{
			name: "extensions",
			file: "testdata/otelcol-extensions.yaml",
			want: nil,
		},
		{
			name: "filelog",
			file: "testdata/otelcol-filelog.yaml",
			want: nil,
		},
		{
			name: "null",
			file: "testdata/otelcol-null-values.yaml",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectorYaml, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			c := &Config{}
			err = go_yaml.Unmarshal(collectorYaml, c)
			require.NoError(t, err)
			ports, err := c.GetExporterPorts(logr.Discard())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatchf(t, tt.want, ports, "GetReceiverPorts()")
		})
	}
}
