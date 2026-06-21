// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"

	"github.com/prometheus/common/model"
)

// Series is an expected Prometheus series for HasSeries. A sample matches when it
// carries every label in Labels (exact value), carries every label in Present (any
// value), carries no label in Absent, and — when Value is non-nil — its value
// satisfies it. When Exact is true the sample must carry EXACTLY the union of Labels
// and Present and nothing else, which pins a target's complete label set while still
// allowing non-deterministic values (e.g. a pod name) via Present.
//
// Absent and Exact are how a test proves target labeling: a static scrape config
// must carry no Kubernetes service-discovery labels, while a ServiceMonitor target
// must carry exactly the relabeled identity the allocator generated.
type Series struct {
	Labels  map[string]string
	Present []string
	Absent  []string
	Exact   bool
	Value   func(model.SampleValue) bool
}

// HasSeries returns a PromQL check (for EventuallyPromQL) that passes when some
// sample in the result vector matches want.
func HasSeries(want Series) func(model.Vector) error {
	return func(v model.Vector) error {
		for _, s := range v {
			if seriesMatches(want, s) {
				return nil
			}
		}
		return fmt.Errorf("no sample matched labels=%v absent=%v among %d samples: %v",
			want.Labels, want.Absent, len(v), v)
	}
}

func seriesMatches(want Series, s *model.Sample) bool {
	for k, val := range want.Labels {
		if string(s.Metric[model.LabelName(k)]) != val {
			return false
		}
	}
	for _, k := range want.Present {
		if _, ok := s.Metric[model.LabelName(k)]; !ok {
			return false
		}
	}
	for _, k := range want.Absent {
		if _, ok := s.Metric[model.LabelName(k)]; ok {
			return false
		}
	}
	if want.Exact {
		allowed := map[string]bool{"__name__": true}
		for k := range want.Labels {
			allowed[k] = true
		}
		for _, k := range want.Present {
			allowed[k] = true
		}
		for k := range s.Metric {
			if !allowed[string(k)] {
				return false
			}
		}
	}
	return want.Value == nil || want.Value(s.Value)
}

// Equals matches a sample value exactly.
func Equals(want model.SampleValue) func(model.SampleValue) bool {
	return func(got model.SampleValue) bool { return got == want }
}

// AtLeast matches a sample value >= want.
func AtLeast(want model.SampleValue) func(model.SampleValue) bool {
	return func(got model.SampleValue) bool { return got >= want }
}
