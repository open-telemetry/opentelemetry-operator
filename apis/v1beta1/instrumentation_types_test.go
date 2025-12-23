// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
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

func validateAgainstSchema(t *testing.T, obj []byte) {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	schemaPath := filepath.Join(testdataDir, "opentelemetry_configuration-v1.0.0-rc3.json")
	// jsonschema requires absolute path or URL for AddResource if it's referenced
	absPath, err := filepath.Abs(schemaPath)
	require.NoError(t, err)

	f, err := os.Open(absPath)
	require.NoError(t, err)
	defer f.Close()

	err = compiler.AddResource("schema.json", f)
	require.NoError(t, err)

	schema, err := compiler.Compile("schema.json")
	require.NoError(t, err)

	var v interface{}
	err = yaml.Unmarshal(obj, &v)
	require.NoError(t, err)

	err = schema.Validate(v)
	require.NoError(t, err)
}

// TestSDKConfigRoundTrip verifies that a comprehensive SDKConfig can be
// unmarshaled from YAML and marshaled back to produce equivalent YAML.
func TestSDKConfigRoundTrip(t *testing.T) {
	original, err := os.ReadFile(filepath.Join(testdataDir, "full_config.yaml"))
	require.NoError(t, err, "Failed to read testdata file")

	var config v1beta1.SDKConfig
	err = yaml.Unmarshal(original, &config)
	require.NoError(t, err, "Failed to unmarshal SDKConfig")

	// Marshal back to YAML and compare
	marshaled, err := yaml.Marshal(config)
	require.NoError(t, err, "Failed to marshal SDKConfig")
	assert.Equal(t, normalizeYAML(t, original), normalizeYAML(t, marshaled))

	// Validate against JSON schema
	validateAgainstSchema(t, marshaled)

	// Integration test: Create Instrumentation CR, apply, query, and validate
	ns := prepareNamespace(t, context.Background())
	inst := v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instrumentation",
			Namespace: ns,
		},
		Spec: v1beta1.InstrumentationSpec{
			Config: config,
		},
	}
	ctx := context.Background()
	err = k8sClient.Create(ctx, &inst)
	require.NoError(t, err, "Failed to create Instrumentation CR")

	var fetchedInst v1beta1.Instrumentation
	err = k8sClient.Get(ctx, types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace}, &fetchedInst)
	require.NoError(t, err, "Failed to get Instrumentation CR")

	fetchedConfig, err := json.Marshal(fetchedInst.Spec.Config)
	require.NoError(t, err, "Failed to marshal fetched config")
	validateAgainstSchema(t, fetchedConfig)
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
			var sampler v1beta1.Sampler
			err := yaml.Unmarshal([]byte(tt.yaml), &sampler)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(sampler)
			require.NoError(t, err)

			assert.Equal(t, normalizeYAML(t, []byte(tt.yaml)), normalizeYAML(t, marshaled))
		})
	}
}

