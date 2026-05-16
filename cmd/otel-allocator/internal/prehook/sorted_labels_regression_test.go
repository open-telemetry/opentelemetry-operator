// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prehook

import (
	"maps"
	"slices"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// TestSortedLabelsBlackboxRelabeling is a regression test for a bug where
// processTargetGroups produced unsorted Labels when group labels (e.g. vendor=nginx)
// sorted alphabetically after target labels (e.g. __address__). Labels.Get() uses
// binary search and silently returns empty on unsorted data, causing relabel rules
// that read source_labels: [__address__] to produce empty results. Both targets ended
// up with identical post-relabel labels, creating a hash collision that silently
// dropped one target in the allocator's buildTargetMap.
//
// The fix in processTargetGroups creates a fresh ScratchBuilder per target and calls
// Sort() after all labels are added, ensuring Labels.Get() works correctly.
func TestSortedLabelsBlackboxRelabeling(t *testing.T) {
	jobName := "blackbox-repro-with-label"

	// Build items using the FIXED processTargetGroups logic (globally sorted labels)
	items := buildItemsSortedLabels(jobName,
		map[model.LabelName]model.LabelValue{"vendor": "nginx"},
		[]string{
			"https://target-alpha.example.com:8393/",
			"https://target-beta.example.com:8393/",
		})

	// Verify labels are sorted (the fix's guarantee)
	for _, item := range items {
		var prevName string
		item.Labels.Range(func(l labels.Label) {
			if prevName != "" {
				assert.Less(t, prevName, l.Name, "Labels must be sorted")
			}
			prevName = l.Name
		})
	}

	// Apply prehook relabeling with standard blackbox-exporter pattern
	relabelCfgs := blackboxRelabelConfigs()
	prehook := New("relabel-config", logf.Log.WithName("regression-test"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: relabelCfgs})

	result := prehook.Apply(items)
	require.Len(t, result, 2, "Both targets must survive relabeling — collision drops one")

	// Verify unique hashes
	hashes := make(map[target.ItemHash]bool)
	for _, item := range result {
		hashes[item.Hash()] = true
	}
	assert.Len(t, hashes, 2, "Each target must have a unique hash after relabeling")
}

// TestNoGroupLabelAlwaysWorks verifies the no-label case works (was never broken).
func TestNoGroupLabelAlwaysWorks(t *testing.T) {
	jobName := "blackbox-no-label"
	items := buildItemsSortedLabels(jobName,
		map[model.LabelName]model.LabelValue{},
		[]string{
			"https://target-alpha.example.com:8393/",
			"https://target-beta.example.com:8393/",
		})

	prehook := New("relabel-config", logf.Log.WithName("regression-test"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: blackboxRelabelConfigs()})

	result := prehook.Apply(items)
	require.Len(t, result, 2, "Both targets must survive without group labels")
}

// blackboxRelabelConfigs returns the standard blackbox-exporter relabel pattern.
func blackboxRelabelConfigs() []*relabel.Config {
	cfgs := []*relabel.Config{
		{
			SourceLabels: model.LabelNames{"__address__"},
			Separator:    ";",
			Regex:        relabel.MustNewRegexp("(.*)"),
			TargetLabel:  "__param_target",
			Replacement:  "$1",
			Action:       relabel.Replace,
		},
		{
			SourceLabels: model.LabelNames{"__param_target"},
			Separator:    ";",
			Regex:        relabel.MustNewRegexp("(.*)"),
			TargetLabel:  "instance",
			Replacement:  "$1",
			Action:       relabel.Replace,
		},
		{
			SourceLabels: model.LabelNames{},
			Separator:    ";",
			Regex:        relabel.MustNewRegexp("(.*)"),
			TargetLabel:  "__address__",
			Replacement:  "fake-blackbox.blackbox-repro.svc.cluster.local:9115",
			Action:       relabel.Replace,
		},
	}
	for _, c := range cfgs {
		if err := c.Validate(model.UTF8Validation); err != nil {
			panic(err)
		}
	}
	return cfgs
}

// --- Workaround validation tests ---
// These replicate the BUGGY code path (struct-copy, no global sort) to verify
// that the two customer workarounds produce correct results on the released version.

// buildItemsBuggy replicates the BUGGY processTargetGroups logic from PR #4587:
// struct-copy of ScratchBuilder produces two independently sorted sublists.
func buildItemsBuggy(jobName string, groupLabelsMap map[model.LabelName]model.LabelValue, targets []model.LabelSet) []*target.Item {
	const preallocSize = 16
	groupBuilder := labels.NewScratchBuilder(preallocSize)
	for ln, lv := range groupLabelsMap {
		groupBuilder.Add(string(ln), string(lv))
	}
	groupBuilder.Sort()

	var items []*target.Item
	for _, t := range targets {
		// BUG: struct-copy shares backing array; appended target labels form
		// a second sorted sublist, but the overall sequence is NOT globally sorted.
		targetBuilder := groupBuilder
		for ln := range t {
			targetBuilder.Add(string(ln), string(t[ln]))
		}
		items = append(items, target.NewItem(jobName, string(t[model.AddressLabel]), targetBuilder.Labels(), ""))
	}
	return items
}

// buildTargetMap simulates the allocator's buildTargetMap: map[Hash]*Item dedup.
func buildTargetMap(items []*target.Item) map[target.ItemHash]*target.Item {
	m := make(map[target.ItemHash]*target.Item)
	for _, item := range items {
		m[item.Hash()] = item
	}
	return m
}

// TestBuggyBaselineConfirmsBug shows the original bug: two targets in the same
// group with a label that sorts after __address__ get the same hash after
// relabeling because Labels.Get("__address__") fails on unsorted data.
// Apply() keeps both items, but buildTargetMap deduplicates by hash → only 1 survives.
func TestBuggyBaselineConfirmsBug(t *testing.T) {
	jobName := "blackbox-buggy"
	groupLabels := map[model.LabelName]model.LabelValue{"vendor": "nginx"}
	targets := []model.LabelSet{
		{model.AddressLabel: "https://target-alpha.example.com:8393/"},
		{model.AddressLabel: "https://target-beta.example.com:8393/"},
	}

	items := buildItemsBuggy(jobName, groupLabels, targets)

	prehook := New("relabel-config", logf.Log.WithName("buggy-baseline"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: blackboxRelabelConfigs()})
	result := prehook.Apply(items)

	// Apply keeps both items (it doesn't dedup), but they have identical hashes
	// because Get("__address__") failed on unsorted labels → relabeling produced
	// identical output → identical hash. buildTargetMap collapses them.
	targetMap := buildTargetMap(result)
	assert.Len(t, targetMap, 1, "Buggy code: buildTargetMap should collapse two targets into one (confirming the bug)")
}

// TestWorkaround1_OneTargetPerGroup_StillBroken demonstrates that splitting targets
// into separate static_configs blocks does NOT fix the bug when relabel_configs read
// __address__. Even with one target per group, Labels.Get("__address__") still fails
// because the group label (vendor) sorts after __address__ in the serialized data,
// triggering the early-termination check in Get(). Both targets get identical
// post-relabel hashes and collide in buildTargetMap.
func TestWorkaround1_OneTargetPerGroup_StillBroken(t *testing.T) {
	jobName := "blackbox-workaround1"
	groupLabels := map[model.LabelName]model.LabelValue{"vendor": "nginx"}

	// Simulate two separate static_configs blocks, each with one target
	var allItems []*target.Item
	for _, addr := range []string{
		"https://target-alpha.example.com:8393/",
		"https://target-beta.example.com:8393/",
	} {
		items := buildItemsBuggy(jobName, groupLabels, []model.LabelSet{
			{model.AddressLabel: model.LabelValue(addr)},
		})
		allItems = append(allItems, items...)
	}

	prehook := New("relabel-config", logf.Log.WithName("workaround1"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: blackboxRelabelConfigs()})
	result := prehook.Apply(allItems)

	// Still broken! Labels are unsorted even with separate groups, so Get("__address__")
	// fails → relabeling produces identical output → same hash → collision.
	targetMap := buildTargetMap(result)
	assert.Len(t, targetMap, 1, "Workaround 1 is NOT valid: one target per group still collides when relabeling reads __address__")
}

// TestWorkaround2_NoGroupLabels verifies that removing group labels and using
// metric_relabel_configs instead avoids the collision even on buggy code.
// Without group labels, the ScratchBuilder only has target labels (__address__),
// which is trivially sorted — Get("__address__") always succeeds.
func TestWorkaround2_NoGroupLabels(t *testing.T) {
	jobName := "blackbox-workaround2"

	// No group labels — labels would be added via metric_relabel_configs post-scrape
	items := buildItemsBuggy(jobName, map[model.LabelName]model.LabelValue{}, []model.LabelSet{
		{model.AddressLabel: "https://target-alpha.example.com:8393/"},
		{model.AddressLabel: "https://target-beta.example.com:8393/"},
	})

	prehook := New("relabel-config", logf.Log.WithName("workaround2"))
	prehook.SetConfig(map[string][]*relabel.Config{jobName: blackboxRelabelConfigs()})
	result := prehook.Apply(items)

	targetMap := buildTargetMap(result)
	assert.Len(t, targetMap, 2, "Workaround 2: no group labels must preserve both targets after buildTargetMap")
}

// buildItemsSortedLabels replicates the fixed processTargetGroups logic:
// fresh ScratchBuilder per target with Sort() to ensure globally sorted Labels.
func buildItemsSortedLabels(jobName string, groupLabelsMap map[model.LabelName]model.LabelValue, addresses []string) []*target.Item {
	const preallocSize = 16
	groupBuilder := labels.NewScratchBuilder(preallocSize)
	targetLabelNames := make([]string, 0, preallocSize)

	for ln, lv := range groupLabelsMap {
		groupBuilder.Add(string(ln), string(lv))
	}
	groupBuilder.Sort()
	groupLabels := groupBuilder.Labels()

	var items []*target.Item
	for _, addr := range addresses {
		tb := labels.NewScratchBuilder(preallocSize)
		groupLabels.Range(func(l labels.Label) {
			tb.Add(l.Name, l.Value)
		})
		targetLabelNames = targetLabelNames[:0]
		tgt := map[model.LabelName]model.LabelValue{model.AddressLabel: model.LabelValue(addr)}
		for ln := range maps.Keys(tgt) {
			targetLabelNames = append(targetLabelNames, string(ln))
		}
		slices.Sort(targetLabelNames)
		for _, ln := range targetLabelNames {
			tb.Add(ln, string(tgt[model.LabelName(ln)]))
		}
		tb.Sort()
		items = append(items, target.NewItem(jobName, addr, tb.Labels(), ""))
	}
	return items
}
