// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"sync"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const testTargetZoneLabel = "__meta_kubernetes_endpointslice_endpoint_zone"

func newTestZoneTopology(t *testing.T) *ZoneTopology {
	t.Helper()
	zt, err := NewZoneTopology(logf.Log.WithName("test"), testTargetZoneLabel)
	require.NoError(t, err)
	return zt
}

func newTargetWithZone(jobName, url, zone string) *target.Item {
	lbls := labels.New(
		labels.Label{Name: testTargetZoneLabel, Value: zone},
		labels.Label{Name: "instance", Value: url},
	)
	return target.NewItem(jobName, url, lbls, "")
}

func TestZoneTopology_GetTargetZone(t *testing.T) {
	tests := []struct {
		name      string
		labelKey  string
		labels    []labels.Label
		nilTarget bool
		want      string
	}{
		{
			name:     "zone label present",
			labelKey: testTargetZoneLabel,
			labels: []labels.Label{
				{Name: testTargetZoneLabel, Value: "us-east-1a"},
			},
			want: "us-east-1a",
		},
		{
			name:     "zone label missing",
			labelKey: testTargetZoneLabel,
			labels: []labels.Label{
				{Name: "instance", Value: "10.0.0.1"},
			},
			want: "",
		},
		{
			name:     "empty configured label disables extraction",
			labelKey: "",
			labels: []labels.Label{
				{Name: testTargetZoneLabel, Value: "us-east-1a"},
			},
			want: "",
		},
		{
			name:      "nil target",
			labelKey:  testTargetZoneLabel,
			nilTarget: true,
			want:      "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			zt, err := NewZoneTopology(logf.Log.WithName("test"), tc.labelKey)
			require.NoError(t, err)
			var item *target.Item
			if !tc.nilTarget {
				item = target.NewItem("job", "url", labels.New(tc.labels...), "")
			}
			assert.Equal(t, tc.want, zt.GetTargetZone(item))
		})
	}
}

// stubNodeZoneLookup is a deterministic in-memory zone resolver used in the
// node-fallback unit tests. It implements the nodeZoneLookup interface
// without pulling in the K8s fake client.
type stubNodeZoneLookup map[string]string

func (s stubNodeZoneLookup) GetZone(nodeName string) string {
	return s[nodeName]
}

func TestZoneTopology_HighCardinality_CountsCollectorsToo(t *testing.T) {
	// Regression for the "warning only checks targets" gap: a bad
	// zoneLabel (node-side) blows up collectors_per_zone series just
	// like a bad target_zone_label blows up targets_per_zone. The
	// guard must union both sides so either misconfiguration trips
	// the one-time WARN. The test injects 64 distinct collector zones
	// (no target zones) and asserts the warning fires.
	zt := newTestZoneTopology(t)
	collectors := make(map[string]*Collector, cardinalityWarnThreshold+1)
	for i := 0; i <= cardinalityWarnThreshold; i++ {
		name := fmt.Sprintf("collector-%d", i)
		zone := fmt.Sprintf("syntheticzone-%d", i)
		collectors[name] = NewCollector(name, fmt.Sprintf("node-%d", i), zone)
	}
	zt.SetCollectors(collectors)

	zt.mu.RLock()
	defer zt.mu.RUnlock()
	assert.True(t, zt.cardinalityWarningEmitted,
		"collector-side cardinality crossing the threshold must trip the warning, not just target-side")
}

