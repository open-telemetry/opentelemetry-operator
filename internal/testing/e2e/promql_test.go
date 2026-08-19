// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePromVector(t *testing.T) {
	t.Run("decodes samples with metric and value", func(t *testing.T) {
		vec, err := parsePromVector([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "up", "job": "sample-app"}, "value": [1700000000.5, "1"]},
					{"metric": {"__name__": "up", "job": "other"}, "value": [1700000000.5, "0"]}
				]
			}
		}`))
		require.NoError(t, err)
		require.Len(t, vec, 2)
		assert.Equal(t, model.LabelValue("sample-app"), vec[0].Metric["job"])
		assert.Equal(t, model.SampleValue(1), vec[0].Value)
		assert.Equal(t, model.Time(1700000000500), vec[0].Timestamp)
		assert.Equal(t, model.SampleValue(0), vec[1].Value)
	})

	t.Run("decodes an empty result", func(t *testing.T) {
		vec, err := parsePromVector([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
		require.NoError(t, err)
		assert.Empty(t, vec)
	})

	t.Run("reports a query error", func(t *testing.T) {
		_, err := parsePromVector([]byte(`{"status":"error","errorType":"bad_data","error":"invalid parameter"}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parameter")
	})

	t.Run("rejects a non-vector result type", func(t *testing.T) {
		_, err := parsePromVector([]byte(`{"status":"success","data":{"resultType":"matrix","result":[]}}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"matrix"`)
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		_, err := parsePromVector([]byte(`not json`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode prometheus response")
	})

	t.Run("rejects a malformed result", func(t *testing.T) {
		_, err := parsePromVector([]byte(`{"status":"success","data":{"resultType":"vector","result":{}}}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode prometheus vector")
	})
}

func TestCanonicalLabels(t *testing.T) {
	metric := model.Metric{
		"__name__": "up",
		"pipeline": "ta",
		"job":      "sample-app",
		"instance": "10.0.0.1:8080",
		"pod":      "sample-app-abc",
	}

	t.Run("renders sorted and drops nothing by default", func(t *testing.T) {
		assert.Equal(t,
			`{__name__="up", instance="10.0.0.1:8080", job="sample-app", pipeline="ta", pod="sample-app-abc"}`,
			canonicalLabels(metric, nil))
	})

	t.Run("drops the requested labels", func(t *testing.T) {
		drop := map[model.LabelName]bool{"__name__": true, "pipeline": true, "pod": true}
		assert.Equal(t, `{instance="10.0.0.1:8080", job="sample-app"}`, canonicalLabels(metric, drop))
	})

	t.Run("label order in the sample does not affect the rendering", func(t *testing.T) {
		assert.Equal(t,
			canonicalLabels(model.Metric{"b": "2", "a": "1"}, nil),
			canonicalLabels(model.Metric{"a": "1", "b": "2"}, nil))
	})

	t.Run("renders an all-dropped label set as the empty set", func(t *testing.T) {
		assert.Equal(t, "{}", canonicalLabels(model.Metric{"job": "x"}, map[model.LabelName]bool{"job": true}))
	})
}

func TestDiffSets(t *testing.T) {
	for _, tt := range []struct {
		name string
		a, b map[string]bool
		want string
	}{
		{
			name: "equal sets have no diff",
			a:    map[string]bool{`{job="a"}`: true, `{job="b"}`: true},
			b:    map[string]bool{`{job="b"}`: true, `{job="a"}`: true},
			want: "",
		},
		{
			name: "both empty",
			a:    map[string]bool{},
			b:    map[string]bool{},
			want: "",
		},
		{
			name: "member only in A",
			a:    map[string]bool{`{job="a"}`: true, `{job="b"}`: true},
			b:    map[string]bool{`{job="a"}`: true},
			want: "  only in A: {job=\"b\"}\n",
		},
		{
			name: "member only in B",
			a:    map[string]bool{`{job="a"}`: true},
			b:    map[string]bool{`{job="a"}`: true, `{job="b"}`: true},
			want: "  only in B: {job=\"b\"}\n",
		},
		{
			name: "both sides, rendered in sorted order",
			a:    map[string]bool{`{job="a2"}`: true, `{job="a1"}`: true},
			b:    map[string]bool{`{job="b2"}`: true, `{job="b1"}`: true},
			want: "  only in A: {job=\"a1\"}\n  only in A: {job=\"a2\"}\n  only in B: {job=\"b1\"}\n  only in B: {job=\"b2\"}\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, diffSets(tt.a, tt.b))
		})
	}
}

func TestSamePartitionLabels(t *testing.T) {
	// Two pipelines scraping the same two targets, agreeing on every label except
	// the partition label itself.
	agreeing := model.Vector{
		sample(1, "__name__", "up", "pipeline", "ta", "job", "sample-app", "instance", "a:8080"),
		sample(1, "__name__", "up", "pipeline", "ta", "job", "sample-app", "instance", "b:8080"),
		sample(1, "__name__", "up", "pipeline", "oracle", "job", "sample-app", "instance", "a:8080"),
		sample(1, "__name__", "up", "pipeline", "oracle", "job", "sample-app", "instance", "b:8080"),
	}
	d := Differential{Query: "up", PartitionLabel: "pipeline", WantPartitions: 2}

	t.Run("passes when the partitions agree", func(t *testing.T) {
		require.NoError(t, samePartitionLabels(agreeing, d))
	})

	t.Run("waits for the wanted number of partitions", func(t *testing.T) {
		err := samePartitionLabels(agreeing[:2], d)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "waiting for 2 partitions")
		assert.Contains(t, err.Error(), "[ta]")
	})

	t.Run("reports a diverging label", func(t *testing.T) {
		diverging := model.Vector{
			sample(1, "__name__", "up", "pipeline", "ta", "job", "sample-app", "instance", "a:8080"),
			sample(1, "__name__", "up", "pipeline", "oracle", "job", "sample-app", "instance", "a:8080", "container", "app"),
		}
		err := samePartitionLabels(diverging, d)
		require.Error(t, err)
		// Partitions are compared in sorted order, so "oracle" is the reference (A).
		assert.Contains(t, err.Error(), `pipeline="oracle" (A)`)
		assert.Contains(t, err.Error(), `pipeline="ta" (B)`)
		assert.Contains(t, err.Error(), `only in A: {container="app", instance="a:8080", job="sample-app"}`)
		assert.Contains(t, err.Error(), `only in B: {instance="a:8080", job="sample-app"}`)
	})

	t.Run("ignored labels may differ", func(t *testing.T) {
		diverging := model.Vector{
			sample(1, "__name__", "up", "pipeline", "ta", "job", "sample-app", "endpoint", "metrics"),
			sample(1, "__name__", "up", "pipeline", "oracle", "job", "sample-app", "endpoint", "http-metrics"),
		}
		require.Error(t, samePartitionLabels(diverging, d))
		ignoring := d
		ignoring.Ignore = []string{"endpoint"}
		require.NoError(t, samePartitionLabels(diverging, ignoring))
	})

	t.Run("the metric name never counts as a difference", func(t *testing.T) {
		renamed := model.Vector{
			sample(1, "__name__", "up", "pipeline", "ta", "job", "sample-app"),
			sample(1, "__name__", "up_total", "pipeline", "oracle", "job", "sample-app"),
		}
		require.NoError(t, samePartitionLabels(renamed, d))
	})
}
