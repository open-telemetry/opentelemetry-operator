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

func TestNewOTLPMetricReader(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.OTLPExporterConfig
	}{
		{
			name: "grpc default",
			cfg:  &config.OTLPExporterConfig{Endpoint: "example.com:4317", Insecure: true},
		},
		{
			name: "grpc explicit delta",
			cfg:  &config.OTLPExporterConfig{Protocol: "grpc", Endpoint: "example.com:4317", Temporality: "delta", Insecure: true},
		},
		{
			name: "http base url",
			cfg:  &config.OTLPExporterConfig{Protocol: "http", Endpoint: "http://example.com:4318", Insecure: true, ExportInterval: 15 * time.Second, Timeout: 5 * time.Second},
		},
		{
			name: "http with signal path already present",
			cfg:  &config.OTLPExporterConfig{Protocol: "http", Endpoint: "http://example.com:4318/v1/metrics", Insecure: true},
		},
		{
			name: "grpc with URL scheme",
			cfg:  &config.OTLPExporterConfig{Protocol: "grpc", Endpoint: "http://example.com:4317", Insecure: true},
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
	// lowmemory: delta for counters, cumulative for gauges.
	assert.Equal(t, metricdata.DeltaTemporality, temporalitySelector("lowmemory")(sdkmetric.InstrumentKindCounter))
	assert.Equal(t, metricdata.CumulativeTemporality, temporalitySelector("lowmemory")(sdkmetric.InstrumentKindObservableGauge))
	// unset defaults to cumulative.
	assert.Equal(t, metricdata.CumulativeTemporality, temporalitySelector("")(sdkmetric.InstrumentKindCounter))
}

func TestExpandEnvRefs(t *testing.T) {
	t.Setenv("TA_TEST_TOKEN", "secret-value")
	assert.Equal(t, "Api-Token secret-value", expandEnvRefs("Api-Token ${env:TA_TEST_TOKEN}"))
	assert.Equal(t, "no refs here", expandEnvRefs("no refs here"))
	// Unset variable expands to empty.
	assert.Equal(t, "prefix-", expandEnvRefs("prefix-${env:TA_TEST_UNSET}"))

	headers := expandHeaders(map[string]string{"Authorization": "Api-Token ${env:TA_TEST_TOKEN}"})
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
	reader, err := newOTLPMetricReader(ctx, &config.OTLPExporterConfig{
		Protocol:       "http",
		Endpoint:       srv.URL,
		Insecure:       true,
		ExportInterval: 100 * time.Millisecond,
		Headers:        map[string]string{"Authorization": "Api-Token ${env:TA_TEST_TOKEN}"},
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
	found := false
	for _, attr := range res.Attributes() {
		if attr.Key == semconv.ServiceNameKey {
			assert.Equal(t, "target-allocator", attr.Value.AsString())
			found = true
		}
	}
	assert.True(t, found, "service.name attribute must be set")
}
