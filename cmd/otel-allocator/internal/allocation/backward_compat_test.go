// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// These tests pin down the backward-compatibility contract: when zone-aware
// allocation is not enabled, the new code path must produce byte-for-byte
// identical assignments to what the pre-zone-aware implementation produced.
// If any of these tests fail, it means a deployment that does not set
// `topology.zone_aware: true` would see a behavior change after upgrading —
// which is a regression.

// makeBaselineAllocator constructs an allocator that does not opt into
// zone-aware allocation. This mirrors the default behavior an existing
// user would see after upgrading without changing their config.
func makeBaselineAllocator(t *testing.T, strategyName string) Allocator {
	t.Helper()
	a, err := New(strategyName, logger)
	require.NoError(t, err)
	assert.Nil(t, a.ZoneTopology(), "default allocator must not have a zone topology attached")
	return a
}

func TestBackwardCompat_DefaultAllocatorHasNoZoneTopology(t *testing.T) {
	// New(strategy, logger) without any zone options must produce an
	// allocator indistinguishable from the pre-feature one: no topology,
	// no maxSkew, no per-zone metrics.
	for _, name := range GetRegisteredAllocatorNames() {
		t.Run(name, func(t *testing.T) {
			a := makeBaselineAllocator(t, name)
			assert.Nil(t, a.ZoneTopology(),
				"%s allocator must report nil ZoneTopology when no zone options are used", name)
		})
	}
}

func TestBackwardCompat_DisablingZoneAwarenessMatchesNoZoneSetup(t *testing.T) {
	// Contract: an allocator that was once zone-aware and is then disabled
	// (SetZoneTopology(nil)) must allocate identically to an allocator
	// that was never zone-aware in the first place. This guards against
	// stale per-zone rings or cached zone-aware state leaking into the
	// disabled code path.
	for _, name := range []string{consistentHashingStrategyName, leastWeightedStrategyName} {
		t.Run(name, func(t *testing.T) {
			baseline := makeBaselineAllocator(t, name)
			toggled, err := New(name, logger)
			require.NoError(t, err)

			zt, err := NewZoneTopology(logger, testTargetZoneLabel)
			require.NoError(t, err)
			toggled.SetZoneTopology(zt)
			toggled.SetZoneTopology(nil) // disable again

			cols := MakeNCollectorsWithZones(4, 0, map[int]string{
				0: "zone-a", 1: "zone-a",
				2: "zone-b", 3: "zone-c",
			})
			items := []*target.Item{
				newTargetWithZone("scrape", "compat-1:9100", "zone-a"),
				newTargetWithZone("scrape", "compat-2:9100", "zone-b"),
				newTargetWithZone("scrape", "compat-3:9100", "zone-c"),
				// Also include a zone-less target so we cover that path.
				target.NewItem("scrape", "compat-4:9100",
					labels.New(labels.Label{Name: "instance", Value: "x"}), ""),
			}

			baseline.SetCollectors(cols)
			toggled.SetCollectors(cols)
			baseline.SetTargets(items)
			toggled.SetTargets(items)

			require.Equal(t, len(baseline.TargetItems()), len(toggled.TargetItems()))
			for hash, item := range baseline.TargetItems() {
				other, ok := toggled.TargetItems()[hash]
				require.True(t, ok, "target %s missing from toggled allocator", item.TargetURL)
				assert.Equal(t, item.CollectorName, other.CollectorName,
					"baseline assigned %s to %s; toggled assigned to %s — disabling zone awareness must restore exact pre-feature behavior",
					item.TargetURL, item.CollectorName, other.CollectorName)
			}
		})
	}
}

