// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	colmetrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/proto"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

// TestOTLPSelfTelemetryEndToEnd verifies the full self-telemetry OTLP export pipeline
//
// It registers a Prometheus counter on the default registry — mirroring what
// the Prometheus SD library does via discovery.CreateAndRegisterSDMetrics —
// and checks that:
//
//  1. The metric is bridged into the OTLP export.
//  2. When delta temporality is configured, successive exports carry
//     incremental (delta) values rather than the ever-growing cumulative sum.
func TestOTLPSelfTelemetryEndToEnd(t *testing.T) {
	// Register a counter on the default Prometheus registry, mirroring what
	// the Prometheus SD library registers (e.g. prometheus_sd_*_total).
	sdCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ta_itest_sd_operations_total",
		Help: "Simulates a Prometheus SD metric for the OTLP export integration test.",
	})
	require.NoError(t, prometheus.DefaultRegisterer.Register(sdCounter))
	t.Cleanup(func() { prometheus.DefaultRegisterer.Unregister(sdCounter) })

	// Collect OTLP/HTTP export requests from the meter provider.
	var (
		mu       sync.Mutex
		received []*colmetrics.ExportMetricsServiceRequest
		gotReq   = make(chan struct{}, 16)
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		req := &colmetrics.ExportMetricsServiceRequest{}
		if err := proto.Unmarshal(body, req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		select {
		case gotReq <- struct{}{}:
		default:
		}
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{
			Metrics: &config.MetricsConfig{
				Readers: []config.MetricReader{{
					Periodic: &config.PeriodicMetricReader{
						Interval: 100, // 100 ms — fast enough for a unit test
						Exporter: config.MetricExporter{
							OTLPHttp: &config.OTLPHttpExporterConfig{
								Endpoint:              srv.URL,
								TemporalityPreference: "delta",
							},
						},
					},
				}},
			},
		},
	}

	_, shutdown, err := setupMeterProvider(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
	})

	// Give the counter a non-zero initial value.
	sdCounter.Add(10)

	// --- First export: delta should equal the full cumulative value (10) ---
	waitForExportContaining(t, gotReq, &mu, &received, "ta_itest_sd_operations_total", 10)

	// Increment the counter so the cumulative value grows from 10 to 15.
	sdCounter.Add(5)

	// --- Second export: delta should be 5 (the increment since last export) ---
	waitForExportContaining(t, gotReq, &mu, &received, "ta_itest_sd_operations_total", 5)
}

// waitForExportContaining blocks until an OTLP export arrives that contains a
// data point for metricName whose delta value equals wantDelta.
func waitForExportContaining(
	t *testing.T,
	gotReq chan struct{},
	mu *sync.Mutex,
	received *[]*colmetrics.ExportMetricsServiceRequest,
	metricName string,
	wantDelta float64,
) {
	t.Helper()
	require.Eventually(t, func() bool {
		// Drain the notification channel so we check all received requests.
		for len(gotReq) > 0 {
			<-gotReq
		}
		mu.Lock()
		defer mu.Unlock()
		for _, req := range *received {
			if findDeltaValue(req, metricName) == wantDelta {
				// Clear processed requests to avoid re-matching in the next call.
				*received = nil
				return true
			}
		}
		return false
	}, 10*time.Second, 50*time.Millisecond,
		"expected %s with delta value %.0f in an OTLP export", metricName, wantDelta)
}

// findDeltaValue searches an ExportMetricsServiceRequest for a Sum data point
// with the given metric name and delta temporality, returning its value (or -1
// if not found).
func findDeltaValue(req *colmetrics.ExportMetricsServiceRequest, name string) float64 {
	for _, rm := range req.ResourceMetrics {
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name != name {
					continue
				}
				sum := m.GetSum()
				if sum == nil || sum.AggregationTemporality != otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA {
					continue
				}
				for _, dp := range sum.DataPoints {
					return dp.GetAsDouble()
				}
			}
		}
	}
	return -1
}
