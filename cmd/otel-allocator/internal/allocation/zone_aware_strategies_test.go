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

// zoneAwareStrategies lists the strategies that support zone-aware
// allocation. per-node intentionally does not (the node pin transitively
// pins the zone), so it's excluded from these tests.
var zoneAwareStrategies = []string{
	consistentHashingStrategyName,
	leastWeightedStrategyName,
}

// makeZoneAwareAllocator builds an allocator with a ZoneTopology attached,
// using the given strategy name and maxSkew. The returned topology shares
// state with the allocator so tests can inspect per-zone counts and
// spillover side effects.
func makeZoneAwareAllocator(t *testing.T, strategyName string, maxSkew int) (Allocator, *ZoneTopology) {
	t.Helper()
	zt, err := NewZoneTopology(logger, testTargetZoneLabel)
	require.NoError(t, err)
	a, err := New(strategyName, logger, WithMaxSkew(maxSkew), WithZoneTopology(zt))
	require.NoError(t, err)
	return a, zt
}

// uniqueURL generates per-test target URLs so consistent-hashing tests
// don't accidentally hit the same hash key across subcases.
func uniqueURL(prefix string, i int) string {
	return fmt.Sprintf("%s-%d.example.local:9100", prefix, i)
}

func TestZoneAware_SameZoneAffinity(t *testing.T) {
	// With zone awareness on and maxSkew=0, every target with a known zone
	// must be assigned to a collector in that zone — regardless of strategy.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, _ := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(6, 0, map[int]string{
				0: "zone-a", 1: "zone-a",
				2: "zone-b", 3: "zone-b",
				4: "zone-c", 5: "zone-c",
			})
			a.SetCollectors(cols)

			var items []*target.Item
			for i := range 30 {
				zone := []string{"zone-a", "zone-b", "zone-c"}[i%3]
				items = append(items, newTargetWithZone("scrape", uniqueURL("svc", i), zone))
			}
			a.SetTargets(items)

			tracked := a.TargetItems()
			require.Len(t, tracked, 30)
			collectorsNow := a.Collectors()
			for _, it := range tracked {
				wantZone := it.Labels.Get(testTargetZoneLabel)
				gotCollector, ok := collectorsNow[it.CollectorName]
				require.True(t, ok, "assigned collector %q must exist", it.CollectorName)
				assert.Equal(t, wantZone, gotCollector.Zone,
					"target wanting %q ended up on %q which is in %q",
					wantZone, gotCollector.Name, gotCollector.Zone)
			}
		})
	}
}

func TestZoneAware_FailoverToGlobalWhenZoneEmpty(t *testing.T) {
	// When a target's desired zone has no collectors, the allocator must
	// fall back to the global pool. The spillover counter records the
	// origin (the uncovered zone) — we verify the topology side-effect
	// via the public UncoveredZones() helper instead of counter values.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, zt := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
			})
			a.SetCollectors(cols)

			// All targets want zone-c, which has no collectors.
			var items []*target.Item
			for i := range 6 {
				items = append(items, newTargetWithZone("scrape", uniqueURL("orphan", i), "zone-c"))
			}
			a.SetTargets(items)

			assert.Equal(t, []string{"zone-c"}, zt.UncoveredZones(),
				"zone-c has 6 desiring targets but no collectors — must be reported uncovered")

			// Every target must still be assigned (failover succeeded).
			for _, it := range a.TargetItems() {
				assert.NotEmpty(t, it.CollectorName,
					"failover path must still produce a valid assignment")
			}
		})
	}
}

func TestZoneAware_MaxSkewZero_NoSpillover(t *testing.T) {
	// With maxSkew=0, the strategy must never spill cross-zone purely for
	// load reasons. We construct a skewed setup (lots of targets wanting
	// zone-a, only one collector in zone-a; zone-b has plenty of capacity)
	// and verify that every zone-a target stays on the zone-a collector.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, _ := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
				2: "zone-b",
			})
			a.SetCollectors(cols)

			var items []*target.Item
			for i := range 50 {
				items = append(items, newTargetWithZone("scrape", uniqueURL("hot-a", i), "zone-a"))
			}
			a.SetTargets(items)

			collectorsNow := a.Collectors()
			zoneACollector := collectorsNow["collector-0"]
			require.NotNil(t, zoneACollector)
			assert.Equal(t, 50, zoneACollector.NumTargets,
				"with maxSkew=0 every zone-a target must stay on the zone-a collector")
		})
	}
}

