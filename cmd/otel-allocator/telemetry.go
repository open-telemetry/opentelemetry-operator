// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	prometheusbridge "go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

// setupMeterProvider builds the Target Allocator's meter provider, sets it as the global
// provider, and returns the Prometheus gatherer to serve on /metrics plus a shutdown func.
//
// The OTel SDK metrics are exported to Prometheus through a dedicated registry rather than
// the default one. This keeps them separate from the metrics that are registered directly on
// the default Prometheus registry (Prometheus service discovery internals, Go runtime and
// process collectors). That separation lets the OTLP reader pull those Prometheus-only metrics
// via the Prometheus bridge without double-counting the SDK metrics, which it already collects
// natively. The /metrics endpoint serves the union of both registries.
func setupMeterProvider(ctx context.Context, cfg *config.Config) (prometheus.Gatherer, func(context.Context) error, error) {
	sdkRegistry := prometheus.NewRegistry()
	metricExporter, err := otelprom.New(otelprom.WithRegisterer(sdkRegistry))
	if err != nil {
		return nil, nil, err
	}
	meterProviderOpts := []sdkmetric.Option{sdkmetric.WithReader(metricExporter)}

	if cfg.Telemetry.Metrics.OTLP != nil {
		otlpReader, otlpErr := newOTLPMetricReader(ctx, cfg.Telemetry.Metrics.OTLP)
		if otlpErr != nil {
			return nil, nil, otlpErr
		}
		res, resErr := telemetryResource(ctx)
		if resErr != nil {
			return nil, nil, resErr
		}
		meterProviderOpts = append(meterProviderOpts, sdkmetric.WithReader(otlpReader), sdkmetric.WithResource(res))
	}

	meterProvider := sdkmetric.NewMeterProvider(meterProviderOpts...)
	otel.SetMeterProvider(meterProvider)

	gatherer := prometheus.Gatherers{prometheus.DefaultGatherer, sdkRegistry}
	return gatherer, meterProvider.Shutdown, nil
}

