// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"slices"
	"sync"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// ZoneTopology maintains a per-zone view of the allocator state for
// observability and zone-aware allocation queries.
//
// It tracks which collectors live in which zone, how many targets are
// currently assigned per desired zone, detects uncovered zones (zones with
// targets but no collectors), and exposes metrics. ZoneTopology is the single
// source of truth for "where are my collectors and targets, per zone" that
// both metrics and the future /zones API endpoint will consume.
//
// Thread-safety: all public methods are safe for concurrent use. Callers
// (the allocator and strategies) are expected to call the mutating methods
// (SetCollectors, IncrementTargetCount, etc.) while holding the allocator's
// write lock so that the topology stays consistent with the allocator's
// internal state.
type ZoneTopology struct {
	mu sync.RWMutex

	// targetZoneLabel is the Prometheus SD meta-label used to extract a
	// target's desired zone (e.g. "__meta_kubernetes_endpointslice_endpoint_zone").
	// Empty string disables target zone extraction.
	targetZoneLabel string

	// collectorsByZone maps zone name -> set of collector names in that zone.
	// The empty-string zone ("") aggregates collectors whose zone could not
	// be resolved (e.g. the node had no zone label).
	collectorsByZone map[string]map[string]struct{}

	// targetsByZone maps a target's *desired* zone -> number of currently
	// tracked targets that want that zone. Targets without a zone label
	// contribute to the "" bucket.
	targetsByZone map[string]int

	log logr.Logger

	collectorsPerZone   metric.Int64Gauge
	targetsPerZone      metric.Int64Gauge
	uncoveredZonesGauge metric.Int64Gauge
	spilloverCounter    metric.Int64Counter
}

// NewZoneTopology constructs a ZoneTopology. The targetZoneLabel argument is
// the Prometheus SD meta-label name used to read a target's zone (typically
// "__meta_kubernetes_endpointslice_endpoint_zone"). Pass "" to disable target
// zone extraction; in that mode the topology still tracks collectors per
// zone but every target is treated as zone-less.
func NewZoneTopology(log logr.Logger, targetZoneLabel string) (*ZoneTopology, error) {
	meter := otel.GetMeterProvider().Meter("targetallocator")
	collectorsPerZone, err := meter.Int64Gauge(
		"opentelemetry_allocator_collectors_per_zone",
		metric.WithDescription("Number of collectors discovered in each topology zone."),
	)
	if err != nil {
		return nil, err
	}
	targetsPerZone, err := meter.Int64Gauge(
		"opentelemetry_allocator_targets_per_zone",
		metric.WithDescription("Number of targets that desire each topology zone."),
	)
	if err != nil {
		return nil, err
	}
	uncoveredZonesGauge, err := meter.Int64Gauge(
		"opentelemetry_allocator_uncovered_zones",
		metric.WithDescription("Number of zones that have targets but no collectors."),
	)
	if err != nil {
		return nil, err
	}
	spilloverCounter, err := meter.Int64Counter(
		"opentelemetry_allocator_zone_spillover",
		metric.WithDescription("Number of cross-zone target assignments caused by missing same-zone collectors or maxSkew enforcement."),
	)
	if err != nil {
		return nil, err
	}

	return &ZoneTopology{
		targetZoneLabel:     targetZoneLabel,
		collectorsByZone:    make(map[string]map[string]struct{}),
		targetsByZone:       make(map[string]int),
		log:                 log.WithValues("component", "zone-topology"),
		collectorsPerZone:   collectorsPerZone,
		targetsPerZone:      targetsPerZone,
		uncoveredZonesGauge: uncoveredZonesGauge,
		spilloverCounter:    spilloverCounter,
	}, nil
}

// GetTargetZone returns the desired zone for the given target by reading the
// configured targetZoneLabel from the target's labels. Returns "" if the
// target has no zone metadata or if target zone extraction is disabled.
func (zt *ZoneTopology) GetTargetZone(item *target.Item) string {
	if item == nil || zt.targetZoneLabel == "" {
		return ""
	}
	return item.Labels.Get(zt.targetZoneLabel)
}

