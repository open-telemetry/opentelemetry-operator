// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/shirou/gopsutil/v4/process"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

const (
	// ScopeName is the instrumentation scope used for every metric emitted by the bridge.
	ScopeName = "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge"

	exportInterval = 10 * time.Second
	shutdownWait   = 5 * time.Second

	// Attribute keys and values used by the bridge metrics.
	resultKey      = attribute.Key("result")
	operationKey   = attribute.Key("operation")
	commandTypeKey = attribute.Key("command.type")
	healthyKey     = attribute.Key("healthy")

	ResultSuccess = "success"
	ResultFailure = "failure"

	OperationApply  = "apply"
	OperationDelete = "delete"
)

// MetricReporter configures an OpenTelemetry SDK meter provider from the TelemetryConnectionSettings offered by the
// OpAMP server and reports the bridge's own metrics (process metrics and metrics about the work the bridge does) to
// the configured OTLP/HTTP destination.
//
// All Record* methods are safe to call on a nil *MetricReporter, in which case they do nothing. This lets callers
// record unconditionally regardless of whether the server has offered own-metrics connection settings.
type MetricReporter struct {
	logger   logr.Logger
	provider *sdkmetric.MeterProvider

	// The Agent's process.
	process *process.Process

	remoteConfigApplies    metric.Int64Counter
	remoteConfigDuration   metric.Float64Histogram
	configOperations       metric.Int64Counter
	commands               metric.Int64Counter
	heartbeats             metric.Int64Counter
	connectionAttempts     metric.Int64Counter
	collectors             metric.Int64Gauge
	proxyAgents            metric.Int64Gauge
	processCpuTime         metric.Float64ObservableCounter
	processMemoryPhysical  metric.Float64ObservableGauge
	processCallbackHandles metric.Registration
}

// NewMetricReporter creates a MetricReporter that exports to the OTLP/HTTP destination supplied by the OpAMP server.
func NewMetricReporter(logger logr.Logger, dest *protobufs.TelemetryConnectionSettings, agentType, agentVersion string, instanceId uuid.UUID) (*MetricReporter, error) {
	opts, err := exporterOptions(dest)
	if err != nil {
		return nil, err
	}
	exporter, err := otlpmetrichttp.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize otlp metric http exporter: %w", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(exportInterval))
	return NewMetricReporterWithReader(logger, reader, agentType, agentVersion, instanceId)
}

// NewMetricReporterWithReader creates a MetricReporter that reports through the given reader. It is mainly useful in
// tests, where a ManualReader can be used to inspect the reported metrics.
func NewMetricReporterWithReader(logger logr.Logger, reader sdkmetric.Reader, agentType, agentVersion string, instanceId uuid.UUID) (*MetricReporter, error) {
	// Define the Resource to be exported with all metrics. Use OpenTelemetry semantic
	// conventions as the OpAMP spec requires:
	// https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#own-telemetry-reporting
	resource, err := otelresource.New(context.Background(),
		otelresource.WithAttributes(
			semconv.ServiceName(agentType),
			semconv.ServiceVersion(agentVersion),
			semconv.ServiceInstanceID(instanceId.String()),
		),
	)
	if err != nil {
		return nil, err
	}

	reporter := &MetricReporter{
		logger: logger,
		provider: sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(resource),
			sdkmetric.WithReader(reader),
		),
	}
	if err = reporter.createInstruments(); err != nil {
		reporter.Shutdown()
		return nil, err
	}
	return reporter, nil
}

func (reporter *MetricReporter) createInstruments() error {
	meter := reporter.provider.Meter(ScopeName)
	var errs []error
	var err error

	reporter.remoteConfigApplies, err = meter.Int64Counter(
		"opamp.bridge.remote_config.applies",
		metric.WithDescription("Number of remote configurations received from the OpAMP server that the bridge tried to apply."),
		metric.WithUnit("{config}"),
	)
	errs = append(errs, err)

	reporter.remoteConfigDuration, err = meter.Float64Histogram(
		"opamp.bridge.remote_config.apply.duration",
		metric.WithDescription("Time taken to apply a remote configuration received from the OpAMP server."),
		metric.WithUnit("s"),
	)
	errs = append(errs, err)

	reporter.configOperations, err = meter.Int64Counter(
		"opamp.bridge.config.operations",
		metric.WithDescription("Number of apply and delete operations performed on managed collector configurations."),
		metric.WithUnit("{operation}"),
	)
	errs = append(errs, err)

	reporter.commands, err = meter.Int64Counter(
		"opamp.bridge.commands",
		metric.WithDescription("Number of commands received from the OpAMP server."),
		metric.WithUnit("{command}"),
	)
	errs = append(errs, err)

	reporter.heartbeats, err = meter.Int64Counter(
		"opamp.bridge.heartbeats",
		metric.WithDescription("Number of heartbeats sent to the OpAMP server."),
		metric.WithUnit("{heartbeat}"),
	)
	errs = append(errs, err)

	reporter.connectionAttempts, err = meter.Int64Counter(
		"opamp.bridge.connection.attempts",
		metric.WithDescription("Number of attempts to connect to the OpAMP server."),
		metric.WithUnit("{attempt}"),
	)
	errs = append(errs, err)

	reporter.collectors, err = meter.Int64Gauge(
		"opamp.bridge.collectors",
		metric.WithDescription("Number of collectors reported by the bridge, by health."),
		metric.WithUnit("{collector}"),
	)
	errs = append(errs, err)

	reporter.proxyAgents, err = meter.Int64Gauge(
		"opamp.bridge.proxy.agents",
		metric.WithDescription("Number of agents connected directly to the bridge's OpAMP proxy."),
		metric.WithUnit("{agent}"),
	)
	errs = append(errs, err)

	// Process metrics reported according to the OpenTelemetry semantic conventions for process metrics.
	reporter.process, err = process.NewProcess(int32(os.Getpid())) //nolint:gosec // this is guaranteed to not overflow
	if err != nil {
		return errors.Join(append(errs, fmt.Errorf("cannot query own process: %w", err))...)
	}

	reporter.processCpuTime, err = meter.Float64ObservableCounter(
		"process.cpu.time",
		metric.WithDescription("Total CPU seconds broken down by different states."),
		metric.WithUnit("s"),
	)
	errs = append(errs, err)

	reporter.processMemoryPhysical, err = meter.Float64ObservableGauge(
		"process.memory.physical_usage",
		metric.WithDescription("The amount of physical memory in use."),
		metric.WithUnit("By"),
	)
	errs = append(errs, err)

	if joined := errors.Join(errs...); joined != nil {
		return fmt.Errorf("failed to create metric instruments: %w", joined)
	}

	reporter.processCallbackHandles, err = meter.RegisterCallback(reporter.observeProcess, reporter.processCpuTime, reporter.processMemoryPhysical)
	if err != nil {
		return fmt.Errorf("failed to register process metrics callback: %w", err)
	}
	return nil
}

