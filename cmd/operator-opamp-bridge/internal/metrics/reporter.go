// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/shirou/gopsutil/process"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// MetricReporter is a metric reporter that collects Agent metrics and sends them to an
// OTLP/HTTP destination.
type MetricReporter struct {
	logger logr.Logger

	meter           metric.Meter
	meterShutdowner func()
	done            chan struct{}

	// The Agent's process.
	process *process.Process

	// Some example metrics to report.
	processMemoryPhysical metric.Float64ObservableGauge
	processCpuTime        metric.Float64ObservableCounter
}

// NewMetricReporter creates an OTLP/HTTP client to the destination address supplied by the server.
// TODO: do more validation on the endpoint, allow for gRPC.
// TODO: set global provider and add more metrics to be reported.
func NewMetricReporter(logger logr.Logger, dest *protobufs.TelemetryConnectionSettings, agentType string, agentVersion string, instanceId uuid.UUID) (*MetricReporter, error) {

	if dest.DestinationEndpoint == "" {
		return nil, fmt.Errorf("metric destination must specify DestinationEndpoint")
	}

	u, err := url.Parse(dest.DestinationEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid DestinationEndpoint: %w", err)
	}

	// Create OTLP/HTTP metric exporter.
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(u.Host),
		otlpmetrichttp.WithURLPath(u.Path),
	}

	headers := map[string]string{}
	for _, header := range dest.Headers.GetHeaders() {
		headers[header.GetKey()] = header.GetValue()
	}
	opts = append(opts, otlpmetrichttp.WithHeaders(headers))

	client, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize otlp metric http client: %w", err)
	}

	// Define the Resource to be exported with all metrics. Use OpenTelemetry semantic
	// conventions as the OpAMP spec requires:
	// https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#own-telemetry-reporting
	resource, resourceErr := otelresource.New(context.Background(),
		otelresource.WithAttributes(
			semconv.ServiceNameKey.String(agentType),
			semconv.ServiceVersionKey.String(agentVersion),
			semconv.ServiceInstanceIDKey.String(instanceId.String()),
		),
	)
	if resourceErr != nil {
		return nil, resourceErr
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(client, sdkmetric.WithInterval(5*time.Second))))

	reporter := &MetricReporter{
		logger: logger,
	}

	reporter.done = make(chan struct{})

	reporter.meter = provider.Meter("opamp")

	reporter.process, err = process.NewProcess(int32(os.Getpid())) //nolint: gosec // this is guaranteed to not overflow
	if err != nil {
		return nil, fmt.Errorf("cannot query own process: %w", err)
	}

	// Create some metrics that will be reported according to OpenTelemetry semantic
	// conventions for process metrics (conventions are TBD for now).
	reporter.processCpuTime, err = reporter.meter.Float64ObservableCounter(
		"process.cpu.time",
		metric.WithFloat64Callback(reporter.processCpuTimeFunc),
	)
	if err != nil {
		return nil, fmt.Errorf("can't create process time metric: %w", err)
	}

	reporter.processMemoryPhysical, err = reporter.meter.Float64ObservableGauge(
		"process.memory.physical_usage",
		metric.WithFloat64Callback(reporter.processMemoryPhysicalFunc),
	)
	if err != nil {
		return nil, fmt.Errorf("can't create memory metric: %w", err)
	}

	reporter.meterShutdowner = func() { _ = provider.Shutdown(context.Background()) }

	return reporter, nil
}

func (reporter *MetricReporter) processCpuTimeFunc(_ context.Context, observer metric.Float64Observer) error {
	times, err := reporter.process.Times()
	if err != nil {
		reporter.logger.Error(err, "cannot get process CPU times")
	}
	observer.Observe(times.User, metric.WithAttributes(attribute.String("state", "user")))
	observer.Observe(times.System, metric.WithAttributes(attribute.String("state", "system")))
	observer.Observe(times.Iowait, metric.WithAttributes(attribute.String("state", "wait")))
	return nil
}

func (reporter *MetricReporter) processMemoryPhysicalFunc(_ context.Context, observer metric.Float64Observer) error {
	memory, err := reporter.process.MemoryInfo()
	if err != nil {
		reporter.logger.Error(err, "cannot get process memory information")
		return nil
	}
	observer.Observe(float64(memory.RSS))
	return nil
}

func (reporter *MetricReporter) Shutdown() {
	if reporter.done != nil {
		close(reporter.done)
	}

	if reporter.meterShutdowner != nil {
		reporter.meterShutdowner()
	}
}