// SetCollectors rebuilds the per-zone collector index from scratch using the
// given collector set. It then re-publishes the collectors_per_zone metric
// and recomputes the uncovered-zones gauge.
func (zt *ZoneTopology) SetCollectors(collectors map[string]*Collector) {
	zt.mu.Lock()
	defer zt.mu.Unlock()

	// Capture the set of zones we previously published so we can zero out
	// gauges for zones that disappear entirely.
	previousZones := make(map[string]struct{}, len(zt.collectorsByZone))
	for z := range zt.collectorsByZone {
		previousZones[z] = struct{}{}
	}

	zt.collectorsByZone = make(map[string]map[string]struct{}, len(collectors))
	for _, c := range collectors {
		if zt.collectorsByZone[c.Zone] == nil {
			zt.collectorsByZone[c.Zone] = make(map[string]struct{})
		}
		zt.collectorsByZone[c.Zone][c.Name] = struct{}{}
	}

	zt.recordCollectorsPerZoneLocked(previousZones)
	zt.recordUncoveredZonesLocked()
}

// IncrementTargetCount records that one additional target now desires the
// given zone. Pass "" for targets that have no zone metadata. The
// targets_per_zone gauge is re-published.
func (zt *ZoneTopology) IncrementTargetCount(zone string) {
	zt.mu.Lock()
	defer zt.mu.Unlock()
	zt.targetsByZone[zone]++
	zt.recordTargetsForZoneLocked(zone)
	zt.recordUncoveredZonesLocked()
}

// DecrementTargetCount records that one target no longer desires the given
// zone (typically because the target was removed or reassigned). Pass "" for
// targets that have no zone metadata. The targets_per_zone gauge is
// re-published.
func (zt *ZoneTopology) DecrementTargetCount(zone string) {
	zt.mu.Lock()
	defer zt.mu.Unlock()
	if zt.targetsByZone[zone] <= 1 {
		delete(zt.targetsByZone, zone)
	} else {
		zt.targetsByZone[zone]--
	}
	zt.recordTargetsForZoneLocked(zone)
	zt.recordUncoveredZonesLocked()
}

// RecordSpillover increments the spillover counter, attributing the event to
// the originating (desired) zone and the destination (assigned) zone.
// fromZone and toZone may be "" when zone information is missing.
func (zt *ZoneTopology) RecordSpillover(fromZone, toZone string) {
	zt.spilloverCounter.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("from_zone", fromZone),
		attribute.String("to_zone", toZone),
	))
}

