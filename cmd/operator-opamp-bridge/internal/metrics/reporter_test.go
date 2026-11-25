// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func collect(t *testing.T, reader sdkmetric.Reader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))
	return rm
}

func findMetric(rm metricdata.ResourceMetrics, name string) (metricdata.Metrics, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m, true
			}
		}
	}
	return metricdata.Metrics{}, false
}

// int64Value returns the value of the int64 sum or gauge data point with exactly the given attributes.
func int64Value(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs ...attribute.KeyValue) int64 {
	t.Helper()
	m, ok := findMetric(rm, name)
	require.True(t, ok, "metric %s not found", name)
	want := attribute.NewSet(attrs...)
	var points []metricdata.DataPoint[int64]
	switch data := m.Data.(type) {
	case metricdata.Sum[int64]:
		points = data.DataPoints
	case metricdata.Gauge[int64]:
		points = data.DataPoints
	default:
		t.Fatalf("metric %s has unexpected type %T", name, m.Data)
	}
	for _, dp := range points {
		if dp.Attributes.Equals(&want) {
			return dp.Value
		}
	}
	t.Fatalf("metric %s has no data point with attributes %v", name, attrs)
	return 0
}

func newTestReporter(t *testing.T) (*MetricReporter, *sdkmetric.ManualReader, uuid.UUID) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	id := uuid.New()
	reporter, err := NewMetricReporterWithReader(logr.Discard(), reader, "test-agent", "v1.2.3", id)
	require.NoError(t, err)
	t.Cleanup(reporter.Shutdown)
	return reporter, reader, id
}

func TestMetricReporter_Resource(t *testing.T) {
	_, reader, id := newTestReporter(t)
	rm := collect(t, reader)

	name, ok := rm.Resource.Set().Value(semconv.ServiceNameKey)
	require.True(t, ok)
	assert.Equal(t, "test-agent", name.AsString())
	version, ok := rm.Resource.Set().Value(semconv.ServiceVersionKey)
	require.True(t, ok)
	assert.Equal(t, "v1.2.3", version.AsString())
	instance, ok := rm.Resource.Set().Value(semconv.ServiceInstanceIDKey)
	require.True(t, ok)
	assert.Equal(t, id.String(), instance.AsString())
}

func TestMetricReporter_RecordsBridgeWork(t *testing.T) {
	reporter, reader, _ := newTestReporter(t)
	ctx := t.Context()
	failure := errors.New("boom")

	reporter.RecordRemoteConfig(ctx, 250*time.Millisecond, nil)
	reporter.RecordRemoteConfig(ctx, time.Second, failure)
	reporter.RecordConfigOperation(ctx, OperationApply, nil)
	reporter.RecordConfigOperation(ctx, OperationApply, nil)
	reporter.RecordConfigOperation(ctx, OperationDelete, failure)
	reporter.RecordCommand(ctx, protobufs.CommandType_CommandType_Restart, nil)
	reporter.RecordHeartbeat(ctx, nil)
	reporter.RecordHeartbeat(ctx, failure)
	reporter.RecordConnectionAttempt(ctx, failure)
	reporter.RecordConnectionAttempt(ctx, nil)
	reporter.RecordCollectors(ctx, 3, 1)
	reporter.RecordProxyAgents(ctx, 2)

	rm := collect(t, reader)

	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.remote_config.applies", resultKey.String(ResultSuccess)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.remote_config.applies", resultKey.String(ResultFailure)))
	assert.Equal(t, int64(2), int64Value(t, rm, "opamp.bridge.config.operations", operationKey.String(OperationApply), resultKey.String(ResultSuccess)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.config.operations", operationKey.String(OperationDelete), resultKey.String(ResultFailure)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.commands", commandTypeKey.String("CommandType_Restart"), resultKey.String(ResultSuccess)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.heartbeats", resultKey.String(ResultSuccess)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.heartbeats", resultKey.String(ResultFailure)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.connection.attempts", resultKey.String(ResultSuccess)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.connection.attempts", resultKey.String(ResultFailure)))
	assert.Equal(t, int64(3), int64Value(t, rm, "opamp.bridge.collectors", healthyKey.Bool(true)))
	assert.Equal(t, int64(1), int64Value(t, rm, "opamp.bridge.collectors", healthyKey.Bool(false)))
	assert.Equal(t, int64(2), int64Value(t, rm, "opamp.bridge.proxy.agents"))

	duration, ok := findMetric(rm, "opamp.bridge.remote_config.apply.duration")
	require.True(t, ok)
	histogram, ok := duration.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 2)
	var total uint64
	for _, dp := range histogram.DataPoints {
		total += dp.Count
	}
	assert.Equal(t, uint64(2), total)
}

func TestMetricReporter_ReportsProcessMetrics(t *testing.T) {
	_, reader, _ := newTestReporter(t)
	rm := collect(t, reader)

	_, ok := findMetric(rm, "process.cpu.time")
	assert.True(t, ok, "process.cpu.time should be reported")
	_, ok = findMetric(rm, "process.memory.physical_usage")
	assert.True(t, ok, "process.memory.physical_usage should be reported")
}

func TestMetricReporter_NilIsNoop(t *testing.T) {
	var reporter *MetricReporter
	ctx := t.Context()
	assert.NotPanics(t, func() {
		reporter.RecordRemoteConfig(ctx, time.Second, nil)
		reporter.RecordConfigOperation(ctx, OperationApply, nil)
		reporter.RecordCommand(ctx, protobufs.CommandType_CommandType_Restart, nil)
		reporter.RecordHeartbeat(ctx, nil)
		reporter.RecordConnectionAttempt(ctx, nil)
		reporter.RecordCollectors(ctx, 1, 0)
		reporter.RecordProxyAgents(ctx, 1)
		reporter.Shutdown()
	})
}

func TestNewMetricReporter_InvalidSettings(t *testing.T) {
	_, err := NewMetricReporter(logr.Discard(), &protobufs.TelemetryConnectionSettings{}, "test-agent", "v1", uuid.New())
	require.Error(t, err)
}
