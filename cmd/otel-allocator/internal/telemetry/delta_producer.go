// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"hash/fnv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// deltaSumKey identifies a single data point for cumulative→delta tracking.
type deltaSumKey struct {
	scope  string
	metric string
	attrs  uint64
}

// attrsKey returns a stable hash of an attribute.Set by iterating over all
// key-value pairs in order and hashing their string representations.
// This is more reliable than fmt.Sprintf("%v", ...) whose output format is
// not guaranteed to be stable or collision-free.
func attrsKey(attrs attribute.Set) uint64 {
	h := fnv.New64a()
	for iter := attrs.Iter(); iter.Next(); {
		kv := iter.Attribute()
		_, _ = h.Write([]byte(string(kv.Key)))
		_, _ = h.Write([]byte(kv.Value.AsString()))
	}
	return h.Sum64()
}

// prevSumPoint stores the last observed cumulative value and its timestamp.
type prevSumPoint struct {
	value float64
	t     time.Time
}

// deltaProducer wraps a metric.Producer and converts cumulative monotonic Sum
// data points to delta. The OTel SDK's PeriodicReader only applies the
// TemporalitySelector to SDK-native instruments; external Producer output is
// forwarded as-is. For the Prometheus bridge this means counters arrive as
// cumulative sums. This wrapper converts them to delta so that backends
// configured to expect delta temporality receive the correct data.
//
// The prev map grows to one entry per unique {scope, metric, attribute-set}
// combination that has ever been observed. For a stable Prometheus scrape target
// (fixed label cardinality, no ephemeral label values such as pod IPs) this set
// is bounded in practice. High-cardinality label churn would cause unbounded
// growth; in that case the Prometheus scrape target itself would already be a
// problem, so this is an acceptable tradeoff for the current use case.
type deltaProducer struct {
	inner sdkmetric.Producer
	mu    sync.Mutex
	prev  map[deltaSumKey]prevSumPoint
}

func newDeltaProducer(inner sdkmetric.Producer) sdkmetric.Producer {
	return &deltaProducer{inner: inner, prev: make(map[deltaSumKey]prevSumPoint)}
}

func (d *deltaProducer) Produce(ctx context.Context) ([]metricdata.ScopeMetrics, error) {
	sms, err := d.inner.Produce(ctx)

	d.mu.Lock()
	defer d.mu.Unlock()

	for si := range sms {
		scope := sms[si].Scope.Name
		for mi := range sms[si].Metrics {
			m := &sms[si].Metrics[mi]
			sum, ok := m.Data.(metricdata.Sum[float64])
			if !ok || !sum.IsMonotonic || sum.Temporality != metricdata.CumulativeTemporality {
				continue
			}
			deltaDPs := make([]metricdata.DataPoint[float64], 0, len(sum.DataPoints))
			for _, dp := range sum.DataPoints {
				key := deltaSumKey{
					scope:  scope,
					metric: m.Name,
					attrs:  attrsKey(dp.Attributes),
				}
				prev, seen := d.prev[key]
				d.prev[key] = prevSumPoint{value: dp.Value, t: dp.Time}
				if !seen {
					// First observation: emit with delta = current value so the
					// backend sees the counter from the start rather than skipping it.
					deltaDPs = append(deltaDPs, metricdata.DataPoint[float64]{
						Attributes: dp.Attributes,
						StartTime:  dp.StartTime,
						Time:       dp.Time,
						Value:      dp.Value,
						Exemplars:  dp.Exemplars,
					})
					continue
				}
				delta := dp.Value - prev.value
				if delta < 0 {
					// Counter reset: emit the new value as the delta.
					delta = dp.Value
				}
				deltaDPs = append(deltaDPs, metricdata.DataPoint[float64]{
					Attributes: dp.Attributes,
					StartTime:  prev.t,
					Time:       dp.Time,
					Value:      delta,
					Exemplars:  dp.Exemplars,
				})
			}
			sum.Temporality = metricdata.DeltaTemporality
			sum.DataPoints = deltaDPs
			m.Data = sum
		}
	}

	return sms, err
}