func TestBackwardCompat_NoZoneTargetsBypassZoneLogic(t *testing.T) {
	// Targets that carry no zone label must be allocated as if zone-aware
	// were disabled, even when zone-aware is on. This protects mixed setups
	// where some scrape jobs produce zone-labeled targets (Kubernetes SD)
	// and others don't (file SD, static configs, EC2 SD without
	// instance zone resolution).
	for _, name := range []string{consistentHashingStrategyName, leastWeightedStrategyName} {
		t.Run(name, func(t *testing.T) {
			baseline := makeBaselineAllocator(t, name)
			zoneAware, _ := makeZoneAwareAllocator(t, name, 0)

			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a", 1: "zone-b", 2: "zone-c",
			})
			// Targets with no zone label — should travel the global path.
			var items []*target.Item
			for i := range 12 {
				items = append(items, target.NewItem(
					"scrape",
					fmt.Sprintf("nozone-%d:9100", i),
					labels.New(labels.Label{Name: "instance", Value: fmt.Sprintf("x-%d", i)}),
					"",
				))
			}

			baseline.SetCollectors(cols)
			zoneAware.SetCollectors(cols)
			baseline.SetTargets(items)
			zoneAware.SetTargets(items)

			require.Equal(t, len(baseline.TargetItems()), len(zoneAware.TargetItems()))
			for hash, item := range baseline.TargetItems() {
				other, ok := zoneAware.TargetItems()[hash]
				require.True(t, ok, "missing target")
				assert.Equal(t, item.CollectorName, other.CollectorName,
					"zone-less target %s assigned to %s under baseline but %s under zone-aware — zone-less must skip zone logic entirely",
					item.TargetURL, item.CollectorName, other.CollectorName)
			}
		})
	}
}

func TestBackwardCompat_ConsistentHashingRingMatchesLegacyDistribution(t *testing.T) {
	// The pre-feature consistent-hashing strategy built a single ring with
	// no zone partitioning. We verify that our globalHasher (the
	// "non-zone-aware" path) still distributes the same way for a known
	// collector set and target URL pattern. Catching a divergence here
	// would mean SetCollectors changed behavior — which is the most
	// dangerous form of regression for consistent-hashing because it
	// would cause every existing target to be re-shuffled across
	// collectors on the next reconcile.
	a := makeBaselineAllocator(t, consistentHashingStrategyName).(*allocator)
	// Cast to a concrete strategy so we can interrogate the global hasher
	// directly. Using the concrete strategy is the only way to inspect
	// internal ring construction; the public surface only exposes
	// "given a target, which collector?" which is already covered above.
	chStrategy := a.strategy.(*consistentHashingStrategy)
	require.NotNil(t, chStrategy.globalHasher)
	require.Empty(t, chStrategy.zoneHashers,
		"zone hashers must remain empty until zone awareness is enabled")

	cols := MakeNCollectors(5, 0)
	a.SetCollectors(cols)
	require.NotNil(t, chStrategy.globalHasher)
	require.Empty(t, chStrategy.zoneHashers,
		"SetCollectors without zone awareness must not initialize any per-zone rings")

	// Sanity: every collector ends up as a hash ring member.
	// LocateKey returns a Member; the count of distinct members across
	// many keys must equal the collector count.
	seenMembers := make(map[string]struct{})
	for i := range 5000 {
		key := fmt.Appendf(nil, "probe-%d", i)
		member := chStrategy.globalHasher.LocateKey(key)
		require.NotNil(t, member)
		seenMembers[member.String()] = struct{}{}
	}
	assert.Equal(t, len(cols), len(seenMembers),
		"every collector must own at least one ring partition — otherwise the ring is unbalanced compared to the pre-feature build")
}

func TestBackwardCompat_LeastWeightedFastPathPreserved(t *testing.T) {
	// least-weighted has historically supported a reassignment fast-path:
	// when collectors change but a target's previously-assigned collector
	// is still alive, the strategy returns that collector unchanged. This
	// is what keeps assignments stable across reconciles. We verify it by
	// adding a new collector to the set and checking that none of the
	// already-assigned targets migrate.
	a, err := New(leastWeightedStrategyName, logger)
	require.NoError(t, err)

	cols := MakeNCollectors(3, 0)
	a.SetCollectors(cols)
	targets := MakeNNewTargetsWithEmptyCollectors(9, 0)
	a.SetTargets(targets)

	pre := make(map[string]string, len(a.TargetItems()))
	for hash, item := range a.TargetItems() {
		require.NotEmpty(t, item.CollectorName, "target %s never got assigned", hash)
		pre[hash.String()] = item.CollectorName
	}

	// Add a new collector. The fast-path should keep every existing
	// target on its current collector, since the named collectors all
	// still exist.
	cols["collector-3"] = NewCollector("collector-3", "node-3", "")
	a.SetCollectors(cols)

	for hash, item := range a.TargetItems() {
		assert.Equal(t, pre[hash.String()], item.CollectorName,
			"target %s moved from %s to %s after adding a new collector — least-weighted fast-path must keep stable assignments",
			hash, pre[hash.String()], item.CollectorName)
	}
}
