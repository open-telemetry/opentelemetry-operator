// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package integrationtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap/zaptest"

	prometheusreceiver "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	taconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

const collectorID = "test-collector"

// startMockTarget serves the given Prometheus exposition text on /metrics and
// returns its URL (use .Host for the scrape target address).
func startMockTarget(t *testing.T, exposition string) *url.URL {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = io.WriteString(w, exposition)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return u
}

// scrapeConfig builds a single-job scrape config pointing at targetAddr.
func scrapeConfig(t *testing.T, targetAddr string) []*promconfig.ScrapeConfig {
	t.Helper()
	cfg, err := promconfig.Load(fmt.Sprintf(`
scrape_configs:
  - job_name: itest
    scrape_interval: 200ms
    scrape_timeout: 200ms
    static_configs:
      - targets: ['%s']
`, targetAddr), taconfig.NopLogger)
	require.NoError(t, err)
	return cfg.ScrapeConfigs
}

// startTargetAllocator assembles the real allocator pipeline in-process and
// returns its base URL. No Kubernetes: a single collector is injected directly,
// so all discovered targets are assigned to it.
func startTargetAllocator(t *testing.T, scrapeConfigs []*promconfig.ScrapeConfig) *url.URL {
	t.Helper()
	log := logr.Discard()
	ctx, cancel := context.WithCancel(t.Context())

	alloc, err := allocation.New("consistent-hashing", log)
	require.NoError(t, err)

	addr := freeLocalAddress(t)
	srv, err := server.NewServer(log, alloc, addr)
	require.NoError(t, err)

	reg := prometheus.NewRegistry()
	sdMetrics, err := discovery.CreateAndRegisterSDMetrics(reg)
	require.NoError(t, err)
	// Lower the discovery manager's coalescing interval (default 5s) so targets
	// propagate quickly in tests.
	discoveryManager := discovery.NewManager(ctx, taconfig.NopLogger, reg, sdMetrics, discovery.Updatert(20*time.Millisecond))

	// Keep the production discovery path (discoverer.Run + reloader), just with a
	// short reload interval so the test does not wait on the 5s default debounce.
	// Relabel filtering now happens inside discovery (selected by the filter strategy).
	discoverer, err := target.NewDiscoverer(log, discoveryManager, target.RelabelConfigFilterStrategy, srv, alloc.SetTargets, target.WithReloadInterval(10*time.Millisecond))
	require.NoError(t, err)

	errs := make(chan error, 3)
	go func() { errs <- discoveryManager.Run() }()
	go func() { errs <- discoverer.Run() }()
	go func() { errs <- srv.Start() }()
	t.Cleanup(func() {
		cancel()
		shutdown(t, srv.Shutdown)
		discoverer.Close()
		// Drain the three Run goroutines, ignoring their expected stop sentinels.
		for range 3 {
			if err := <-errs; err != nil &&
				!errors.Is(err, context.Canceled) &&
				!errors.Is(err, http.ErrServerClosed) {
				assert.NoError(t, err)
			}
		}
	})

	// In production a collector watcher feeds the allocator from Kubernetes.
	alloc.SetCollectors(map[string]*allocation.Collector{
		collectorID: allocation.NewCollector(collectorID, ""),
	})
	require.NoError(t, discoverer.ApplyConfig(watcher.EventSourceConfigMap, scrapeConfigs))

	return &url.URL{Scheme: "http", Host: addr}
}

// startReceiver builds a real prometheus receiver configured to pull its targets
// from the allocator and push scraped metrics into sink.
func startReceiver(t *testing.T, taEndpoint *url.URL, sink *consumertest.MetricsSink) {
	t.Helper()
	endpoint := taEndpoint.String()
	factory := prometheusreceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*prometheusreceiver.Config)
	// The receiver's target_allocator config type is in an internal package we
	// cannot import, so set it via confmap rather than a struct literal.
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
		"target_allocator": map[string]any{
			"endpoint":     endpoint,
			"collector_id": collectorID,
			"interval":     "200ms",
			// Poll the allocator's HTTP SD endpoint frequently (the default refresh
			// is 30s, far longer than the test window).
			"http_sd_config": map[string]any{
				"url":              endpoint,
				"refresh_interval": "200ms",
			},
		},
	}).Unmarshal(cfg))
	// Give the receiver a fully-defaulted Prometheus config so that per-job
	// scrape-config validation has a global config to fall back on.
	pCfg, err := promconfig.Load("", taconfig.NopLogger)
	require.NoError(t, err)
	cfg.PrometheusConfig = (*prometheusreceiver.PromConfig)(pCfg)

	set := receivertest.NewNopSettings(factory.Type())
	set.Logger = zaptest.NewLogger(t)

	rcvr, err := factory.CreateMetrics(t.Context(), set, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, rcvr.Start(t.Context(), componenttest.NewNopHost()))
	t.Cleanup(func() { shutdown(t, rcvr.Shutdown) })
}

// shutdown runs a component's Shutdown with a bounded context, so a hung shutdown
// can't block test cleanup forever. A fresh context is used because t.Context() is
// already cancelled by the time cleanup runs.
func shutdown(t *testing.T, fn func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assert.NoError(t, fn(ctx))
}

func freeLocalAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// metricNames returns the set of metric names present across the sink's batches.
func metricNames(sink *consumertest.MetricsSink) map[string]struct{} {
	names := map[string]struct{}{}
	for _, md := range sink.AllMetrics() {
		rms := md.ResourceMetrics()
		for i := 0; i < rms.Len(); i++ {
			sms := rms.At(i).ScopeMetrics()
			for j := 0; j < sms.Len(); j++ {
				ms := sms.At(j).Metrics()
				for k := 0; k < ms.Len(); k++ {
					names[ms.At(k).Name()] = struct{}{}
				}
			}
		}
	}
	return names
}