func TestZoneTopology_GetTargetZone_FallbackToNodeResolver(t *testing.T) {
	// The two-stage lookup contract: target zone label wins when present;
	// otherwise the node-name resolver fills in. This is the path that
	// keeps zone-aware allocation working for Pod SD, classic Endpoints
	// SD, and static configs that emit a node label but no zone label.
	resolver := stubNodeZoneLookup{
		"node-a": "us-east-1a",
		"node-b": "us-east-1b",
	}
	zt, err := NewZoneTopology(logf.Log.WithName("test"), testTargetZoneLabel)
	require.NoError(t, err)
	zt.WithNodeZoneResolver(resolver)

	tests := []struct {
		name   string
		labels []labels.Label
		want   string
	}{
		{
			name: "explicit zone label wins over node fallback",
			labels: []labels.Label{
				{Name: testTargetZoneLabel, Value: "us-east-1c"},
				{Name: "__meta_kubernetes_pod_node_name", Value: "node-a"},
			},
			want: "us-east-1c",
		},
		{
			name: "no zone label, node fallback resolves",
			labels: []labels.Label{
				{Name: "__meta_kubernetes_pod_node_name", Value: "node-b"},
			},
			want: "us-east-1b",
		},
		{
			name: "no zone label and unknown node returns empty",
			labels: []labels.Label{
				{Name: "__meta_kubernetes_pod_node_name", Value: "node-unknown"},
			},
			want: "",
		},
		{
			name: "no zone label and no node name returns empty",
			labels: []labels.Label{
				{Name: "instance", Value: "x"},
			},
			want: "",
		},
		{
			name: "endpointslice node-kind fallback also resolves",
			labels: []labels.Label{
				{Name: "__meta_kubernetes_endpointslice_address_target_kind", Value: "Node"},
				{Name: "__meta_kubernetes_endpointslice_address_target_name", Value: "node-a"},
			},
			want: "us-east-1a",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := target.NewItem("scrape", "10.0.0.1:9100", labels.New(tc.labels...), "")
			assert.Equal(t, tc.want, zt.GetTargetZone(item))
		})
	}
}

func TestZoneTopology_GetTargetZone_NoResolverFallsThrough(t *testing.T) {
	// Without a node resolver attached, the topology behaves as if the
	// node fallback didn't exist: label-only extraction, no implicit
	// cluster knowledge. This preserves the documented label-only mode.
	zt, err := NewZoneTopology(logf.Log.WithName("test"), testTargetZoneLabel)
	require.NoError(t, err)
	item := target.NewItem("scrape", "10.0.0.1:9100", labels.New(
		labels.Label{Name: "__meta_kubernetes_pod_node_name", Value: "node-a"},
	), "")
	assert.Equal(t, "", zt.GetTargetZone(item),
		"node fallback must not run when no resolver is attached")
}

func TestZoneTopology_SetCollectors_BuildsPerZoneIndex(t *testing.T) {
	zt := newTestZoneTopology(t)

	collectors := map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-a-2": NewCollector("c-a-2", "node-a-2", "us-east-1a"),
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
		"c-x":   NewCollector("c-x", "node-x", ""), // zone-less
	}
	zt.SetCollectors(collectors)

	assert.ElementsMatch(t, []string{"us-east-1a", "us-east-1b"}, zt.Zones(),
		"Zones() must omit the zone-less bucket")
	assert.Equal(t, []string{"c-a-1", "c-a-2"}, zt.CollectorsInZone("us-east-1a"))
	assert.Equal(t, []string{"c-b-1"}, zt.CollectorsInZone("us-east-1b"))
	assert.Equal(t, []string{"c-x"}, zt.CollectorsInZone(""),
		"zone-less collectors must be queryable via the empty zone")
	assert.Nil(t, zt.CollectorsInZone("us-west-2a"),
		"unknown zones must return nil, not an empty slice")
}

func TestZoneTopology_SetCollectors_ReplacesPreviousIndex(t *testing.T) {
	zt := newTestZoneTopology(t)

	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
	})
	// Replace the collector set entirely: zone-b disappears, zone-c appears.
	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-c-1": NewCollector("c-c-1", "node-c-1", "us-east-1c"),
	})

	assert.ElementsMatch(t, []string{"us-east-1a", "us-east-1c"}, zt.Zones())
	assert.Nil(t, zt.CollectorsInZone("us-east-1b"),
		"zone-b must be removed entirely once its last collector leaves")
}