// TestDefaultValues verifies that default values are applied by the API server.
func TestDefaultValues(t *testing.T) {
	ctx := context.Background()
	ns := prepareNamespace(t, ctx)

	otelinst := v1beta1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instrumentation-defaults",
			Namespace: ns,
		},
		Spec: v1beta1.InstrumentationSpec{
			Config: v1beta1.SDKConfig{
				FileFormat: "1.0",
				TracerProvider: &v1beta1.TracerProvider{
					Sampler: &v1beta1.Sampler{
						TraceIDRatioBased: &v1beta1.TraceIDRatioBasedSampler{
							// Ratio is matching the default, so we can omit it.
						},
					},
				},
				MeterProvider: &v1beta1.MeterProvider{
					Views: []v1beta1.View{
						{
							Stream: &v1beta1.ViewStream{
								Aggregation: &v1beta1.ViewAggregation{
									Base2ExponentialBucketHistogram: &v1beta1.Base2ExponentialBucketHistogramAggregation{
										// MaxScale and MaxSize and RecordMinMax match default.
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := k8sClient.Create(ctx, &otelinst)
	require.NoError(t, err)

	var fetched v1beta1.Instrumentation
	err = k8sClient.Get(ctx, types.NamespacedName{Name: otelinst.Name, Namespace: ns}, &fetched)
	require.NoError(t, err)

	// Verify Defaults
	assert.NotNil(t, fetched.Spec.Config.TracerProvider.Sampler.TraceIDRatioBased.Ratio)
	assert.Equal(t, 1.0, *fetched.Spec.Config.TracerProvider.Sampler.TraceIDRatioBased.Ratio)

	agg := fetched.Spec.Config.MeterProvider.Views[0].Stream.Aggregation.Base2ExponentialBucketHistogram
	assert.NotNil(t, agg.MaxScale)
	assert.Equal(t, 20, *agg.MaxScale)
	assert.NotNil(t, agg.MaxSize)
	assert.Equal(t, 160, *agg.MaxSize)
	assert.NotNil(t, agg.RecordMinMax)
	assert.True(t, *agg.RecordMinMax)
}

// TestSDKConfigValidation verifies that invalid configuration is rejected by the API server.
func TestSDKConfigValidation(t *testing.T) {
	ctx := context.Background()
	ns := prepareNamespace(t, ctx)

	ratio := 1.5
	maxScale := 21
	maxSize := 1
	protocol := "invalid"

	tests := []struct {
		name   string
		config v1beta1.SDKConfig
	}{
		{
			name: "invalid_ratio",
			config: v1beta1.SDKConfig{
				FileFormat: "1.0",
				TracerProvider: &v1beta1.TracerProvider{
					Sampler: &v1beta1.Sampler{
						TraceIDRatioBased: &v1beta1.TraceIDRatioBasedSampler{
							Ratio: &ratio,
						},
					},
				},
			},
		},
		{
			name: "invalid_max_scale",
			config: v1beta1.SDKConfig{
				FileFormat: "1.0",
				MeterProvider: &v1beta1.MeterProvider{
					Views: []v1beta1.View{
						{
							Stream: &v1beta1.ViewStream{
								Aggregation: &v1beta1.ViewAggregation{
									Base2ExponentialBucketHistogram: &v1beta1.Base2ExponentialBucketHistogramAggregation{
										MaxScale: &maxScale,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "invalid_max_size",
			config: v1beta1.SDKConfig{
				FileFormat: "1.0",
				MeterProvider: &v1beta1.MeterProvider{
					Views: []v1beta1.View{
						{
							Stream: &v1beta1.ViewStream{
								Aggregation: &v1beta1.ViewAggregation{
									Base2ExponentialBucketHistogram: &v1beta1.Base2ExponentialBucketHistogramAggregation{
										MaxSize: &maxSize,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "invalid_enum_protocol",
			config: v1beta1.SDKConfig{
				FileFormat: "1.0",
				TracerProvider: &v1beta1.TracerProvider{
					Processors: []v1beta1.SpanProcessor{
						{
							Batch: &v1beta1.BatchSpanProcessor{
								Exporter: v1beta1.SpanExporter{
									OTLP: &v1beta1.OTLP{
										Protocol: &protocol,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otelinst := v1beta1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-instrumentation-" + tt.name,
					Namespace: ns,
				},
				Spec: v1beta1.InstrumentationSpec{
					Config: tt.config,
				},
			}
			err := k8sClient.Create(ctx, &otelinst)
			require.Error(t, err)
		})
	}
}

// TestOTLPHeaderRoundTrip verifies header configuration can be round-tripped.
func TestOTLPHeaderRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "headers_with_value",
			yaml: `
batch:
  exporter:
    otlp:
      headers:
        - name: Content-Type
          value: application/json`,
		},
		{
			name: "headers_with_null_value",
			yaml: `
batch:
  exporter:
    otlp:
      headers:
        - name: X-Custom-Header
          value: null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var processor v1beta1.SpanProcessor
			err := yaml.Unmarshal([]byte(tt.yaml), &processor)
			require.NoError(t, err)

			marshaled, err := yaml.Marshal(processor)
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
			var processor v1beta1.SpanProcessor
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
			var reader v1beta1.MetricReader
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
			var agg v1beta1.ViewAggregation
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
			var processor v1beta1.LogRecordProcessor
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

	var view v1beta1.View
	err := yaml.Unmarshal([]byte(viewYAML), &view)
	require.NoError(t, err)

	marshaled, err := yaml.Marshal(view)
	require.NoError(t, err)

	assert.Equal(t, normalizeYAML(t, []byte(viewYAML)), normalizeYAML(t, marshaled))
}

// TestPropagatorRoundTrip verifies propagator configuration can be round-tripped.
func TestPropagatorRoundTrip(t *testing.T) {
	propagatorYAML := `
composite:
  - tracecontext: {}
  - baggage: {}
  - b3: {}
  - b3multi: {}
  - jaeger: {}
  - ottrace: {}`

	var propagator v1beta1.Propagator
	err := yaml.Unmarshal([]byte(propagatorYAML), &propagator)
	require.NoError(t, err)

	marshaled, err := yaml.Marshal(propagator)
	require.NoError(t, err)

	assert.Equal(t, normalizeYAML(t, []byte(propagatorYAML)), normalizeYAML(t, marshaled))
}
