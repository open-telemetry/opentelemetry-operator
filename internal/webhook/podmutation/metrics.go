// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package podmutation

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	meterName  = "pod-mutation-webhook"
	metricName = "opentelemetry_operator_pod_mutations_total"
)

// Status constants for mutation outcomes.
const (
	StatusSuccess  = "success"
	StatusSkipped  = "skipped"
	StatusRejected = "rejected"
	StatusError    = "error"
)

// Reason constants for mutation outcomes.
const (
	ReasonAlreadyInstrumented = "already_instrumented"
	ReasonAlreadyExists       = "already_exists"
	ReasonFeatureDisabled     = "feature_disabled"
	ReasonLookupFailed        = "lookup_failed"
	ReasonValidationFailed    = "validation_failed"
	ReasonInvalidContainers   = "invalid_containers"
	ReasonMultipleInstances   = "multiple_instances"
	ReasonNoInstances         = "no_instances"
	ReasonNotSidecarMode      = "not_sidecar_mode"
	ReasonUnknown             = "unknown"
)

// MutationType constants.
const (
	MutationTypeSidecar         = "sidecar"
	MutationTypeInstrumentation = "instrumentation"
)

// Attribute keys.
var (
	attrMutationType = attribute.Key("mutation_type")
	attrStatus       = attribute.Key("status")
	attrReason       = attribute.Key("reason")
	attrLanguage     = attribute.Key("language")
)

// PodMutationMetricsRecorder defines the interface for recording pod mutation metrics.
// +kubebuilder:object:generate=false
type PodMutationMetricsRecorder interface {
	RecordSidecarMutation(ctx context.Context, status, reason, namespace string)
	RecordInstrumentationMutation(ctx context.Context, status, reason, language, namespace string)
}

// PodMutationMetrics holds the metrics for the pod mutation webhook.
// +kubebuilder:object:generate=false
type PodMutationMetrics struct {
	mutationsTotal metric.Int64Counter
}

// Ensure PodMutationMetrics implements PodMutationMetricsRecorder.
var _ PodMutationMetricsRecorder = (*PodMutationMetrics)(nil)

// NoopMetrics is a no-operation implementation of PodMutationMetricsRecorder for testing.
// +kubebuilder:object:generate=false
type NoopMetrics struct{}

// Ensure NoopMetrics implements PodMutationMetricsRecorder.
var _ PodMutationMetricsRecorder = (*NoopMetrics)(nil)

func (n *NoopMetrics) RecordSidecarMutation(ctx context.Context, status, reason, namespace string) {}
func (n *NoopMetrics) RecordInstrumentationMutation(ctx context.Context, status, reason, language, namespace string) {
}

// NewMetrics creates a new PodMutationMetrics instance.
func NewMetrics(meterProvider metric.MeterProvider) (*PodMutationMetrics, error) {
	meter := meterProvider.Meter(meterName)

	mutationsTotal, err := meter.Int64Counter(
		metricName,
		metric.WithDescription("Total number of pod mutation attempts"),
	)
	if err != nil {
		return nil, err
	}

	return &PodMutationMetrics{
		mutationsTotal: mutationsTotal,
	}, nil
}

// NewNoopMetrics returns a no-operation metrics recorder for testing or when metrics are disabled.
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

func (m *PodMutationMetrics) RecordSidecarMutation(ctx context.Context, status, reason, namespace string) {
	attrs := []attribute.KeyValue{
		attrMutationType.String(MutationTypeSidecar),
		attrStatus.String(status),
		semconv.K8SNamespaceName(namespace),
	}
	if reason != "" {
		attrs = append(attrs, attrReason.String(reason))
	}
	m.mutationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (m *PodMutationMetrics) RecordInstrumentationMutation(ctx context.Context, status, reason, language, namespace string) {
	attrs := []attribute.KeyValue{
		attrMutationType.String(MutationTypeInstrumentation),
		attrStatus.String(status),
		semconv.K8SNamespaceName(namespace),
	}
	if language != "" {
		attrs = append(attrs, attrLanguage.String(language))
	}
	if reason != "" {
		attrs = append(attrs, attrReason.String(reason))
	}
	m.mutationsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}
