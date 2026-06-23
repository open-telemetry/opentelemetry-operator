// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

var update = flag.Bool("update", false, "regenerate golden.json files via promtool ($PROMTOOL)")

// seededLabels are the labels Prometheus's scrape layer (PopulateLabels) adds
// before relabeling, which the target allocator intentionally does not seed
// (the collector's prometheus receiver adds them later). They are excluded when
// comparing the allocator's served pre-relabel labels to Prometheus's
// discoveredLabels.
var seededLabels = []string{
	"job",
	"__scheme__",
	"__metrics_path__",
	"__scrape_interval__",
	"__scrape_timeout__",
}

// divergentFixtures are fixtures whose behavior is known to differ from raw
// Prometheus. They are skipped until the allocator is fixed.
var divergentFixtures = map[string]string{
	"seeded-labels": "relabel rules referencing scrape-seeded labels diverge from Prometheus (regression of #4074); see testdata/gap-seeded-labels/",
}

// TestConformance runs every fixture under testdata/ through the target
// allocator and compares against the committed promtool golden.
func TestConformance(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/*/prometheus.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures, "no fixtures found under testdata/")

	for _, cfgPath := range fixtures {
		dir := filepath.Dir(cfgPath)
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			promYAML, err := os.ReadFile(cfgPath)
			require.NoError(t, err)
			goldenPath := filepath.Join(dir, "golden.json")

			if *update {
				regenGolden(t, cfgPath, jobNames(t, string(promYAML)), goldenPath)
			}
			if reason, ok := divergentFixtures[name]; ok {
				t.Skip(reason)
			}

			golden := loadGolden(t, goldenPath)
			allocator := runAllocatorPipeline(t, string(promYAML))
			for _, m := range compare(allocator, golden) {
				assert.Fail(t, m)
			}
		})
	}
}

func jobNames(t *testing.T, promYAML string) []string {
	cfg, err := promconfig.Load(promYAML, config.NopLogger)
	require.NoError(t, err)
	jobs := make([]string, 0, len(cfg.ScrapeConfigs))
	for _, sc := range cfg.ScrapeConfigs {
		jobs = append(jobs, sc.JobName)
	}
	return jobs
}

// matchedPair is a target matched between the allocator and the Prometheus
// golden, joined on (job, address) and — for shared addresses — on the
// pre-relabel label set.
type matchedPair struct {
	key targetKey
	a   allocatorResult
	gt  goldenTarget
}

// compare runs the three differential assertions against raw Prometheus and
// returns a (sorted) list of mismatch descriptions; empty means conformant.
//
//	keep/drop parity:    the allocator keeps a target iff Prometheus does.
//	merge fidelity:      the allocator's served (pre-relabel) labels equal
//	                     Prometheus's discoveredLabels minus the scrape labels it
//	                     defers to the receiver.
//	identity grouping:   the allocator's identity-hash partition equals
//	                     Prometheus's post-relabel-label partition (no silent
//	                     collisions, no false splits).
func compare(allocator map[targetKey][]allocatorResult, golden map[targetKey][]goldenTarget) []string {
	var mismatches []string
	var pairs []matchedPair

	keys := map[targetKey]struct{}{}
	for k := range allocator {
		keys[k] = struct{}{}
	}
	for k := range golden {
		keys[k] = struct{}{}
	}
	for k := range keys {
		p, unmatched := matchBucket(k, allocator[k], golden[k])
		pairs = append(pairs, p...)
		mismatches = append(mismatches, unmatched...)
	}

	for _, p := range pairs {
		if want := !p.gt.dropped(); want != p.a.kept {
			mismatches = append(mismatches, fmt.Sprintf("keep/drop mismatch for %+v: Prometheus kept=%v, allocator kept=%v", p.key, want, p.a.kept))
		}
		if want, got := withoutSeeded(p.gt.DiscoveredLabels), p.a.item.Labels; !labels.Equal(want, got) {
			mismatches = append(mismatches, fmt.Sprintf("merge/discovered-label mismatch for %+v:\n      want %v\n      got  %v", p.key, want, got))
		}
	}

	mismatches = append(mismatches, identityPartitionMismatches(pairs)...)
	slices.Sort(mismatches)
	return mismatches
}

// matchBucket pairs the allocator and golden targets that share a (job, address)
// bucket. The common case is exactly one of each, paired directly so keep/drop
// and merge differences read cleanly. When several targets share an address they
// are matched by their pre-relabel label set; leftovers are reported as unmatched.
func matchBucket(key targetKey, al []allocatorResult, gl []goldenTarget) (pairs []matchedPair, unmatched []string) {
	if len(al) == 1 && len(gl) == 1 {
		return []matchedPair{{key, al[0], gl[0]}}, nil
	}
	used := make([]bool, len(gl))
	for _, a := range al {
		ah := a.item.Labels.Hash()
		match := -1
		for i, g := range gl {
			if !used[i] && withoutSeeded(g.DiscoveredLabels).Hash() == ah {
				match = i
				break
			}
		}
		if match < 0 {
			unmatched = append(unmatched, fmt.Sprintf("allocator target %+v %v has no matching Prometheus target", key, a.item.Labels))
			continue
		}
		used[match] = true
		pairs = append(pairs, matchedPair{key, a, gl[match]})
	}
	for i, g := range gl {
		if !used[i] {
			unmatched = append(unmatched, fmt.Sprintf("Prometheus target %+v %v has no matching allocator target", key, withoutSeeded(g.DiscoveredLabels)))
		}
	}
	return pairs, unmatched
}

// identityPartitionMismatches checks that, over surviving targets, the
// equivalence relation induced by the allocator's identity hash matches the one
// induced by Prometheus's post-relabel label set.
func identityPartitionMismatches(pairs []matchedPair) []string {
	var kept []matchedPair
	for _, p := range pairs {
		if p.a.kept && !p.gt.dropped() {
			kept = append(kept, p)
		}
	}

	var mismatches []string
	for i := 0; i < len(kept); i++ {
		for j := i + 1; j < len(kept); j++ {
			p1, p2 := kept[i], kept[j]
			sameProm := p1.gt.Labels.Hash() == p2.gt.Labels.Hash()
			sameAllocator := p1.a.item.Hash() == p2.a.item.Hash()
			if sameProm != sameAllocator {
				mismatches = append(mismatches, fmt.Sprintf(
					"identity-grouping mismatch for %+v vs %+v: Prometheus same-identity=%v, allocator same-hash=%v",
					p1.key, p2.key, sameProm, sameAllocator))
			}
		}
	}
	return mismatches
}

// withoutSeeded returns the label set with the Prometheus-seeded labels removed.
func withoutSeeded(ls labels.Labels) labels.Labels {
	return labels.NewBuilder(ls).Del(seededLabels...).Labels()
}
