// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

const testdataDir = "testdata/instrumentation"

// normalizeYAML unmarshals and re-marshals YAML to normalize formatting.
func normalizeYAML(t *testing.T, data []byte) string {
	t.Helper()
	var obj interface{}
	err := yaml.Unmarshal(data, &obj)
	require.NoError(t, err, "Failed to unmarshal YAML for normalization")
	normalized, err := yaml.Marshal(obj)
	require.NoError(t, err, "Failed to marshal YAML for normalization")
	return string(normalized)
}

// TestSDKConfigRoundTrip verifies that a comprehensive SDKConfig can be
// unmarshaled from YAML and marshaled back to produce equivalent YAML.
func TestSDKConfigRoundTrip(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(testdataDir, "full_config.yaml"))
	require.NoError(t, err, "Failed to read testdata file")

	var config SDKConfig
	err = yaml.Unmarshal(original, &config)
	require.NoError(t, err, "Failed to unmarshal SDKConfig")

	// Verify key fields were parsed correctly
	assert.Equal(t, "1.0-rc.3", config.FileFormat)
	require.NotNil(t, config.Disabled)
	assert.False(t, *config.Disabled)
	require.NotNil(t, config.AttributeLimits)
	assert.Equal(t, 4096, *config.AttributeLimits.AttributeValueLengthLimit)
	require.NotNil(t, config.Resource)
	assert.Len(t, config.Resource.Attributes, 2)
	require.NotNil(t, config.Propagator)
	assert.Equal(t, []string{"tracecontext", "baggage", "b3"}, config.Propagator.Composite)
	require.NotNil(t, config.TracerProvider)
	require.NotNil(t, config.MeterProvider)
	require.NotNil(t, config.LoggerProvider)

	// Marshal back to YAML and compare
	marshaled, err := yaml.Marshal(config)
	require.NoError(t, err, "Failed to marshal SDKConfig")
	assert.Equal(t, normalizeYAML(t, original), normalizeYAML(t, marshaled))
}

