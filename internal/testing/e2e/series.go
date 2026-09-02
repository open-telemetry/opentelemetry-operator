// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"maps"
	"slices"
	"strings"

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

// String renders every clause of the matcher, so a failure message says which
// conditions the samples were judged against — not just some of them.
func (s Series) String() string {
	var clauses []string
	if len(s.Labels) > 0 {
		pairs := make([]string, 0, len(s.Labels))
		for _, k := range slices.Sorted(maps.Keys(s.Labels)) {
			pairs = append(pairs, fmt.Sprintf("%s=%q", k, s.Labels[k]))
		}
		clauses = append(clauses, "labels{"+strings.Join(pairs, ", ")+"}")
	}
	if len(s.Present) > 0 {
		clauses = append(clauses, fmt.Sprintf("present%v", s.Present))
	}
	if len(s.Absent) > 0 {
		clauses = append(clauses, fmt.Sprintf("absent%v", s.Absent))
	}
	if s.Exact {
		clauses = append(clauses, "exact")
	}
	if s.Value != nil {
		clauses = append(clauses, "value predicate")
	}
	if len(clauses) == 0 {
		return "Series{any sample}"
	}
	return "Series{" + strings.Join(clauses, ", ") + "}"
}

// HasSeries returns a PromQL check (for Prom.Eventually) that passes when some
// sample in the result vector matches want.
func HasSeries(want Series) func(model.Vector) error {
	return func(v model.Vector) error {
		for _, s := range v {
			if seriesMatches(want, s) {
				return nil
			}
		}
		return fmt.Errorf("no sample matched %s among %d samples: %v", want, len(v), v)
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