func TestZoneAware_MaxSkewTriggersSpillover(t *testing.T) {
	// With maxSkew > 0, severe load imbalance must cause spillover so the
	// global collectors don't sit idle while one collector drowns.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			const maxSkew = 5
			a, _ := makeZoneAwareAllocator(t, s, maxSkew)
			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a", // sole collector for the hot zone
				1: "zone-b",
				2: "zone-b",
			})
			a.SetCollectors(cols)

			var items []*target.Item
			for i := range 50 {
				items = append(items, newTargetWithZone("scrape", uniqueURL("hot-skew", i), "zone-a"))
			}
			a.SetTargets(items)

			collectorsNow := a.Collectors()
			counts := make(map[string]int, len(collectorsNow))
			for name, c := range collectorsNow {
				counts[name] = c.NumTargets
			}
			minLoad := counts["collector-0"]
			maxLoad := counts["collector-0"]
			for _, n := range counts {
				if n < minLoad {
					minLoad = n
				}
				if n > maxLoad {
					maxLoad = n
				}
			}

			// The post-assignment skew must respect the configured limit.
			// We allow skew == maxSkew (boundary OK) but not skew > maxSkew.
			assert.LessOrEqual(t, maxLoad-minLoad, maxSkew,
				"observed skew (%d-%d=%d) exceeded maxSkew=%d",
				maxLoad, minLoad, maxLoad-minLoad, maxSkew)
			// The zone-b collectors must have picked up real work from
			// spillover — otherwise the skew would still be 50.
			assert.Greater(t, counts["collector-1"]+counts["collector-2"], 0,
				"spillover must have placed some targets on zone-b collectors")
		})
	}
}

func TestZoneAware_ZonelessTargetUsesGlobalPool(t *testing.T) {
	// Targets that don't carry the zone label must still be assigned; they
	// are zone-agnostic and travel through the global path.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, _ := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
				2: "zone-c",
			})
			a.SetCollectors(cols)

			items := []*target.Item{
				target.NewItem("scrape", "no-zone-1:9100",
					labels.New(labels.Label{Name: "instance", Value: "x1"}), ""),
				target.NewItem("scrape", "no-zone-2:9100",
					labels.New(labels.Label{Name: "instance", Value: "x2"}), ""),
			}
			a.SetTargets(items)

			for _, it := range a.TargetItems() {
				assert.NotEmpty(t, it.CollectorName,
					"zone-less targets must be assigned via the global pool")
			}
		})
	}
}

func TestZoneAwareConsistentHashing_DeterministicWithinZone(t *testing.T) {
	// Determinism guarantee: with the same collector set and target, a
	// consistent-hashing strategy must always pick the same collector.
	// This protects against accidental non-determinism in the per-zone
	// ring construction.
	a1, _ := makeZoneAwareAllocator(t, consistentHashingStrategyName, 0)
	a2, _ := makeZoneAwareAllocator(t, consistentHashingStrategyName, 0)
	cols := MakeNCollectorsWithZones(4, 0, map[int]string{
		0: "zone-a", 1: "zone-a",
		2: "zone-b", 3: "zone-b",
	})
	a1.SetCollectors(cols)
	a2.SetCollectors(cols)

	items := []*target.Item{
		newTargetWithZone("scrape", "deterministic-1:9100", "zone-a"),
		newTargetWithZone("scrape", "deterministic-2:9100", "zone-a"),
		newTargetWithZone("scrape", "deterministic-3:9100", "zone-b"),
	}
	a1.SetTargets(items)
	a2.SetTargets(items)

	t1 := a1.TargetItems()
	t2 := a2.TargetItems()
	require.Equal(t, len(t1), len(t2))
	for hash, item := range t1 {
		other, ok := t2[hash]
		require.True(t, ok)
		assert.Equal(t, item.CollectorName, other.CollectorName,
			"target %s assigned to %s in run 1 vs %s in run 2",
			item.TargetURL, item.CollectorName, other.CollectorName)
	}
}

