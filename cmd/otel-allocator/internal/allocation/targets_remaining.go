// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// targetsRemainingMetrics holds the values observed for opentelemetry_allocator_targets_remaining.
type targetsRemainingMetrics struct {
	// mu protects total and perJob.
	mu     sync.Mutex
	total  int64
	perJob map[jobNamespace]int64
}

// jobNamespace identifies the targets_remaining series for a job and namespace.
type jobNamespace struct {
	job       string
	namespace string
}

func (k jobNamespace) attributes() metric.MeasurementOption {
	attrs := []attribute.KeyValue{attribute.String("job.name", k.job)}
	// Targets without a __meta_kubernetes_namespace label, such as static targets, have no namespace.
	if k.namespace != "" {
		attrs = append(attrs, semconv.K8SNamespaceName(k.namespace))
	}
	return metric.WithAttributes(attrs...)
}

// newTargetsRemainingMetrics registers the opentelemetry_allocator_targets_remaining observable gauge.
func newTargetsRemainingMetrics(meter metric.Meter) (*targetsRemainingMetrics, error) {
	m := &targetsRemainingMetrics{}
	_, err := meter.Int64ObservableGauge("opentelemetry_allocator_targets_remaining", metric.WithDescription("Number of targets kept after filtering."), metric.WithInt64Callback(m.observe))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// set saves the number of targets kept after filtering.
func (m *targetsRemainingMetrics) set(targets []*target.Item) {
	var perJob map[jobNamespace]int64
	if featuregate.TargetsRemainingAttributes.IsEnabled() {
		perJob = make(map[jobNamespace]int64)
		for _, item := range targets {
			perJob[jobNamespace{job: item.JobName, namespace: item.GetNamespace()}]++
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.total = int64(len(targets))
	m.perJob = perJob
}

// observe reports the number of targets kept after filtering. With the
// targetallocator.targetsremainingattributes feature gate enabled, it reports one value per job and
// namespace. The SDK exports only the values from the latest callback, so a job and namespace
// without targets is no longer reported.
// See open-telemetry/opentelemetry-operator#4637.
func (m *targetsRemainingMetrics) observe(_ context.Context, o metric.Int64Observer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !featuregate.TargetsRemainingAttributes.IsEnabled() {
		o.Observe(m.total)
		return nil
	}
	for key, count := range m.perJob {
		o.Observe(count, key.attributes())
	}
	return nil
}