func TestZoneTopology_TargetCounts(t *testing.T) {
	zt := newTestZoneTopology(t)

	zt.IncrementTargetCount("us-east-1a")
	zt.IncrementTargetCount("us-east-1a")
	zt.IncrementTargetCount("us-east-1b")
	zt.IncrementTargetCount("") // zone-less

	snap := snapshotByZone(zt.Snapshot())
	assert.Equal(t, 2, snap["us-east-1a"].TargetsDesired)
	assert.Equal(t, 1, snap["us-east-1b"].TargetsDesired)
	assert.Equal(t, 1, snap[""].TargetsDesired)

	zt.DecrementTargetCount("us-east-1a")
	zt.DecrementTargetCount("us-east-1b")

	snap = snapshotByZone(zt.Snapshot())
	assert.Equal(t, 1, snap["us-east-1a"].TargetsDesired)
	assert.NotContains(t, snap, "us-east-1b",
		"once the count for a zone reaches zero, the zone must be dropped from the index")
}

func TestZoneTopology_DecrementBelowZeroIsSafe(t *testing.T) {
	// Defensive: extra decrements (e.g. from a buggy caller) must not produce
	// negative counts or panic. The expected behavior is that the zone is
	// simply dropped from the index.
	zt := newTestZoneTopology(t)
	zt.DecrementTargetCount("us-east-1a")
	zt.DecrementTargetCount("us-east-1a")
	snap := snapshotByZone(zt.Snapshot())
	assert.NotContains(t, snap, "us-east-1a")
}

func TestZoneTopology_UncoveredZones(t *testing.T) {
	zt := newTestZoneTopology(t)

	// Targets desire zones a, b, c, and "" (zone-less).
	zt.IncrementTargetCount("us-east-1a")
	zt.IncrementTargetCount("us-east-1b")
	zt.IncrementTargetCount("us-east-1c")
	zt.IncrementTargetCount("")

	// Only zone-a has a collector. Zones b and c are uncovered. The zone-less
	// bucket is never uncovered because zone-less targets are served by the
	// global collector pool, not by missing-zone failover.
	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
	})

	assert.Equal(t, []string{"us-east-1b", "us-east-1c"}, zt.UncoveredZones())

	snap := snapshotByZone(zt.Snapshot())
	assert.True(t, snap["us-east-1a"].Covered)
	assert.False(t, snap["us-east-1b"].Covered)
	assert.False(t, snap["us-east-1c"].Covered)
	assert.True(t, snap[""].Covered, "the zone-less bucket must report as covered")
}

func TestZoneTopology_UncoveredRecoveryWhenCollectorAppears(t *testing.T) {
	zt := newTestZoneTopology(t)
	zt.IncrementTargetCount("us-east-1b")
	assert.Equal(t, []string{"us-east-1b"}, zt.UncoveredZones())

	// Spinning up a collector in zone-b must close the coverage gap.
	zt.SetCollectors(map[string]*Collector{
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
	})
	assert.Empty(t, zt.UncoveredZones())
}

func TestZoneTopology_Snapshot_SortedAndStable(t *testing.T) {
	zt := newTestZoneTopology(t)
	zt.SetCollectors(map[string]*Collector{
		"c-c-1": NewCollector("c-c-1", "node-c-1", "us-east-1c"),
		"c-a-2": NewCollector("c-a-2", "node-a-2", "us-east-1a"),
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
	})
	zt.IncrementTargetCount("us-east-1a")
	zt.IncrementTargetCount("us-east-1d") // targets without a matching collector

	snap := zt.Snapshot()
	require.Len(t, snap, 4)

	// Zones are sorted alphabetically including the uncovered one.
	want := []string{"us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d"}
	got := make([]string, len(snap))
	for i, z := range snap {
		got[i] = z.Zone
	}
	assert.Equal(t, want, got)

	// Collector lists inside each zone must also be sorted.
	assert.Equal(t, []string{"c-a-1", "c-a-2"}, snap[0].Collectors)
	// The uncovered zone has no collectors.
	assert.Empty(t, snap[3].Collectors)
	assert.False(t, snap[3].Covered)
}