// CollectorsInZone returns a sorted list of collector names in the given
// zone, or nil if no collectors live in that zone.
func (zt *ZoneTopology) CollectorsInZone(zone string) []string {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	members, ok := zt.collectorsByZone[zone]
	if !ok || len(members) == 0 {
		return nil
	}
	out := make([]string, 0, len(members))
	for name := range members {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// Zones returns the sorted list of zones that have at least one collector.
// The empty-string zone (zone-less collectors) is omitted; use ZonelessCollectorCount
// to query that bucket.
func (zt *ZoneTopology) Zones() []string {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	out := make([]string, 0, len(zt.collectorsByZone))
	for z := range zt.collectorsByZone {
		if z == "" {
			continue
		}
		out = append(out, z)
	}
	slices.Sort(out)
	return out
}

// UncoveredZones returns zones that have at least one target wanting them but
// no collectors available to scrape them. Targets in these zones will be
// allocated cross-zone as failover. The empty-string zone is never reported
// as uncovered (zone-less targets are not "uncovered", just zone-agnostic).
func (zt *ZoneTopology) UncoveredZones() []string {
	zt.mu.RLock()
	defer zt.mu.RUnlock()
	return zt.uncoveredZonesLocked()
}

// ZoneSnapshot captures the per-zone state at a single point in time and is
// suitable for serialization to the /zones API endpoint or UI.
type ZoneSnapshot struct {
	// Zone is the topology zone name. Empty string represents the
	// zone-less bucket (collectors without a node zone label or targets
	// without a zone label).
	Zone string `json:"zone"`
	// Collectors is the sorted list of collector names in this zone.
	Collectors []string `json:"collectors"`
	// TargetsDesired is the number of targets that desire this zone, even
	// if they ended up assigned to a collector in another zone via failover
	// or maxSkew spillover.
	TargetsDesired int `json:"targetsDesired"`
	// Covered is true when at least one collector exists in this zone, or
	// when the zone is the zone-less bucket. False indicates the targets
	// in this zone will be served cross-zone (failover).
	Covered bool `json:"covered"`
}

// Snapshot returns a stable, sorted, deeply-copied view of the per-zone state
// suitable for serialization. The slice is sorted by zone name.
func (zt *ZoneTopology) Snapshot() []ZoneSnapshot {
	zt.mu.RLock()
	defer zt.mu.RUnlock()

	// Collect the union of zones seen in either collectors or targets so the
	// snapshot surfaces uncovered zones too.
	seen := make(map[string]struct{}, len(zt.collectorsByZone)+len(zt.targetsByZone))
	for z := range zt.collectorsByZone {
		seen[z] = struct{}{}
	}
	for z := range zt.targetsByZone {
		seen[z] = struct{}{}
	}

	out := make([]ZoneSnapshot, 0, len(seen))
	for z := range seen {
		members := zt.collectorsByZone[z]
		collectorNames := make([]string, 0, len(members))
		for name := range members {
			collectorNames = append(collectorNames, name)
		}
		slices.Sort(collectorNames)

		// The "" zone is treated as covered (zone-agnostic), all named zones
		// are covered only if they have at least one collector.
		covered := z == "" || len(members) > 0

		out = append(out, ZoneSnapshot{
			Zone:           z,
			Collectors:     collectorNames,
			TargetsDesired: zt.targetsByZone[z],
			Covered:        covered,
		})
	}
	slices.SortFunc(out, func(a, b ZoneSnapshot) int {
		if a.Zone < b.Zone {
			return -1
		}
		if a.Zone > b.Zone {
			return 1
		}
		return 0
	})
	return out
}

// recordCollectorsPerZoneLocked publishes the collectors_per_zone gauge for
// every currently-known zone and zeroes out any zones that have disappeared
// since the previous publication. The caller must hold zt.mu.
func (zt *ZoneTopology) recordCollectorsPerZoneLocked(previousZones map[string]struct{}) {
	ctx := context.Background()
	for zone, members := range zt.collectorsByZone {
		zt.collectorsPerZone.Record(ctx, int64(len(members)),
			metric.WithAttributes(attribute.String("zone", zone)))
		delete(previousZones, zone)
	}
	// Zero out zones that no longer have any collectors.
	for zone := range previousZones {
		zt.collectorsPerZone.Record(ctx, 0,
			metric.WithAttributes(attribute.String("zone", zone)))
	}
}

// recordTargetsForZoneLocked publishes the targets_per_zone gauge for a
// single zone. The caller must hold zt.mu.
func (zt *ZoneTopology) recordTargetsForZoneLocked(zone string) {
	zt.targetsPerZone.Record(context.Background(), int64(zt.targetsByZone[zone]),
		metric.WithAttributes(attribute.String("zone", zone)))
}

// recordUncoveredZonesLocked publishes the uncovered_zones gauge and emits a
// log warning per uncovered zone (V(1) verbosity to avoid noise). The caller
// must hold zt.mu.
func (zt *ZoneTopology) recordUncoveredZonesLocked() {
	uncovered := zt.uncoveredZonesLocked()
	zt.uncoveredZonesGauge.Record(context.Background(), int64(len(uncovered)))
	if len(uncovered) > 0 {
		zt.log.V(1).Info("Detected zones with targets but no collectors; targets in these zones will be served cross-zone",
			"uncoveredZones", uncovered)
	}
}

// uncoveredZonesLocked computes the set of zones that have targets but no
// collectors. The empty-string bucket is excluded (zone-less targets are
// served by the global pool, not "uncovered"). The caller must hold zt.mu.
func (zt *ZoneTopology) uncoveredZonesLocked() []string {
	var out []string
	for zone, count := range zt.targetsByZone {
		if zone == "" || count == 0 {
			continue
		}
		if members, ok := zt.collectorsByZone[zone]; !ok || len(members) == 0 {
			out = append(out, zone)
		}
	}
	slices.Sort(out)
	return out
}
