package metrics

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/shirou/gopsutil/process"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// MetricReporter is a metric reporter that collects Agent metrics and sends them to an
// OTLP/HTTP destination.
type MetricReporter struct {
	logger types.Logger

	meter           metric.Meter
	meterShutdowner func()
	done            chan struct{}

	// The Agent's process.
	process *process.Process

	// Some example metrics to report.
	processMemoryPhysical asyncint64.Gauge
	counter               syncint64.Counter
	processCpuTime        asyncfloat64.Counter
}

func NewMetricReporter(
	logger types.Logger,
	dest *protobufs.TelemetryConnectionSettings,
	agentType string,
	agentVersion string,
	instanceId ulid.ULID,
) (*MetricReporter, error) {

	// Check the destination credentials to make sure they look like a valid OTLP/HTTP
	// destination.

	if dest.DestinationEndpoint == "" {
		err := fmt.Errorf("metric destination must specify DestinationEndpoint")
		return nil, err
	}
	u, err := url.Parse(dest.DestinationEndpoint)
	if err != nil {
		err := fmt.Errorf("invalid DestinationEndpoint: %v", err)
		return nil, err
	}

	// Create OTLP/HTTP metric exporter.
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(u.Host),
		otlpmetrichttp.WithURLPath(u.Path),
	}

	if u.Scheme == "http" {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	client, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		err := fmt.Errorf("failed to initialize otlp metric http client: %v", err)
		return nil, err
	}

	// Define the Resource to be exported with all metrics. Use OpenTelemetry semantic
	// conventions as the OpAMP spec requires:
	// https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#own-telemetry-reporting
	resource, err := otelresource.New(context.Background(),
		otelresource.WithAttributes(
			semconv.ServiceNameKey.String(agentType),
			semconv.ServiceVersionKey.String(agentVersion),
			semconv.ServiceInstanceIDKey.String(instanceId.String()),
		),
	)

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(client)))

	global.SetMeterProvider(provider)

	reporter := &MetricReporter{
		logger: logger,
	}

	reporter.done = make(chan struct{})

	reporter.meter = global.Meter("opamp")

	reporter.process, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		err := fmt.Errorf("cannot query own process: %v", err)
		return nil, err
	}

	// Create some metrics that will be reported according to OpenTelemetry semantic
	// conventions for process metrics (conventions are TBD for now).
	reporter.processCpuTime, err = reporter.meter.AsyncFloat64().Counter(
		"process.cpu.time",
	)
	if err != nil {
		err := fmt.Errorf("can't create process time metric: %v", err)
		return nil, err
	}
	err = reporter.meter.RegisterCallback([]instrument.Asynchronous{reporter.processCpuTime}, reporter.processCpuTimeFunc)
	if err != nil {
		err := fmt.Errorf("can't create register callback: %v", err)
		return nil, err
	}
	reporter.processMemoryPhysical, err = reporter.meter.AsyncInt64().Gauge(
		"process.memory.physical_usage",
	)
	if err != nil {
		err := fmt.Errorf("can't create memory metric: %v", err)
		return nil, err
	}
	err = reporter.meter.RegisterCallback([]instrument.Asynchronous{reporter.processMemoryPhysical}, reporter.processMemoryPhysicalFunc)
	if err != nil {
		err := fmt.Errorf("can't register callback: %v", err)
		return nil, err
	}

	reporter.counter, err = reporter.meter.SyncInt64().Counter("custom_metric_ticks")
	if err != nil {
		err := fmt.Errorf("can't register counter metric: %v", err)
		return nil, err
	}

	reporter.meterShutdowner = func() { _ = provider.Shutdown(context.Background()) }

	go reporter.sendMetrics()

	return reporter, nil
}

func (reporter *MetricReporter) processCpuTimeFunc(c context.Context) {
	times, err := reporter.process.Times()
	if err != nil {
		reporter.logger.Errorf("Cannot get process CPU times: %v", err)
	}

	// Report process CPU times, but also add some randomness to make it interesting for demo.
	reporter.processCpuTime.Observe(c, math.Min(times.User+rand.Float64(), 1), attribute.String("state", "user"))
	reporter.processCpuTime.Observe(c, math.Min(times.System+rand.Float64(), 1), attribute.String("state", "system"))
	reporter.processCpuTime.Observe(c, math.Min(times.Iowait+rand.Float64(), 1), attribute.String("state", "wait"))
}

func (reporter *MetricReporter) processMemoryPhysicalFunc(ctx context.Context) {
	memory, err := reporter.process.MemoryInfo()
	if err != nil {
		reporter.logger.Errorf("Cannot get process memory information: %v", err)
		return
	}

	// Report the RSS, but also add some randomness to make it interesting for demo.
	reporter.processMemoryPhysical.Observe(ctx, int64(memory.RSS)+rand.Int63n(10000000))
}

func (reporter *MetricReporter) sendMetrics() {

	// Collect metrics every 5 seconds.
	t := time.NewTicker(time.Second * 5)
	ticks := int64(0)

	for {
		select {
		case <-reporter.done:
			return

		case <-t.C:
			ctx := context.Background()
			reporter.counter.Add(ctx, ticks)
			ticks++
		}
	}
}

func (reporter *MetricReporter) Shutdown() {
	if reporter.done != nil {
		close(reporter.done)
	}

	if reporter.meterShutdowner != nil {
		reporter.meterShutdowner()
	}
}
