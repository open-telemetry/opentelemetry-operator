// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
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

	if cfg.Telemetry.Metrics != nil {
		for _, reader := range cfg.Telemetry.Metrics.Readers {
			if reader.Periodic == nil {
				continue
			}
			otlpReader, otlpErr := newOTLPMetricReader(ctx, reader.Periodic)
			if otlpErr != nil {
				return nil, nil, otlpErr
			}
			res, resErr := telemetryResource(ctx)
			if resErr != nil {
				return nil, nil, resErr
			}
			meterProviderOpts = append(meterProviderOpts, sdkmetric.WithReader(otlpReader), sdkmetric.WithResource(res))
		}
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

// expandHeaders returns a map of expanded header values from a slice of NameValuePairs,
// with ${env:VAR} references substituted in each value.
func expandHeaders(headers []config.NameValuePair) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	expanded := make(map[string]string, len(headers))
	for _, h := range headers {
		if h.Value == nil {
			continue
		}
		expanded[h.Name] = expandEnvRefs(*h.Value)
	}
	return expanded
}

// newOTLPMetricReader creates a PeriodicExportingMetricReader from a PeriodicMetricReader
// config. It dispatches to gRPC or HTTP based on which exporter is configured.
// Exactly one of otlp_grpc or otlp_http must be set; validation in validateTelemetry
// enforces this before we reach here.
func newOTLPMetricReader(ctx context.Context, cfg *config.PeriodicMetricReader) (sdkmetric.Reader, error) {
	readerOpts := periodicReaderOptions(cfg)
	exp := cfg.Exporter
	if exp.OTLPGrpc != nil {
		return newGRPCMetricReader(ctx, exp.OTLPGrpc, readerOpts)
	}
	if exp.OTLPHttp != nil {
		return newHTTPMetricReader(ctx, exp.OTLPHttp, readerOpts)
	}
	return nil, errors.New("periodic metric reader: must configure otlp_grpc or otlp_http exporter")
}

// newGRPCMetricReader creates a PeriodicExportingMetricReader backed by an OTLP/gRPC exporter.
func newGRPCMetricReader(ctx context.Context, cfg *config.OTLPGrpcExporterConfig, readerOpts []sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, error) {
	temporality := temporalitySelector(cfg.TemporalityPreference)
	headers := expandHeaders(cfg.Headers)
	endpoint := expandEnvRefs(cfg.Endpoint)

	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithTemporalitySelector(temporality),
	}
	// Accept both a full URL (with scheme) and a bare host:port.
	// WithEndpointURL requires a scheme; WithEndpoint expects host:port.
	if strings.Contains(endpoint, "://") {
		opts = append(opts, otlpmetricgrpc.WithEndpointURL(endpoint))
	} else if endpoint != "" {
		opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint))
	}
	if cfg.Tls != nil && cfg.Tls.Insecure {
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

// newHTTPMetricReader creates a PeriodicExportingMetricReader backed by an OTLP/HTTP exporter.
// The endpoint URL scheme (http:// vs https://) determines whether TLS is used.
// /v1/metrics is appended automatically unless the endpoint already includes it.
func newHTTPMetricReader(ctx context.Context, cfg *config.OTLPHttpExporterConfig, readerOpts []sdkmetric.PeriodicReaderOption) (sdkmetric.Reader, error) {
	temporality := temporalitySelector(cfg.TemporalityPreference)
	headers := expandHeaders(cfg.Headers)
	endpoint := expandEnvRefs(cfg.Endpoint)

	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(endpoint, "/v1/metrics") {
		endpoint += "/v1/metrics"
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(endpoint),
		otlpmetrichttp.WithTemporalitySelector(temporality),
	}
	if len(headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(headers))
	}
	exp, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return sdkmetric.NewPeriodicReader(exp, readerOpts...), nil
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
// cumulative sums. This wrapper converts them to delta so that backends
// configured to expect delta temporality receive the correct data.
//
// The prev map grows to one entry per unique {scope, metric, attribute-set}
// combination that has ever been observed. For a stable Prometheus scrape target
// (fixed label cardinality, no ephemeral label values such as pod IPs) this set
// is bounded in practice. High-cardinality label churn would cause unbounded
// growth; in that case the Prometheus scrape target itself would already be a
// problem, so this is an acceptable tradeoff for the current use case.
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
					// First observation: emit with delta = current value so the
					// backend sees the counter from the start rather than skipping it.
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

// periodicReaderOptions builds PeriodicReaderOptions from the interval and timeout config.
// Interval and Timeout are in milliseconds, matching the otelconf spec.
func periodicReaderOptions(cfg *config.PeriodicMetricReader) []sdkmetric.PeriodicReaderOption {
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
		// output. deltaProducer wraps the bridge and converts cumulative monotonic
		// sums to delta for backends configured to expect delta temporality.
		sdkmetric.WithProducer(newDeltaProducer(prometheusbridge.NewMetricProducer())),
	}
	if cfg.Interval > 0 {
		opts = append(opts, sdkmetric.WithInterval(time.Duration(cfg.Interval)*time.Millisecond))
	}
	if cfg.Timeout > 0 {
		opts = append(opts, sdkmetric.WithTimeout(time.Duration(cfg.Timeout)*time.Millisecond))
	}
	return opts
}

// telemetryResource builds the OpenTelemetry resource attached to exported metrics.
// It sets service.name ("target-allocator") and service.instance.id (pod hostname)
// so that metrics from different TA replicas can be distinguished in a backend.
// The standard OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES environment variables
// are honored and win on conflict, allowing additional attributes (e.g. k8s.*) to
// be injected without code changes.
func telemetryResource(ctx context.Context) (*resource.Resource, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName("target-allocator"),
			semconv.ServiceInstanceID(hostname),
		),
		// Applied last so environment configuration wins on conflict.
		resource.WithFromEnv(),
	)
}

// temporalitySelector maps a config string to an SDK TemporalitySelector.
// Accepts otelconf-compatible values: "delta", "low_memory", "cumulative" (or empty for default).
func temporalitySelector(t string) sdkmetric.TemporalitySelector {
	switch t {
	case "delta":
		return sdkmetric.DeltaTemporalitySelector
	case "low_memory":
		return sdkmetric.LowMemoryTemporalitySelector
	default:
		return sdkmetric.DefaultTemporalitySelector
	}
}
