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
	"slices"
	"testing"

	"github.com/go-logr/logr"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/prehook"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
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
// It reuses the allocator's own code paths:
//   - target.Discoverer.UpdateTsets + Reload run processTargetGroups/mergeLabels
//     synchronously (bypassing the production 5s reload debounce).
//   - prehook.Hook.Apply runs relabel filtering and computes the identity hash.
//
// Only static_configs are supported for now; file_sd and others can be added later.
func runAllocatorPipeline(t *testing.T, promYAML string) map[targetKey][]allocatorResult {
	t.Helper()

	cfg, err := promconfig.Load(promYAML, config.NopLogger)
	require.NoError(t, err, "loading fixture prometheus config")

	// Drive processTargetGroups via the real Discoverer. The capture callback
	// receives all discovered items (post-merge, pre-relabel-filter).
	var captured []*target.Item
	discoverer, err := target.NewDiscoverer(logr.Discard(), nil, nil, nil, func(items []*target.Item) {
		captured = items
	})
	require.NoError(t, err)

	tsets := map[string][]*targetgroup.Group{}
	relabelCfg := map[string][]*relabel.Config{}
	for _, sc := range cfg.ScrapeConfigs {
		var groups []*targetgroup.Group
		for _, sdc := range sc.ServiceDiscoveryConfigs {
			static, ok := sdc.(discovery.StaticConfig)
			require.Truef(t, ok, "job %q uses unsupported service discovery %T; only static_configs are supported", sc.JobName, sdc)
			groups = append(groups, static...)
		}
		tsets[sc.JobName] = groups
		relabelCfg[sc.JobName] = sc.RelabelConfigs
	}

	discoverer.UpdateTsets(tsets)
	discoverer.Reload()
	allItems := captured

	// Apply relabel filtering exactly as production does. Apply mutates its
	// input slice in place, so feed it a clone to keep allItems intact.
	hook := prehook.New("relabel-config", logr.Discard())
	require.NotNil(t, hook)
	hook.SetConfig(relabelCfg)
	kept := hook.Apply(slices.Clone(allItems))

	// Index kept items by (job, full pre-relabel label-set hash) — not by address,
	// so targets that share a pre-relabel address are not confused. We keep a
	// pointer to the filtered Item because it carries the relabel-computed identity
	// hash used later.
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
