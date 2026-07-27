// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	instrumentation "go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// staticProducer is a test double that returns a fixed set of ScopeMetrics.
type staticProducer struct {
	sms []metricdata.ScopeMetrics
	err error
}

func (p *staticProducer) Produce(_ context.Context) ([]metricdata.ScopeMetrics, error) {
	return p.sms, p.err
}

// cumulativeSum builds a ScopeMetrics containing a single monotonic cumulative Sum.
func cumulativeSum(scope, metric string, dps ...metricdata.DataPoint[float64]) metricdata.ScopeMetrics {
	return metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{Name: scope},
		Metrics: []metricdata.Metrics{{
			Name: metric,
			Data: metricdata.Sum[float64]{
				IsMonotonic: true,
				Temporality: metricdata.CumulativeTemporality,
				DataPoints:  dps,
			},
		}},
	}
}

// dp builds a float64 DataPoint with the given attributes and value.
func dp(val float64, attrs ...attribute.KeyValue) metricdata.DataPoint[float64] {
	t := time.Now()
	return metricdata.DataPoint[float64]{
		Attributes: attribute.NewSet(attrs...),
		StartTime:  t.Add(-time.Minute),
		Time:       t,
		Value:      val,
	}
}

func TestDeltaProducer(t *testing.T) {
	t.Run("first observation emits full value as delta", func(t *testing.T) {
		inner := &staticProducer{sms: []metricdata.ScopeMetrics{
			cumulativeSum("scope", "counter", dp(10.0)),
		}}
		prod := newDeltaProducer(inner)

		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		require.Len(t, sms[0].Metrics[0].Data.(metricdata.Sum[float64]).DataPoints, 1)
		got := sms[0].Metrics[0].Data.(metricdata.Sum[float64])
		assert.Equal(t, metricdata.DeltaTemporality, got.Temporality)
		assert.Equal(t, 10.0, got.DataPoints[0].Value)
	})

	t.Run("second observation emits difference", func(t *testing.T) {
		inner := &staticProducer{}
		prod := newDeltaProducer(inner)

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(10.0))}
		_, _ = prod.Produce(t.Context())

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(25.0))}
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		got := sms[0].Metrics[0].Data.(metricdata.Sum[float64])
		assert.Equal(t, metricdata.DeltaTemporality, got.Temporality)
		assert.Equal(t, 15.0, got.DataPoints[0].Value, "delta should be 25-10=15")
	})

	t.Run("counter reset emits new value as delta", func(t *testing.T) {
		inner := &staticProducer{}
		prod := newDeltaProducer(inner)

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(100.0))}
		_, _ = prod.Produce(t.Context())

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(5.0))}
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		got := sms[0].Metrics[0].Data.(metricdata.Sum[float64])
		assert.Equal(t, 5.0, got.DataPoints[0].Value, "on reset, delta should equal new value")
	})

	t.Run("different attribute sets tracked independently", func(t *testing.T) {
		inner := &staticProducer{}
		prod := newDeltaProducer(inner)

		a := attribute.String("job", "a")
		b := attribute.String("job", "b")

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(10.0, a), dp(20.0, b))}
		_, _ = prod.Produce(t.Context())

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(13.0, a), dp(27.0, b))}
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		dps := sms[0].Metrics[0].Data.(metricdata.Sum[float64]).DataPoints
		require.Len(t, dps, 2)
		vals := map[string]float64{}
		for _, p := range dps {
			v, _ := p.Attributes.Value(attribute.Key("job"))
			vals[v.AsString()] = p.Value
		}
		assert.Equal(t, 3.0, vals["a"], "delta for job=a should be 13-10=3")
		assert.Equal(t, 7.0, vals["b"], "delta for job=b should be 27-20=7")
	})

	t.Run("non-monotonic sum is passed through unchanged", func(t *testing.T) {
		inner := &staticProducer{sms: []metricdata.ScopeMetrics{{
			Metrics: []metricdata.Metrics{{
				Name: "updown",
				Data: metricdata.Sum[float64]{
					IsMonotonic: false,
					Temporality: metricdata.CumulativeTemporality,
					DataPoints:  []metricdata.DataPoint[float64]{dp(42.0)},
				},
			}},
		}}}
		prod := newDeltaProducer(inner)
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		got := sms[0].Metrics[0].Data.(metricdata.Sum[float64])
		assert.Equal(t, metricdata.CumulativeTemporality, got.Temporality, "non-monotonic sum must not be converted")
		assert.Equal(t, 42.0, got.DataPoints[0].Value)
	})

	t.Run("gauge data is passed through unchanged", func(t *testing.T) {
		inner := &staticProducer{sms: []metricdata.ScopeMetrics{{
			Metrics: []metricdata.Metrics{{
				Name: "temperature",
				Data: metricdata.Gauge[float64]{
					DataPoints: []metricdata.DataPoint[float64]{dp(98.6)},
				},
			}},
		}}}
		prod := newDeltaProducer(inner)
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		got, ok := sms[0].Metrics[0].Data.(metricdata.Gauge[float64])
		require.True(t, ok, "gauge data type must be preserved")
		assert.Equal(t, 98.6, got.DataPoints[0].Value)
	})

	t.Run("zero delta on unchanged value", func(t *testing.T) {
		inner := &staticProducer{}
		prod := newDeltaProducer(inner)

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(50.0))}
		_, _ = prod.Produce(t.Context())

		inner.sms = []metricdata.ScopeMetrics{cumulativeSum("scope", "counter", dp(50.0))}
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		got := sms[0].Metrics[0].Data.(metricdata.Sum[float64])
		assert.Equal(t, 0.0, got.DataPoints[0].Value, "no change should produce delta of 0")
	})

	t.Run("multiple scopes and metrics tracked independently", func(t *testing.T) {
		inner := &staticProducer{}
		prod := newDeltaProducer(inner)

		inner.sms = []metricdata.ScopeMetrics{
			cumulativeSum("scopeA", "requests", dp(100.0)),
			cumulativeSum("scopeB", "requests", dp(200.0)),
		}
		_, _ = prod.Produce(t.Context())

		inner.sms = []metricdata.ScopeMetrics{
			cumulativeSum("scopeA", "requests", dp(110.0)),
			cumulativeSum("scopeB", "requests", dp(230.0)),
		}
		sms, err := prod.Produce(t.Context())
		require.NoError(t, err)
		require.Len(t, sms, 2)
		dpA := sms[0].Metrics[0].Data.(metricdata.Sum[float64]).DataPoints[0].Value
		dpB := sms[1].Metrics[0].Data.(metricdata.Sum[float64]).DataPoints[0].Value
		assert.Equal(t, 10.0, dpA, "scopeA delta should be 110-100=10")
		assert.Equal(t, 30.0, dpB, "scopeB delta should be 230-200=30")
	})

	t.Run("inner producer error is propagated", func(t *testing.T) {
		inner := &staticProducer{err: errors.New("scrape failed")}
		prod := newDeltaProducer(inner)
		_, err := prod.Produce(t.Context())
		require.ErrorContains(t, err, "scrape failed")
	})
}