// envRefPattern matches ${env:VAR} references, mirroring the OTel Collector's
// environment-variable substitution syntax.
var envRefPattern = regexp.MustCompile(`\$\{env:([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// expandEnvRefs replaces ${env:VAR} occurrences in s with the value of the
// environment variable VAR (empty if unset). This lets sensitive header values
// (e.g. an API token) be sourced from the pod environment / a Secret at runtime
// rather than being written in plaintext into the Target Allocator ConfigMap.
func expandEnvRefs(s string) string {
	return envRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := envRefPattern.FindStringSubmatch(match)[1]
		return os.Getenv(name)
	})
}

// expandHeaders returns a copy of headers with ${env:VAR} references expanded in
// every value.
func expandHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	expanded := make(map[string]string, len(headers))
	for k, v := range headers {
		expanded[k] = expandEnvRefs(v)
	}
	return expanded
}

// newOTLPMetricReader creates a PeriodicExportingMetricReader backed by either
// an OTLP/gRPC or OTLP/HTTP exporter depending on cfg.Protocol.
func newOTLPMetricReader(ctx context.Context, cfg *config.OTLPExporterConfig) (sdkmetric.Reader, error) {
	temporality := temporalitySelector(cfg.Temporality)
	readerOpts := periodicReaderOptions(cfg)
	endpoint := expandEnvRefs(cfg.Endpoint)
	headers := expandHeaders(cfg.Headers)
	switch cfg.Protocol {
	case "http":
		// WithEndpointURL does not append /v1/metrics automatically. We do it here
		// so the endpoint convention matches the OTel collector's otlphttp exporter
		// (where users specify the base URL, not the full signal path).
		// Skip if the caller already included the signal path.
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasSuffix(endpoint, "/v1/metrics") {
			endpoint += "/v1/metrics"
		}
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpointURL(endpoint),
			otlpmetrichttp.WithTemporalitySelector(temporality),
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(headers))
		}
		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, readerOpts...), nil
	default: // "grpc"
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithTemporalitySelector(temporality),
		}
		// Accept both a full URL (with scheme, consistent with the HTTP protocol and
		// the collector's otlp exporter) and a bare host:port. WithEndpointURL requires
		// a scheme; WithEndpoint expects host:port.
		if strings.Contains(endpoint, "://") {
			opts = append(opts, otlpmetricgrpc.WithEndpointURL(endpoint))
		} else {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint))
		}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
		}
		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return sdkmetric.NewPeriodicReader(exp, readerOpts...), nil
	}
}

// deltaSumKey identifies a single data point for cumulative→delta tracking.
type deltaSumKey struct {
	scope  string
	metric string
	attrs  string
}

// prevSumPoint stores the last observed cumulative value and its timestamp.
type prevSumPoint struct {
	value float64
	t     time.Time
}

// deltaProducer wraps a metric.Producer and converts cumulative monotonic Sum
// data points to delta. The OTel SDK's PeriodicReader only applies the
// TemporalitySelector to SDK-native instruments; external Producer output is
// forwarded as-is. For the Prometheus bridge this means counters arrive as
// cumulative sums, which DT rejects. This wrapper fixes that gap.
type deltaProducer struct {
	inner sdkmetric.Producer
	mu    sync.Mutex
	prev  map[deltaSumKey]prevSumPoint
}

func newDeltaProducer(inner sdkmetric.Producer) sdkmetric.Producer {
	return &deltaProducer{inner: inner, prev: make(map[deltaSumKey]prevSumPoint)}
}

func (d *deltaProducer) Produce(ctx context.Context) ([]metricdata.ScopeMetrics, error) {
	sms, err := d.inner.Produce(ctx)

	d.mu.Lock()
	defer d.mu.Unlock()

	for si := range sms {
		scope := sms[si].Scope.Name
		for mi := range sms[si].Metrics {
			m := &sms[si].Metrics[mi]
			sum, ok := m.Data.(metricdata.Sum[float64])
			if !ok || !sum.IsMonotonic || sum.Temporality != metricdata.CumulativeTemporality {
				continue
			}
			deltaDPs := make([]metricdata.DataPoint[float64], 0, len(sum.DataPoints))
			for _, dp := range sum.DataPoints {
				key := deltaSumKey{
					scope:  scope,
					metric: m.Name,
					attrs:  fmt.Sprintf("%v", dp.Attributes),
				}
				prev, seen := d.prev[key]
				d.prev[key] = prevSumPoint{value: dp.Value, t: dp.Time}
				if !seen {
					// First observation: emit with delta = current value so DT
					// sees the counter from the start rather than skipping it.
					startTime := dp.StartTime
					if startTime.IsZero() {
						startTime = dp.Time
					}
					deltaDPs = append(deltaDPs, metricdata.DataPoint[float64]{
						Attributes: dp.Attributes,
						StartTime:  startTime,
						Time:       dp.Time,
						Value:      dp.Value,
						Exemplars:  dp.Exemplars,
					})
					continue
				}
				delta := dp.Value - prev.value
				if delta < 0 {
					// Counter reset: emit the new value as the delta.
					delta = dp.Value
				}
				deltaDPs = append(deltaDPs, metricdata.DataPoint[float64]{
					Attributes: dp.Attributes,
					StartTime:  prev.t,
					Time:       dp.Time,
					Value:      delta,
					Exemplars:  dp.Exemplars,
				})
			}
			sum.Temporality = metricdata.DeltaTemporality
			sum.DataPoints = deltaDPs
			m.Data = sum
		}
	}

	return sms, err
}

// periodicReaderOptions builds PeriodicReaderOptions from the export interval and timeout config.
func periodicReaderOptions(cfg *config.OTLPExporterConfig) []sdkmetric.PeriodicReaderOption {
	opts := []sdkmetric.PeriodicReaderOption{
		// Bridge the metrics registered directly on the default Prometheus registry
		// (Prometheus service discovery internals, Go runtime and process collectors)
		// into the OTLP export. Without this, those metrics would be visible on the
		// Prometheus /metrics endpoint but missing from OTLP. The SDK's own metrics live
		// on a separate registry and are collected natively by this reader, so bridging
		// the default registry does not double-count them.
		//
		// The bridge always returns CumulativeTemporality for counters, but the
		// PeriodicReader does not apply the TemporalitySelector to external Producer
		// output. deltaProducer wraps the bridge and performs the conversion so DT
		// receives delta monotonic sums (cumulative is rejected by DT).
		sdkmetric.WithProducer(newDeltaProducer(prometheusbridge.NewMetricProducer())),
	}
	if cfg.ExportInterval > 0 {
		opts = append(opts, sdkmetric.WithInterval(cfg.ExportInterval))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, sdkmetric.WithTimeout(cfg.Timeout))
	}
	return opts
}

// telemetryResource builds the OpenTelemetry resource attached to exported metrics.
// It sets a default service.name and honors the standard OTEL_SERVICE_NAME /
// OTEL_RESOURCE_ATTRIBUTES environment variables, so additional attributes (e.g. k8s.*)
// can be injected without code changes.
func telemetryResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName("target-allocator")),
		// Applied last so environment configuration wins on conflict.
		resource.WithFromEnv(),
	)
}

// temporalitySelector maps a config string to an SDK TemporalitySelector.
// "delta" → all instruments delta; "lowmemory" → delta for counters/histograms,
// cumulative for gauges; anything else → cumulative (SDK default).
func temporalitySelector(t string) sdkmetric.TemporalitySelector {
	switch t {
	case "delta":
		return sdkmetric.DeltaTemporalitySelector
	case "lowmemory":
		return sdkmetric.LowMemoryTemporalitySelector
	default:
		return sdkmetric.DefaultTemporalitySelector
	}
}
