// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sample builds a sample with the given value and labels, e.g.
// sample(1, "__name__", "up", "job", "sample-app").
func sample(value model.SampleValue, labels ...string) *model.Sample {
	metric := model.Metric{}
	for i := 0; i < len(labels); i += 2 {
		metric[model.LabelName(labels[i])] = model.LabelValue(labels[i+1])
	}
	return &model.Sample{Metric: metric, Value: value}
}

func TestSeriesMatches(t *testing.T) {
	up := sample(1, "__name__", "up", "job", "sample-app", "instance", "sample-app:8080", "pod", "sample-app-abc")

	for _, tt := range []struct {
		name  string
		want  Series
		s     *model.Sample
		match bool
	}{
		{
			name:  "empty matcher matches anything",
			want:  Series{},
			s:     up,
			match: true,
		},
		{
			name:  "labels match on exact value",
			want:  Series{Labels: map[string]string{"job": "sample-app"}},
			s:     up,
			match: true,
		},
		{
			name:  "labels reject a different value",
			want:  Series{Labels: map[string]string{"job": "other"}},
			s:     up,
			match: false,
		},
		{
			name:  "labels treat a missing label as the empty value",
			want:  Series{Labels: map[string]string{"namespace": ""}},
			s:     sample(1, "job", "sample-app"),
			match: true,
		},
		{
			name:  "present matches any value",
			want:  Series{Present: []string{"pod", "instance"}},
			s:     up,
			match: true,
		},
		{
			name:  "present rejects a missing label",
			want:  Series{Present: []string{"container"}},
			s:     up,
			match: false,
		},
		{
			name:  "absent matches when the label is missing",
			want:  Series{Absent: []string{"container"}},
			s:     up,
			match: true,
		},
		{
			name:  "absent rejects a present label",
			want:  Series{Absent: []string{"pod"}},
			s:     up,
			match: false,
		},
		{
			name:  "exact allows only labels, present and __name__",
			want:  Series{Labels: map[string]string{"job": "sample-app"}, Present: []string{"instance", "pod"}, Exact: true},
			s:     up,
			match: true,
		},
		{
			name:  "exact rejects an extra label",
			want:  Series{Labels: map[string]string{"job": "sample-app"}, Present: []string{"instance"}, Exact: true},
			s:     up,
			match: false,
		},
		{
			name:  "exact tolerates a sample without __name__",
			want:  Series{Labels: map[string]string{"job": "sample-app"}, Exact: true},
			s:     sample(1, "job", "sample-app"),
			match: true,
		},
		{
			name:  "value predicate accepts a matching value",
			want:  Series{Value: Equals(1)},
			s:     up,
			match: true,
		},
		{
			name:  "value predicate rejects a different value",
			want:  Series{Value: Equals(0)},
			s:     up,
			match: false,
		},
		{
			name:  "at least accepts a greater value",
			want:  Series{Value: AtLeast(1)},
			s:     sample(7, "job", "sample-app"),
			match: true,
		},
		{
			name:  "at least rejects a smaller value",
			want:  Series{Value: AtLeast(10)},
			s:     sample(7, "job", "sample-app"),
			match: false,
		},
		{
			name: "clauses combine",
			want: Series{
				Labels:  map[string]string{"job": "sample-app"},
				Present: []string{"pod"},
				Absent:  []string{"container"},
				Value:   Equals(1),
			},
			s:     up,
			match: true,
		},
		{
			name: "one failing clause fails the whole matcher",
			want: Series{
				Labels:  map[string]string{"job": "sample-app"},
				Present: []string{"pod"},
				Absent:  []string{"instance"},
			},
			s:     up,
			match: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.match, seriesMatches(tt.want, tt.s))
		})
	}
}

func TestHasSeries(t *testing.T) {
	up := sample(1, "__name__", "up", "job", "sample-app")

	t.Run("passes when one sample of the vector matches", func(t *testing.T) {
		check := HasSeries(Series{Labels: map[string]string{"job": "sample-app"}})
		require.NoError(t, check(model.Vector{sample(0, "job", "other"), up}))
	})

	t.Run("reports every clause of the matcher", func(t *testing.T) {
		check := HasSeries(Series{
			Labels:  map[string]string{"job": "sample-app"},
			Present: []string{"pod"},
			Absent:  []string{"container"},
			Exact:   true,
			Value:   Equals(1),
		})
		err := check(model.Vector{up})
		require.Error(t, err)
		// A failure caused by Present/Exact/Value must not read as a label mismatch.
		assert.Contains(t, err.Error(), `labels{job="sample-app"}`)
		assert.Contains(t, err.Error(), "present[pod]")
		assert.Contains(t, err.Error(), "absent[container]")
		assert.Contains(t, err.Error(), "exact")
		assert.Contains(t, err.Error(), "value predicate")
		assert.Contains(t, err.Error(), "among 1 samples")
	})

	t.Run("empty matcher against an empty vector", func(t *testing.T) {
		err := HasSeries(Series{})(model.Vector{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Series{any sample}")
	})
}