func TestZoneAware_LateTopologyAttachReassignsExistingTargets(t *testing.T) {
	// Regression for the "stuck cross-zone" bug: when targets are
	// loaded BEFORE zone awareness is enabled, they get placed by the
	// pre-feature global path and end up cross-zone. Attaching a
	// topology afterwards must re-run the assignment so those targets
	// move into their desired zones — without this step the targets
	// stay cross-zone until the next discovery cycle, silently
	// breaking the egress-cost guarantee operators expect from
	// zone-aware allocation.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, err := New(s, logger)
			require.NoError(t, err)
			// Collectors AND targets arrive while zone-aware is off.
			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
				2: "zone-c",
			})
			a.SetCollectors(cols)
			a.SetTargets([]*target.Item{
				newTargetWithZone("scrape", "preloaded-a:9100", "zone-a"),
				newTargetWithZone("scrape", "preloaded-b:9100", "zone-b"),
				newTargetWithZone("scrape", "preloaded-c:9100", "zone-c"),
			})

			// Every target now sits on some collector via the global
			// path. Attach the topology — they must move to their
			// desired zones.
			zt, err := NewZoneTopology(logger, testTargetZoneLabel)
			require.NoError(t, err)
			a.SetZoneTopology(zt)

			collectorsNow := a.Collectors()
			for _, it := range a.TargetItems() {
				desired := it.Labels.Get(testTargetZoneLabel)
				assigned := collectorsNow[it.CollectorName].Zone
				assert.Equal(t, desired, assigned,
					"target %q wants zone %q but stayed on %q — late topology attach did not re-run assignment",
					it.TargetURL, desired, assigned)
			}
		})
	}
}

func TestZoneAware_LateTopologyAttachRebuildsZoneCache(t *testing.T) {
	// Regression for the "late attach" bug: when SetZoneTopology is
	// called after SetCollectors has already populated the strategy,
	// the strategy must rebuild its per-zone state on the toggle.
	// Without this, least-weighted's collectorsByZone would still be
	// nil and zone-targeted assignments would silently fall through
	// to the failover path until the next collector reconcile.
	// Exercises both zone-aware strategies because each maintains its
	// own per-zone cache (rings for CH, map for LW).
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, err := New(s, logger)
			require.NoError(t, err)
			// Collectors arrive first, zone-aware is OFF at this point.
			cols := MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
				2: "zone-c",
			})
			a.SetCollectors(cols)

			// Now attach a topology AFTER collectors are already loaded.
			zt, err := NewZoneTopology(logger, testTargetZoneLabel)
			require.NoError(t, err)
			a.SetZoneTopology(zt)

			// Targets must reach their desired zone — the per-zone cache
			// must have been rebuilt by SetZoneAwareness even though
			// SetCollectors was never re-invoked.
			a.SetTargets([]*target.Item{
				newTargetWithZone("scrape", "late-a:9100", "zone-a"),
				newTargetWithZone("scrape", "late-b:9100", "zone-b"),
				newTargetWithZone("scrape", "late-c:9100", "zone-c"),
			})
			collectorsNow := a.Collectors()
			for _, it := range a.TargetItems() {
				desired := it.Labels.Get(testTargetZoneLabel)
				assigned := collectorsNow[it.CollectorName].Zone
				assert.Equal(t, desired, assigned,
					"target %q wants zone %q but landed in %q — late SetZoneTopology did not rebuild the per-zone cache",
					it.TargetURL, desired, assigned)
			}
		})
	}
}

func TestZoneAwareConsistentHashing_ToggleZoneAwarenessRebuildsRings(t *testing.T) {
	// Verify SetZoneAwareness can flip the strategy in both directions
	// without leaving stale per-zone rings around. We exercise both
	// transitions and check that allocation still succeeds afterwards.
	a, err := New(consistentHashingStrategyName, logger)
	require.NoError(t, err)
	cols := MakeNCollectorsWithZones(3, 0, map[int]string{
		0: "zone-a",
		1: "zone-b",
		2: "zone-c",
	})
	a.SetCollectors(cols)
	a.SetTargets([]*target.Item{
		newTargetWithZone("scrape", "toggle-1:9100", "zone-a"),
	})
	for _, it := range a.TargetItems() {
		require.NotEmpty(t, it.CollectorName, "global mode must assign targets")
	}

	// Enable zone awareness — the next assignment must respect zones.
	zt, err := NewZoneTopology(logger, testTargetZoneLabel)
	require.NoError(t, err)
	a.SetZoneTopology(zt)
	a.SetTargets([]*target.Item{
		newTargetWithZone("scrape", "toggle-2:9100", "zone-b"),
	})
	collectorsNow := a.Collectors()
	for _, it := range a.TargetItems() {
		c := collectorsNow[it.CollectorName]
		assert.Equal(t, it.Labels.Get(testTargetZoneLabel), c.Zone,
			"after enabling zone awareness, target must land in its zone")
	}

	// Disable again — strategy should drop the per-zone rings and still work.
	a.SetZoneTopology(nil)
	a.SetTargets([]*target.Item{
		newTargetWithZone("scrape", "toggle-3:9100", "zone-c"),
	})
	for _, it := range a.TargetItems() {
		assert.NotEmpty(t, it.CollectorName,
			"after disabling zone awareness, allocation must continue to work")
	}
}

