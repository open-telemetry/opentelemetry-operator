// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package conformance contains a differential conformance suite for the target
// allocator's discovery + relabel pipeline. It runs the allocator's real code
// (label merge, relabel filtering, identity hashing) over a matrix of scrape
// configs and compares the result against raw Prometheus, captured via
// `promtool check service-discovery` golden files.
//
// The suite asserts purely on target label sets and identity — the target
// allocator's actual job and where its historically recurring bugs live
// (silent target loss from identity/label-set divergence). It needs no
// Kubernetes cluster, no collector, and no metrics backend.
package conformance

import (
	"testing"

	"github.com/go-logr/logr"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

// targetKey joins a Prometheus golden target to an allocator target. Both sides
// retain the pre-relabel address, so (job, address) is a stable join key for the
// fixtures, which use distinct addresses per target except where they
// deliberately share one (then disambiguated by their label set).
type targetKey struct {
	job     string
	address string
}

// allocatorResult is the target allocator's view of one discovered target: the
// produced target.Item plus whether it survived relabel filtering — a status the
// Item itself does not carry. When kept, item is the filtered Item, so item.Hash()
// returns the allocator's relabel-computed identity hash.
type allocatorResult struct {
	item *target.Item
	kept bool
}

func (r allocatorResult) key() targetKey {
	return targetKey{r.item.JobName, r.item.TargetURL}
}

// runAllocatorPipeline drives the target allocator's real discovery + relabel
// pipeline over a raw Prometheus config and returns its per-target results,
// bucketed by (job, address). A bucket holds more than one result only when
// several targets share a pre-relabel address.
//
// Relabel filtering now happens inside the discoverer (processTargetGroups), which
// only emits survivors. To still observe both sides of the filter, the suite drives
// the discoverer twice over the same injected target sets:
//   - filtering OFF: every merged target, pre-relabel (the discovered set).
//   - filtering ON:  the survivors, each carrying the relabel-computed identity hash.
//
// Both passes run the allocator's own code (no relabeling is reimplemented here); a
// target present in the first pass but not the second was dropped by relabeling.
//
// Only static_configs are supported for now; file_sd and others can be added later.
func runAllocatorPipeline(t *testing.T, promYAML string) map[targetKey][]allocatorResult {
	t.Helper()

	cfg, err := promconfig.Load(promYAML, config.NopLogger)
	require.NoError(t, err, "loading fixture prometheus config")

	tsets := map[string][]*targetgroup.Group{}
	for _, sc := range cfg.ScrapeConfigs {
		var groups []*targetgroup.Group
		for _, sdc := range sc.ServiceDiscoveryConfigs {
			static, ok := sdc.(discovery.StaticConfig)
			require.Truef(t, ok, "job %q uses unsupported service discovery %T; only static_configs are supported", sc.JobName, sdc)
			groups = append(groups, static...)
		}
		tsets[sc.JobName] = groups
	}

	// Pass 1: filtering off — all discovered targets, post-merge / pre-relabel.
	allItems := runDiscovery(t, tsets, "", nil)
	// Pass 2: filtering on — the survivors, carrying the relabel identity hash.
	kept := runDiscovery(t, tsets, target.RelabelConfigFilterStrategy, cfg.ScrapeConfigs)

	// Index survivors by (job, full pre-relabel label-set hash) — not by address, so
	// targets that share a pre-relabel address are not confused. The Item carries the
	// relabel-computed identity hash used later.
	type labelIndexKey struct {
		job       string
		labelHash uint64
	}
	keptByLabels := make(map[labelIndexKey]*target.Item, len(kept))
	for _, it := range kept {
		keptByLabels[labelIndexKey{it.JobName, it.Labels.Hash()}] = it
	}

	out := make(map[targetKey][]allocatorResult, len(allItems))
	for _, it := range allItems {
		r := allocatorResult{item: it}
		if filtered, ok := keptByLabels[labelIndexKey{it.JobName, it.Labels.Hash()}]; ok {
			r.item = filtered
			r.kept = true
		}
		out[r.key()] = append(out[r.key()], r)
	}
	return out
}

// runDiscovery drives the allocator's real discovery pipeline once over tsets and
// returns the targets it produces. With filterStrategy set, relabel filtering is
// applied (scrapeConfigs supply the relabel rules and the same no-sharding handling
// as production); otherwise every merged target is returned, pre-relabel.
//
// Target sets are injected directly via UpdateTsets, so no real service discovery
// runs — a fake discovery manager stands in for the real one.
func runDiscovery(t *testing.T, tsets map[string][]*targetgroup.Group, filterStrategy string, scrapeConfigs []*promconfig.ScrapeConfig) []*target.Item {
	t.Helper()
	var captured []*target.Item
	d, err := target.NewDiscoverer(logr.Discard(), fakeDiscoveryManager{}, filterStrategy, nil, func(items []*target.Item) {
		captured = items
	})
	require.NoError(t, err)
	if len(scrapeConfigs) > 0 {
		require.NoError(t, d.ApplyConfig(watcher.EventSourceConfigMap, scrapeConfigs))
	}
	d.UpdateTsets(tsets)
	d.Reload()
	return captured
}

// fakeDiscoveryManager satisfies the discoverer's manager dependency for the suite,
// which injects target sets directly and never runs real service discovery.
type fakeDiscoveryManager struct{}

func (fakeDiscoveryManager) ApplyConfig(map[string]discovery.Configs) error { return nil }

func (fakeDiscoveryManager) SyncCh() <-chan map[string][]*targetgroup.Group { return nil }
