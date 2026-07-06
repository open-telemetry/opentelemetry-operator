// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTelemetry(t *testing.T) {
	otlp := func(o OTLPExporterConfig) TelemetryConfig {
		return TelemetryConfig{Metrics: MetricsConfig{OTLP: &o}}
	}

	tests := []struct {
		name    string
		cfg     TelemetryConfig
		wantErr bool
	}{
		{name: "empty is valid", cfg: TelemetryConfig{}},
		{name: "valid grpc delta", cfg: otlp(OTLPExporterConfig{Protocol: "grpc", Endpoint: "example.com:4317", Temporality: "delta"})},
		{name: "valid http defaults", cfg: otlp(OTLPExporterConfig{Endpoint: "http://gw:4318"})},
		{name: "missing endpoint", cfg: otlp(OTLPExporterConfig{Protocol: "grpc"}), wantErr: true},
		{name: "bad protocol", cfg: otlp(OTLPExporterConfig{Protocol: "thrift", Endpoint: "gw:4317"}), wantErr: true},
		{name: "bad temporality", cfg: otlp(OTLPExporterConfig{Endpoint: "gw:4317", Temporality: "gauge"}), wantErr: true},
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