func TestZoneAware_RecordsSpilloverOnFailover(t *testing.T) {
	// The ZoneTopology snapshot doesn't expose spillover counts directly
	// (those go to the metric), but failover by definition produces an
	// uncovered zone — verify that's visible.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, zt := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-a",
			})
			a.SetCollectors(cols)
			a.SetTargets([]*target.Item{
				newTargetWithZone("scrape", uniqueURL("failover", 1), "zone-d"),
			})
			assert.Contains(t, zt.UncoveredZones(), "zone-d",
				"a target for an uncovered zone must mark that zone uncovered")
		})
	}
}

func TestZoneAwareLeastWeighted_HonorsReassignmentFastPath(t *testing.T) {
	// The fast-path in leastWeighted's GetCollectorForTarget keeps an
	// already-assigned target on its existing collector when that
	// collector is still alive. This is essential for stability across
	// reconciles. Verify zone-awareness doesn't accidentally break it.
	a, _ := makeZoneAwareAllocator(t, leastWeightedStrategyName, 0)
	cols := MakeNCollectorsWithZones(3, 0, map[int]string{
		0: "zone-a",
		1: "zone-a",
		2: "zone-b",
	})
	a.SetCollectors(cols)
	items := []*target.Item{
		newTargetWithZone("scrape", "sticky:9100", "zone-a"),
	}
	a.SetTargets(items)

	firstAssign := ""
	for _, it := range a.TargetItems() {
		firstAssign = it.CollectorName
	}
	require.NotEmpty(t, firstAssign)

	// Re-applying the same target set must not move the target around.
	a.SetTargets(items)
	for _, it := range a.TargetItems() {
		assert.Equal(t, firstAssign, it.CollectorName,
			"reassignment fast-path must keep stable assignment across SetTargets calls")
	}
}

func TestZoneAware_FailedOverTargetsRecoverWhenZoneCollectorAppears(t *testing.T) {
	// Regression for the "stuck failover" bug: when a target's desired zone
	// initially has no collectors it lands cross-zone via the global pool.
	// If a same-zone collector subsequently spins up, the next
	// reconciliation must re-evaluate the assignment instead of letting
	// least-weighted's sticky fast-path keep the target cross-zone.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, zt := makeZoneAwareAllocator(t, s, 0)
			// Start with no collectors in the target's desired zone.
			a.SetCollectors(MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
			}))
			items := []*target.Item{
				newTargetWithZone("scrape", "stuck-failover:9100", "zone-c"),
			}
			a.SetTargets(items)
			require.Contains(t, zt.UncoveredZones(), "zone-c",
				"precondition: zone-c must report uncovered before the collector arrives")

			// A collector now spins up in zone-c. The previously-failed-over
			// target must move back home.
			a.SetCollectors(MakeNCollectorsWithZones(3, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
				2: "zone-c",
			}))
			assert.Empty(t, zt.UncoveredZones(), "uncovered set must clear once zone-c has a collector")
			collectorsNow := a.Collectors()
			for _, it := range a.TargetItems() {
				assignedZone := collectorsNow[it.CollectorName].Zone
				assert.Equal(t, "zone-c", assignedZone,
					"target wanting zone-c must recover to the new zone-c collector after the previous failover, got %q",
					assignedZone)
			}
		})
	}
}

