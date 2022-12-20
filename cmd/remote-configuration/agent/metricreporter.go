package agent

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
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
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
	processMemoryPhysical metric.Int64GaugeObserver
	counter               metric.Int64Counter
	processCpuTime        metric.Float64CounterObserver
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

	client := otlpmetrichttp.NewClient(opts...)

	metricExporter, err := otlpmetric.New(context.Background(), client)
	if err != nil {
		err := fmt.Errorf("failed to initialize stdoutmetric export pipeline: %v", err)
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

	// Wire up the Resource and the exporter together.
	cont := controller.New(
		processor.NewFactory(
			simple.NewWithInexpensiveDistribution(),
			metricExporter,
		),
		controller.WithExporter(metricExporter),
		controller.WithCollectPeriod(5*time.Second),
		controller.WithResource(resource),
	)

	err = cont.Start(context.Background())
	if err != nil {
		err := fmt.Errorf("failed to initialize metric controller: %v", err)
		return nil, err
	}

	global.SetMeterProvider(cont)

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
	reporter.processCpuTime = metric.Must(reporter.meter).NewFloat64CounterObserver(
		"process.cpu.time",
		reporter.processCpuTimeFunc,
	)

	reporter.processMemoryPhysical = metric.Must(reporter.meter).NewInt64GaugeObserver(
		"process.memory.physical_usage",
		reporter.processMemoryPhysicalFunc,
	)

	reporter.counter = metric.Must(reporter.meter).NewInt64Counter("custom_metric_ticks")

	reporter.meterShutdowner = func() { _ = cont.Stop(context.Background()) }

	go reporter.sendMetrics()

	return reporter, nil
}

func (reporter *MetricReporter) processCpuTimeFunc(_ context.Context, result metric.Float64ObserverResult) {
	times, err := reporter.process.Times()
	if err != nil {
		reporter.logger.Errorf("Cannot get process CPU times: %v", err)
	}

	// Report process CPU times, but also add some randomness to make it interesting for demo.
	result.Observe(math.Min(times.User+rand.Float64(), 1), attribute.String("state", "user"))
	result.Observe(math.Min(times.System+rand.Float64(), 1), attribute.String("state", "system"))
	result.Observe(math.Min(times.Iowait+rand.Float64(), 1), attribute.String("state", "wait"))
}

func (reporter *MetricReporter) processMemoryPhysicalFunc(_ context.Context, result metric.Int64ObserverResult) {
	memory, err := reporter.process.MemoryInfo()
	if err != nil {
		reporter.logger.Errorf("Cannot get process memory information: %v", err)
		return
	}

	// Report the RSS, but also add some randomness to make it interesting for demo.
	result.Observe(int64(memory.RSS) + rand.Int63n(10000000))
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
			reporter.meter.RecordBatch(
				ctx,
				[]attribute.KeyValue{},
				reporter.counter.Measurement(ticks),
			)
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
