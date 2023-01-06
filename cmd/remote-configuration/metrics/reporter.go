// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/shirou/gopsutil/process"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
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
	processCpuTime        asyncfloat64.Counter
}

// NewMetricReporter creates an OTLP gRPC client to the destination address supplied by the server.
// TODO: do more validation on the endpoint, allow for http.
// TODO: set global provider and add more metrics to be reported.
func NewMetricReporter(
	logger types.Logger,
	dest *protobufs.TelemetryConnectionSettings,
	agentType string,
	agentVersion string,
	instanceId ulid.ULID,
) (*MetricReporter, error) {

	if dest.DestinationEndpoint == "" {
		err := fmt.Errorf("metric destination must specify DestinationEndpoint")
		return nil, err
	}

	// Create OTLP/grpc metric exporter.
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(dest.DestinationEndpoint),
	}

	headers := map[string]string{}
	for _, header := range dest.Headers.GetHeaders() {
		headers[header.GetKey()] = header.GetValue()
	}
	opts = append(opts, otlpmetricgrpc.WithHeaders(headers))

	client, err := otlpmetricgrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize otlp metric grpc client: %w", err)
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

	reporter.process, err = process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, fmt.Errorf("cannot query own process: %w", err)
	}

	// Create some metrics that will be reported according to OpenTelemetry semantic
	// conventions for process metrics (conventions are TBD for now).
	reporter.processCpuTime, err = reporter.meter.AsyncFloat64().Counter(
		"process.cpu.time",
	)
	if err != nil {
		return nil, fmt.Errorf("can't create process time metric: %w", err)
	}
	err = reporter.meter.RegisterCallback([]instrument.Asynchronous{reporter.processCpuTime}, reporter.processCpuTimeFunc)
	if err != nil {
		return nil, fmt.Errorf("can't create register callback: %w", err)
	}
	reporter.processMemoryPhysical, err = reporter.meter.AsyncInt64().Gauge(
		"process.memory.physical_usage",
	)
	if err != nil {
		return nil, fmt.Errorf("can't create memory metric: %w", err)
	}
	err = reporter.meter.RegisterCallback([]instrument.Asynchronous{reporter.processMemoryPhysical}, reporter.processMemoryPhysicalFunc)
	if err != nil {
		return nil, fmt.Errorf("can't register callback: %w", err)
	}

	reporter.meterShutdowner = func() { _ = provider.Shutdown(context.Background()) }

	return reporter, nil
}

func (reporter *MetricReporter) processCpuTimeFunc(c context.Context) {
	times, err := reporter.process.Times()
	if err != nil {
		reporter.logger.Errorf("Cannot get process CPU times: %w", err)
	}
	reporter.processCpuTime.Observe(c, times.User, attribute.String("state", "user"))
	reporter.processCpuTime.Observe(c, times.System, attribute.String("state", "system"))
	reporter.processCpuTime.Observe(c, times.Iowait, attribute.String("state", "wait"))
}

func (reporter *MetricReporter) processMemoryPhysicalFunc(ctx context.Context) {
	memory, err := reporter.process.MemoryInfo()
	if err != nil {
		reporter.logger.Errorf("Cannot get process memory information: %w", err)
		return
	}
	reporter.processMemoryPhysical.Observe(ctx, int64(memory.RSS))
}

func (reporter *MetricReporter) Shutdown() {
	if reporter.done != nil {
		close(reporter.done)
	}

	if reporter.meterShutdowner != nil {
		reporter.meterShutdowner()
	}
}
