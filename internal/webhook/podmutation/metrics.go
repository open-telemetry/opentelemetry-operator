// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package podmutation

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PodMutationMetrics struct {
	mutationsTotal metric.Int64Counter
}

func NewMetrics(meterProvider metric.MeterProvider) (*PodMutationMetrics, error) {
	meter := meterProvider.Meter("pod-mutation-webhook")

	mutationsTotal, err := meter.Int64Counter(
		"opentelemetry_operator_pod_mutations_total",
		metric.WithDescription("Total number of pod mutation attempts"),
	)
	if err != nil {
		return nil, err
	}

	return &PodMutationMetrics{
		mutationsTotal: mutationsTotal,
	}, nil
}

func (m *PodMutationMetrics) RecordSidecarMutation(ctx context.Context, status, reason, namespace string) {
	attrs := []attribute.KeyValue{
		attribute.String("mutation_type", "sidecar"),
		attribute.String("status", status),
		attribute.String("namespace", namespace),
	}
	if reason != "" {
		attrs = append(attrs, attribute.String("reason", reason))
	}
	m.mutationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (m *PodMutationMetrics) RecordInstrumentationMutation(ctx context.Context, status, reason, language, namespace string) {
	attrs := []attribute.KeyValue{
		attribute.String("mutation_type", "instrumentation"),
		attribute.String("status", status),
		attribute.String("namespace", namespace),
	}
	if language != "" {
		attrs = append(attrs, attribute.String("language", language))
	}
	if reason != "" {
		attrs = append(attrs, attribute.String("reason", reason))
	}
	m.mutationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}