func (reporter *MetricReporter) observeProcess(ctx context.Context, observer metric.Observer) error {
	times, err := reporter.process.TimesWithContext(ctx)
	if err != nil {
		reporter.logger.Error(err, "cannot get process CPU times")
	} else {
		observer.ObserveFloat64(reporter.processCpuTime, times.User, metric.WithAttributes(attribute.String("state", "user")))
		observer.ObserveFloat64(reporter.processCpuTime, times.System, metric.WithAttributes(attribute.String("state", "system")))
		observer.ObserveFloat64(reporter.processCpuTime, times.Iowait, metric.WithAttributes(attribute.String("state", "wait")))
	}

	memory, err := reporter.process.MemoryInfoWithContext(ctx)
	if err != nil {
		reporter.logger.Error(err, "cannot get process memory information")
	} else {
		observer.ObserveFloat64(reporter.processMemoryPhysical, float64(memory.RSS))
	}
	return nil
}

// RecordRemoteConfig records that a remote configuration was applied, how long it took, and whether it failed.
func (reporter *MetricReporter) RecordRemoteConfig(ctx context.Context, duration time.Duration, err error) {
	if reporter == nil {
		return
	}
	attrs := metric.WithAttributes(resultKey.String(result(err)))
	reporter.remoteConfigApplies.Add(ctx, 1, attrs)
	reporter.remoteConfigDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordConfigOperation records a single apply or delete of a managed collector configuration.
func (reporter *MetricReporter) RecordConfigOperation(ctx context.Context, operation string, err error) {
	if reporter == nil {
		return
	}
	reporter.configOperations.Add(ctx, 1, metric.WithAttributes(operationKey.String(operation), resultKey.String(result(err))))
}

// RecordCommand records a command received from the OpAMP server.
func (reporter *MetricReporter) RecordCommand(ctx context.Context, commandType protobufs.CommandType, err error) {
	if reporter == nil {
		return
	}
	reporter.commands.Add(ctx, 1, metric.WithAttributes(commandTypeKey.String(commandType.String()), resultKey.String(result(err))))
}

// RecordHeartbeat records a heartbeat sent to the OpAMP server.
func (reporter *MetricReporter) RecordHeartbeat(ctx context.Context, err error) {
	if reporter == nil {
		return
	}
	reporter.heartbeats.Add(ctx, 1, metric.WithAttributes(resultKey.String(result(err))))
}

// RecordConnectionAttempt records an attempt to connect to the OpAMP server.
func (reporter *MetricReporter) RecordConnectionAttempt(ctx context.Context, err error) {
	if reporter == nil {
		return
	}
	reporter.connectionAttempts.Add(ctx, 1, metric.WithAttributes(resultKey.String(result(err))))
}

// RecordCollectors records how many of the collectors reported by the bridge are healthy and unhealthy.
func (reporter *MetricReporter) RecordCollectors(ctx context.Context, healthy, unhealthy int64) {
	if reporter == nil {
		return
	}
	reporter.collectors.Record(ctx, healthy, metric.WithAttributes(healthyKey.Bool(true)))
	reporter.collectors.Record(ctx, unhealthy, metric.WithAttributes(healthyKey.Bool(false)))
}

// RecordProxyAgents records how many agents are connected directly to the bridge's OpAMP proxy.
func (reporter *MetricReporter) RecordProxyAgents(ctx context.Context, count int64) {
	if reporter == nil {
		return
	}
	reporter.proxyAgents.Record(ctx, count)
}

// Shutdown flushes any pending metrics and stops the reporter.
func (reporter *MetricReporter) Shutdown() {
	if reporter == nil {
		return
	}
	if reporter.processCallbackHandles != nil {
		if err := reporter.processCallbackHandles.Unregister(); err != nil {
			reporter.logger.Error(err, "failed to unregister process metrics callback")
		}
	}
	if reporter.provider != nil {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownWait)
		defer cancel()
		if err := reporter.provider.Shutdown(ctx); err != nil {
			reporter.logger.Error(err, "failed to shut down metric reporter")
		}
	}
}

func result(err error) string {
	if err != nil {
		return ResultFailure
	}
	return ResultSuccess
}