// TestSamplerTypes verifies all sampler configurations can be round-tripped.
func TestSamplerTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "always_on",
			yaml: `always_on: {}`,
		},
		{
			name: "always_off",
			yaml: `always_off: {}`,
		},
		{
			name: "trace_id_ratio_based",
			yaml: `
trace_id_ratio_based:
  ratio: 0.25`,
		},
		{
			name: "parent_based",
			yaml: `
parent_based:
  root:
    always_on: {}
  remote_parent_sampled:
    always_on: {}
  remote_parent_not_sampled:
    always_off: {}
  local_parent_sampled:
    always_on: {}
  local_parent_not_sampled:
    always_off: {}`,
		},
		{
			name: "jaeger_remote",
			yaml: `
jaeger_remote:
  endpoint: http://jaeger:14250
  polling_interval: 5000
  initial_sampler:
    trace_id_ratio_based:
      ratio: 0.1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sampler Sampler
			err := yaml.Unmarshal([]byte(tt.yaml), &sampler)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(sampler)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestSpanProcessorTypes verifies span processor configurations can be round-tripped.
func TestSpanProcessorTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "batch_with_otlp",
			yaml: `
batch:
  exporter:
    otlp:
      protocol: http/protobuf
      endpoint: http://collector:4318
      headers:
        - name: Authorization
          value: Bearer token
      compression: gzip
      timeout: 10000
  schedule_delay: 5000
  export_timeout: 30000
  max_queue_size: 2048
  max_export_batch_size: 512`,
		},
		{
			name: "simple_with_console",
			yaml: `
simple:
  exporter:
    console: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var processor SpanProcessor
			err := yaml.Unmarshal([]byte(tt.yaml), &processor)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(processor)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestMetricReaderTypes verifies metric reader configurations can be round-tripped.
func TestMetricReaderTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "periodic_with_otlp",
			yaml: `
periodic:
  exporter:
    otlp:
      protocol: grpc
      endpoint: http://collector:4317
      temporality_preference: delta
      default_histogram_aggregation: base2_exponential_bucket_histogram
  interval: 60000
  timeout: 30000`,
		},
		{
			name: "pull_with_prometheus",
			yaml: `
pull:
  exporter:
    prometheus:
      host: 0.0.0.0
      port: 9090
      without_units: true
      without_type_suffix: false
      without_scope_info: false`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader MetricReader
			err := yaml.Unmarshal([]byte(tt.yaml), &reader)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(reader)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestViewAggregationTypes verifies all aggregation types can be round-tripped.
func TestViewAggregationTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "default",
			yaml: `default: {}`,
		},
		{
			name: "drop",
			yaml: `drop: {}`,
		},
		{
			name: "sum",
			yaml: `sum: {}`,
		},
		{
			name: "last_value",
			yaml: `last_value: {}`,
		},
		{
			name: "explicit_bucket_histogram",
			yaml: `
explicit_bucket_histogram:
  boundaries:
    - 0
    - 5
    - 10
    - 25
    - 50
    - 100
  record_min_max: true`,
		},
		{
			name: "base2_exponential_bucket_histogram",
			yaml: `
base2_exponential_bucket_histogram:
  max_scale: 20
  max_size: 160
  record_min_max: true`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var agg ViewAggregation
			err := yaml.Unmarshal([]byte(tt.yaml), &agg)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(agg)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestLogRecordProcessorTypes verifies log processor configurations can be round-tripped.
func TestLogRecordProcessorTypes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "batch_with_otlp",
			yaml: `
batch:
  exporter:
    otlp:
      protocol: grpc
      endpoint: http://collector:4317
  schedule_delay: 1000
  export_timeout: 30000
  max_queue_size: 2048
  max_export_batch_size: 512`,
		},
		{
			name: "simple_with_console",
			yaml: `
simple:
  exporter:
    console: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var processor LogRecordProcessor
			err := yaml.Unmarshal([]byte(tt.yaml), &processor)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(processor)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestViewRoundTrip verifies a complete view configuration can be round-tripped.
func TestViewRoundTrip(t *testing.T) {
	viewYAML := `
selector:
  instrument_name: http.*
  instrument_type: histogram
  meter_name: my-meter
  meter_version: 1.0.0
  meter_schema_url: https://example.com/schema
  unit: ms
stream:
  name: custom_name
  description: Custom description
  attribute_keys:
    included:
      - http.method
      - http.status_code
    excluded:
      - http.url
  aggregation:
    explicit_bucket_histogram:
      boundaries:
        - 0
        - 5
        - 10
        - 25
        - 50
        - 75
        - 100
        - 250
        - 500
        - 1000
      record_min_max: true`

	var view View
	err := yaml.Unmarshal([]byte(viewYAML), &view)
	require.NoError(t, err)

	// Verify key fields
	require.NotNil(t, view.Selector)
	assert.Equal(t, "http.*", *view.Selector.InstrumentName)
	assert.Equal(t, "histogram", *view.Selector.InstrumentType)
	require.NotNil(t, view.Stream)
	assert.Equal(t, "custom_name", *view.Stream.Name)
	require.NotNil(t, view.Stream.AttributeKeys)
	assert.Equal(t, []string{"http.method", "http.status_code"}, view.Stream.AttributeKeys.Included)
	assert.Equal(t, []string{"http.url"}, view.Stream.AttributeKeys.Excluded)

	marshaled, err := yaml.Marshal(view)
	require.NoError(t, err)

	assert.Equal(t, normalizeYAML(t, []byte(viewYAML)), normalizeYAML(t, marshaled))
}

// TestPropagatorRoundTrip verifies propagator configuration can be round-tripped.
func TestPropagatorRoundTrip(t *testing.T) {
	propagatorYAML := `
composite:
  - tracecontext
  - baggage
  - b3
  - b3multi
  - jaeger
  - ottrace`

	var propagator Propagator
	err := yaml.Unmarshal([]byte(propagatorYAML), &propagator)
	require.NoError(t, err)

	expectedPropagators := []string{"tracecontext", "baggage", "b3", "b3multi", "jaeger", "ottrace"}
	assert.Equal(t, expectedPropagators, propagator.Composite)

	marshaled, err := yaml.Marshal(propagator)
	require.NoError(t, err)

	assert.Equal(t, normalizeYAML(t, []byte(propagatorYAML)), normalizeYAML(t, marshaled))
}
