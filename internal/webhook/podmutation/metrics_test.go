// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package podmutation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
)

func TestMetricsCounts(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	metrics, err := podmutation.NewMetrics(provider)
	require.NoError(t, err)

	ctx := context.Background()

	// Record some sidecar mutations
	metrics.RecordSidecarMutation(ctx, "success", "", "default")
	metrics.RecordSidecarMutation(ctx, "skipped", "already_exists", "ns-1")
	metrics.RecordSidecarMutation(ctx, "error", "no_instances", "ns-2")
	metrics.RecordSidecarMutation(ctx, "error", "multiple_instances", "ns-2")

	// Record some instrumentation mutations
	metrics.RecordInstrumentationMutation(ctx, "success", "", "java", "default")
	metrics.RecordInstrumentationMutation(ctx, "success", "", "python", "default")
	metrics.RecordInstrumentationMutation(ctx, "skipped", "already_instrumented", "", "ns-1")
	metrics.RecordInstrumentationMutation(ctx, "rejected", "feature_disabled", "nodejs", "ns-2")
	metrics.RecordInstrumentationMutation(ctx, "error", "lookup_failed", "go", "ns-3")

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	require.Len(t, rm.ScopeMetrics, 1)
	scopeMetrics := rm.ScopeMetrics[0]
	assert.Equal(t, "pod-mutation-webhook", scopeMetrics.Scope.Name)
	require.Len(t, scopeMetrics.Metrics, 1)
	m := scopeMetrics.Metrics[0]
	assert.Equal(t, "opentelemetry_operator_pod_mutations_total", m.Name)
	assert.Equal(t, "Total number of pod mutation attempts", m.Description)

	// Verify data points
	require.IsType(t, metricdata.Sum[int64]{}, m.Data)
	sum := m.Data.(metricdata.Sum[int64])
	assert.True(t, sum.IsMonotonic)
	require.Len(t, sum.DataPoints, 9)

	// Helper to check for a specific data point
	checkPoint := func(expectedValue int64, expectedAttrs []attribute.KeyValue) {
		found := false
		attrSet := attribute.NewSet(expectedAttrs...)
		for _, dp := range sum.DataPoints {
			if dp.Attributes.Equals(&attrSet) {
				assert.Equal(t, expectedValue, dp.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "Expected data point not found: %v", expectedAttrs)
	}

	// Verify sidecar points
	checkPoint(1, []attribute.KeyValue{
		attribute.String("mutation_type", "sidecar"),
		attribute.String("status", "success"),
		attribute.String("namespace", "default"),
	})
	checkPoint(1, []attribute.KeyValue{
		attribute.String("mutation_type", "sidecar"),
		attribute.String("status", "skipped"),
		attribute.String("reason", "already_exists"),
		attribute.String("namespace", "ns-1"),
	})

	// Verify instrumentation points
	checkPoint(1, []attribute.KeyValue{
		attribute.String("mutation_type", "instrumentation"),
		attribute.String("status", "success"),
		attribute.String("language", "java"),
		attribute.String("namespace", "default"),
	})
	checkPoint(1, []attribute.KeyValue{
		attribute.String("mutation_type", "instrumentation"),
		attribute.String("status", "rejected"),
		attribute.String("reason", "feature_disabled"),
		attribute.String("language", "nodejs"),
		attribute.String("namespace", "ns-2"),
	})
}