func TestZoneAware_CollectorZoneChangeReshufflesAssignments(t *testing.T) {
	// Regression for diff.Maps blindness: when a Collector's Zone field
	// changes but its name stays the same (e.g. StatefulSet pod
	// rescheduled onto a node in a different AZ, or node-zone resolver
	// finally learns the zone of a previously-unknown node), the
	// allocator must propagate the update so targets land on the new
	// zone, not the old one.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, _ := makeZoneAwareAllocator(t, s, 0)
			a.SetCollectors(MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
			}))
			a.SetTargets([]*target.Item{
				newTargetWithZone("scrape", "a-target:9100", "zone-a"),
				newTargetWithZone("scrape", "b-target:9100", "zone-b"),
			})

			// Flip collector-1 from zone-b to zone-c (same name, new zone).
			// This is the "zone-only change" case that diff.Maps misses.
			a.SetCollectors(MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-c",
			}))
			collectorsNow := a.Collectors()
			for _, it := range a.TargetItems() {
				desired := it.Labels.Get(testTargetZoneLabel)
				assignedZone := collectorsNow[it.CollectorName].Zone
				switch desired {
				case "zone-a":
					assert.Equal(t, "zone-a", assignedZone,
						"zone-a target must stay on the zone-a collector")
				case "zone-b":
					// zone-b is no longer covered — must failover via global pool.
					assert.NotEqual(t, "", assignedZone,
						"failed-over zone-b target must still be assigned somewhere")
				}
			}
		})
	}
}

func TestZoneAware_SetZoneTopologyRepeatedHydrationDoesNotDoubleCount(t *testing.T) {
	// Repeatedly attaching the same ZoneTopology (e.g. on config reload)
	// must not accumulate per-zone target counts. The hydration path is
	// expected to Reset() the topology before re-counting.
	zt, err := NewZoneTopology(logger, testTargetZoneLabel)
	require.NoError(t, err)
	a, err := New(leastWeightedStrategyName, logger, WithZoneTopology(zt))
	require.NoError(t, err)
	a.SetCollectors(MakeNCollectorsWithZones(2, 0, map[int]string{
		0: "zone-a",
		1: "zone-b",
	}))
	a.SetTargets([]*target.Item{
		newTargetWithZone("scrape", "first:9100", "zone-a"),
		newTargetWithZone("scrape", "second:9100", "zone-b"),
	})

	// Re-attach the same topology a few times. The post-rehydrate target
	// counts must match a single hydration, not multiples.
	a.SetZoneTopology(zt)
	a.SetZoneTopology(zt)
	a.SetZoneTopology(zt)

	snap := snapshotByZone(zt.Snapshot())
	assert.Equal(t, 1, snap["zone-a"].TargetsDesired,
		"zone-a count must stay at 1 across repeated SetZoneTopology calls (got %d)", snap["zone-a"].TargetsDesired)
	assert.Equal(t, 1, snap["zone-b"].TargetsDesired,
		"zone-b count must stay at 1 across repeated SetZoneTopology calls (got %d)", snap["zone-b"].TargetsDesired)
}

func TestZoneAware_TopologyTargetCountsTrackDesiredZone(t *testing.T) {
	// Even when targets spill cross-zone (via failover or maxSkew), the
	// topology must continue to attribute them to their *desired* zone,
	// not their assigned collector's zone. This keeps the /zones API
	// honest about what the workload wants vs what it got.
	for _, s := range zoneAwareStrategies {
		t.Run(s, func(t *testing.T) {
			a, zt := makeZoneAwareAllocator(t, s, 0)
			cols := MakeNCollectorsWithZones(2, 0, map[int]string{
				0: "zone-a",
				1: "zone-b",
			})
			a.SetCollectors(cols)

			items := []*target.Item{
				newTargetWithZone("scrape", uniqueURL("desired-a", 1), "zone-a"),
				newTargetWithZone("scrape", uniqueURL("desired-c", 1), "zone-c"), // failover
				newTargetWithZone("scrape", uniqueURL("desired-c", 2), "zone-c"), // failover
			}
			a.SetTargets(items)

			snap := snapshotByZone(zt.Snapshot())
			assert.Equal(t, 1, snap["zone-a"].TargetsDesired)
			assert.Equal(t, 2, snap["zone-c"].TargetsDesired,
				"desired-zone counting must include targets that ended up spilled")
			assert.False(t, snap["zone-c"].Covered)
		})
	}
}
