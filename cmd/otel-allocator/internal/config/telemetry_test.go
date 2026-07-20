// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelconf "go.opentelemetry.io/contrib/otelconf"
)

func strPtr(s string) *string { return &s }

func grpcReader(endpoint, temporality string) TelemetryConfig {
	return TelemetryConfig{Metrics: &MetricsConfig{Readers: []MetricReader{{
		Periodic: &PeriodicMetricReader{
			Exporter: MetricExporter{
				OTLPGrpc: &OTLPGrpcExporterConfig{
					Endpoint:              endpoint,
					TemporalityPreference: temporality,
				},
			},
		},
	}}}}
}

func httpReader(endpoint string) TelemetryConfig {
	return TelemetryConfig{Metrics: &MetricsConfig{Readers: []MetricReader{{
		Periodic: &PeriodicMetricReader{
			Exporter: MetricExporter{
				OTLPHttp: &OTLPHttpExporterConfig{Endpoint: endpoint},
			},
		},
	}}}}
}

func TestValidateTelemetry(t *testing.T) {
	tests := []struct {
		name    string
		cfg     TelemetryConfig
		wantErr bool
	}{
		{name: "empty is valid", cfg: TelemetryConfig{}},
		{name: "nil metrics is valid", cfg: TelemetryConfig{Metrics: nil}},
		{name: "valid grpc delta", cfg: grpcReader("example.com:4317", "delta")},
		{name: "valid grpc low_memory", cfg: grpcReader("example.com:4317", "low_memory")},
		{name: "valid grpc cumulative", cfg: grpcReader("example.com:4317", "cumulative")},
		{name: "valid grpc empty temporality", cfg: grpcReader("example.com:4317", "")},
		{name: "valid http defaults", cfg: httpReader("http://gw:4318")},
		{name: "missing grpc endpoint", cfg: grpcReader("", ""), wantErr: true},
		{name: "missing http endpoint", cfg: httpReader(""), wantErr: true},
		{name: "bad grpc temporality", cfg: grpcReader("gw:4317", "gauge"), wantErr: true},
		{name: "bad http temporality", cfg: TelemetryConfig{Metrics: &MetricsConfig{Readers: []MetricReader{{
			Periodic: &PeriodicMetricReader{Exporter: MetricExporter{
				OTLPHttp: &OTLPHttpExporterConfig{Endpoint: "http://gw:4318", TemporalityPreference: "lowmemory"},
			}},
		}}}}, wantErr: true},
		{name: "periodic without exporter", cfg: TelemetryConfig{Metrics: &MetricsConfig{Readers: []MetricReader{{
			Periodic: &PeriodicMetricReader{},
		}}}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTelemetry(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTelemetryDeclarativeCompatibility verifies that the testdata YAML representing a
// Target Allocator telemetry config (in the TA format) can also be parsed as a valid
// OTel declarative configuration fragment via otelconf.ParseYAML. This acts as a schema
// compatibility smoke test: if the field names diverge from the spec, this test fails.
func TestTelemetryDeclarativeCompatibility(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "telemetry_otlp_test.yaml"))
	require.NoError(t, err)

	cfg, err := otelconf.ParseYAML(content)
	require.NoError(t, err, "telemetry YAML must be valid OTel declarative config")
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.MeterProvider, "meter_provider must be parsed")
	assert.Len(t, cfg.MeterProvider.Readers, 2, "both readers must be parsed")

	// Verify first reader: gRPC with delta temporality
	grpcReader := cfg.MeterProvider.Readers[0]
	require.NotNil(t, grpcReader.Periodic)
	require.NotNil(t, grpcReader.Periodic.Exporter.OTLPGrpc)
	assert.Equal(t, "example.com:4317", string(*grpcReader.Periodic.Exporter.OTLPGrpc.Endpoint))

	// Verify second reader: HTTP
	httpReader := cfg.MeterProvider.Readers[1]
	require.NotNil(t, httpReader.Periodic)
	require.NotNil(t, httpReader.Periodic.Exporter.OTLPHttp)
	assert.Equal(t, "https://ingest.example.com/api/v2/otlp", string(*httpReader.Periodic.Exporter.OTLPHttp.Endpoint))
}