func TestZoneTopology_RecordSpilloverDoesNotMutateState(t *testing.T) {
	// RecordSpillover only emits a metric; it must not change collector or
	// target indexes. This guards against future regressions where someone
	// adds bookkeeping to the spillover path and forgets the unit invariant.
	zt := newTestZoneTopology(t)
	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
	})
	zt.IncrementTargetCount("us-east-1a")

	before := zt.Snapshot()
	zt.RecordSpillover("us-east-1a", "us-east-1b")
	after := zt.Snapshot()
	assert.Equal(t, before, after)
}

func TestZoneTopology_ConcurrentAccessIsSafe(t *testing.T) {
	// Best-effort race-detection guard: hammer the topology from many
	// goroutines and rely on `go test -race` to flag data races. Even
	// without -race this exercise asserts that the final per-zone target
	// counts match what we incremented, which would silently break if any
	// of the mutation paths dropped updates.
	zt := newTestZoneTopology(t)
	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
		"c-b-1": NewCollector("c-b-1", "node-b-1", "us-east-1b"),
	})

	const writers = 8
	const writesPerWriter = 200
	var wg sync.WaitGroup
	wg.Add(writers + 1)

	// Concurrent target-count writers, half incrementing zone-a and half
	// incrementing zone-b, so the expected final counts are deterministic.
	for w := range writers {
		zone := "us-east-1a"
		if w%2 == 1 {
			zone = "us-east-1b"
		}
		go func(zone string) {
			defer wg.Done()
			for range writesPerWriter {
				zt.IncrementTargetCount(zone)
			}
		}(zone)
	}

	// Concurrent reader: keep calling Snapshot to surface any read/write
	// race on the underlying maps.
	go func() {
		defer wg.Done()
		for range writers * writesPerWriter {
			_ = zt.Snapshot()
		}
	}()

	wg.Wait()

	snap := snapshotByZone(zt.Snapshot())
	wantPerZone := (writers / 2) * writesPerWriter
	assert.Equal(t, wantPerZone, snap["us-east-1a"].TargetsDesired)
	assert.Equal(t, wantPerZone, snap["us-east-1b"].TargetsDesired)
}

func TestZoneTopology_TargetItemZoneExtractionRoundTrip(t *testing.T) {
	// End-to-end check that GetTargetZone -> IncrementTargetCount -> Snapshot
	// reflects the zone metadata Prometheus SD attached to the target.
	zt := newTestZoneTopology(t)
	zt.SetCollectors(map[string]*Collector{
		"c-a-1": NewCollector("c-a-1", "node-a-1", "us-east-1a"),
	})

	items := []*target.Item{
		newTargetWithZone("job-1", "10.0.0.1", "us-east-1a"),
		newTargetWithZone("job-1", "10.0.0.2", "us-east-1a"),
		newTargetWithZone("job-1", "10.0.0.3", "us-east-1b"),
		target.NewItem("job-1", "10.0.0.4", labels.New(labels.Label{Name: "instance", Value: "x"}), ""),
	}
	for _, it := range items {
		zt.IncrementTargetCount(zt.GetTargetZone(it))
	}

	snap := snapshotByZone(zt.Snapshot())
	assert.Equal(t, 2, snap["us-east-1a"].TargetsDesired)
	assert.Equal(t, 1, snap["us-east-1b"].TargetsDesired)
	assert.Equal(t, 1, snap[""].TargetsDesired)
	assert.Equal(t, []string{"us-east-1b"}, zt.UncoveredZones(),
		"zone-b has a desiring target but no collector and must be reported uncovered")
}

// snapshotByZone is a test helper that turns a Snapshot slice into a
// zone-keyed map for easier assertions.
func snapshotByZone(snap []ZoneSnapshot) map[string]ZoneSnapshot {
	out := make(map[string]ZoneSnapshot, len(snap))
	for _, z := range snap {
		out[z.Zone] = z
	}
	return out
}
