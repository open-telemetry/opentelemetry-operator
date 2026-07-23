// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

func grpcPeriodicReader(endpoint string, extra ...func(*config.OTLPGrpcExporterConfig)) *config.PeriodicMetricReader {
	cfg := &config.OTLPGrpcExporterConfig{Endpoint: endpoint}
	for _, fn := range extra {
		fn(cfg)
	}
	return &config.PeriodicMetricReader{Exporter: config.MetricExporter{OTLPGrpc: cfg}}
}

func httpPeriodicReader(endpoint string, extra ...func(*config.OTLPHttpExporterConfig)) *config.PeriodicMetricReader {
	cfg := &config.OTLPHttpExporterConfig{Endpoint: endpoint}
	for _, fn := range extra {
		fn(cfg)
	}
	return &config.PeriodicMetricReader{Exporter: config.MetricExporter{OTLPHttp: cfg}}
}

func TestNewOTLPMetricReader(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.PeriodicMetricReader
	}{
		{
			name: "grpc default",
			cfg:  grpcPeriodicReader("example.com:4317", func(c *config.OTLPGrpcExporterConfig) { c.Tls = &config.GrpcTlsConfig{Insecure: true} }),
		},
		{
			name: "grpc explicit delta",
			cfg: grpcPeriodicReader("example.com:4317", func(c *config.OTLPGrpcExporterConfig) {
				c.TemporalityPreference = "delta"
				c.Tls = &config.GrpcTlsConfig{Insecure: true}
			}),
		},
		{
			name: "http base url",
			cfg: &config.PeriodicMetricReader{
				Interval: 15000,
				Timeout:  5000,
				Exporter: config.MetricExporter{OTLPHttp: &config.OTLPHttpExporterConfig{Endpoint: "http://example.com:4318"}},
			},
		},
		{
			name: "http with signal path already present",
			cfg:  httpPeriodicReader("http://example.com:4318/v1/metrics"),
		},
		{
			name: "grpc with URL scheme",
			cfg:  grpcPeriodicReader("http://example.com:4317", func(c *config.OTLPGrpcExporterConfig) { c.Tls = &config.GrpcTlsConfig{Insecure: true} }),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := newOTLPMetricReader(context.Background(), tt.cfg)
			require.NoError(t, err)
			require.NotNil(t, reader)
			// The reader must attach to a meter provider (exercises the Prometheus bridge
			// producer wiring). We avoid Shutdown here since that would try to flush to
			// the unreachable endpoint.
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			require.NotNil(t, mp)
		})
	}
}

func TestTemporalitySelector(t *testing.T) {
	assert.Equal(t, metricdata.DeltaTemporality, temporalitySelector("delta")(sdkmetric.InstrumentKindCounter))
	assert.Equal(t, metricdata.CumulativeTemporality, temporalitySelector("cumulative")(sdkmetric.InstrumentKindCounter))
	// low_memory: delta for counters, cumulative for gauges.
	assert.Equal(t, metricdata.DeltaTemporality, temporalitySelector("low_memory")(sdkmetric.InstrumentKindCounter))
	assert.Equal(t, metricdata.CumulativeTemporality, temporalitySelector("low_memory")(sdkmetric.InstrumentKindObservableGauge))
	// unset defaults to cumulative.
	assert.Equal(t, metricdata.CumulativeTemporality, temporalitySelector("")(sdkmetric.InstrumentKindCounter))
}

func TestExpandEnvRefs(t *testing.T) {
	t.Setenv("TA_TEST_TOKEN", "secret-value")
	assert.Equal(t, "Api-Token secret-value", expandEnvRefs("Api-Token ${env:TA_TEST_TOKEN}"))
	assert.Equal(t, "no refs here", expandEnvRefs("no refs here"))
	// Unset variable expands to empty.
	assert.Equal(t, "prefix-", expandEnvRefs("prefix-${env:TA_TEST_UNSET}"))

	val := "Api-Token ${env:TA_TEST_TOKEN}"
	headers := expandHeaders([]config.NameValuePair{{Name: "Authorization", Value: &val}})
	assert.Equal(t, "Api-Token secret-value", headers["Authorization"])
	assert.Nil(t, expandHeaders(nil))
}

// TestOTLPExportEndToEnd starts a mock OTLP/HTTP receiver and verifies that a recorded
// metric is actually exported, that the endpoint's signal path is used, and that a header
// value referencing an environment variable is expanded before being sent.
func TestOTLPExportEndToEnd(t *testing.T) {
	t.Setenv("TA_TEST_TOKEN", "secret-value")

	var (
		mu       sync.Mutex
		gotPath  string
		gotAuth  string
		received = make(chan struct{}, 1)
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		// An empty body is a valid ExportMetricsServiceResponse.
		w.WriteHeader(http.StatusOK)
		select {
		case received <- struct{}{}:
		default:
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	authVal := "Api-Token ${env:TA_TEST_TOKEN}"
	reader, err := newOTLPMetricReader(ctx, &config.PeriodicMetricReader{
		Interval: 100,
		Exporter: config.MetricExporter{
			OTLPHttp: &config.OTLPHttpExporterConfig{
				Endpoint: srv.URL,
				Headers:  []config.NameValuePair{{Name: "Authorization", Value: &authVal}},
			},
		},
	})
	require.NoError(t, err)

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	counter, err := mp.Meter("test").Int64Counter("test_counter")
	require.NoError(t, err)
	counter.Add(ctx, 1)

	require.NoError(t, mp.ForceFlush(ctx))

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for OTLP export")
	}
	_ = mp.Shutdown(ctx)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "/v1/metrics", gotPath, "exporter should post to the OTLP metrics signal path")
	assert.Equal(t, "Api-Token secret-value", gotAuth, "env reference in header should be expanded")
}

func TestTelemetryResource(t *testing.T) {
	res, err := telemetryResource(context.Background())
	require.NoError(t, err)
	attrs := map[string]string{}
	for _, attr := range res.Attributes() {
		attrs[string(attr.Key)] = attr.Value.AsString()
	}
	assert.Equal(t, "target-allocator", attrs[string(semconv.ServiceNameKey)], "service.name must be set")
	assert.NotEmpty(t, attrs[string(semconv.ServiceInstanceIDKey)], "service.instance.id must be set")
}
