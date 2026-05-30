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

// nodeZoneLookup resolves a Kubernetes node name to its topology zone.
// ZoneTopology accepts an interface (not the concrete *NodeZoneResolver)
// so tests can stub the lookup and so future callers can plug in
// alternative resolution paths (cloud SDK, static map, etc.) without
// taking a hard dependency on the K8s client wiring.
type nodeZoneLookup interface {
	GetZone(nodeName string) string
}

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
	// Empty string disables direct label extraction; the node fallback
	// (see nodeResolver) still applies.
	targetZoneLabel string
	// nodeResolver, when non-nil, is consulted whenever the target zone
	// label is missing or empty. The target's node name (via
	// target.Item.GetNodeName) is looked up in the resolver to find the
	// zone. This covers Pod SD, Endpoints SD, static configs, and other
	// non-EndpointSlice paths that emit a node label but not a zone label.
	nodeResolver nodeZoneLookup

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

	// distinctZoneHighWatermark is the largest number of distinct zones
	// the topology has observed across collectors and target labels in
	// its lifetime. We use it as a tripwire for accidental
	// high-cardinality misconfiguration (e.g. operator points
	// target_zone_label at an instance-id label by mistake): the field is
	// monotonic, so a single warning log is emitted the first time the
	// count crosses the configured threshold.
	distinctZoneHighWatermark int
	cardinalityWarningEmitted bool
}

// cardinalityWarnThreshold is the number of distinct zones at which the
// topology emits a one-time WARN log pointing operators at the
// target_zone_label config. Real cloud topologies have a handful of
// zones per region (AWS up to 6, GCP/Azure typically 3); 64 is well
// above that ceiling and well below any plausible high-cardinality
// label (instance IDs, pod IPs, etc.) which would explode into
// thousands. The threshold is a constant — operators can't tune it
// because the only correct response to crossing it is reading the warning.
const cardinalityWarnThreshold = 64

// NewZoneTopology constructs a ZoneTopology. The targetZoneLabel argument is
// the Prometheus SD meta-label name used to read a target's zone (typically
// "__meta_kubernetes_endpointslice_endpoint_zone"). Pass "" to disable target
// zone extraction; in that mode the topology still tracks collectors per
// zone but every target is treated as zone-less unless a node-zone resolver
// is attached.
//
// Use WithNodeZoneResolver to plug in the node-zone fallback path so the
// topology can resolve a zone from the target's node name when the SD
// pipeline does not emit a zone meta-label (Pod SD, Endpoints SD without
// EndpointSlice, etc.).
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

// WithNodeZoneResolver attaches a node-name -> zone resolver so the topology
// can fall back to the target's node zone when the SD-supplied zone label
// is missing. nil clears the resolver (label-only mode).
func (zt *ZoneTopology) WithNodeZoneResolver(r nodeZoneLookup) *ZoneTopology {
	zt.mu.Lock()
	defer zt.mu.Unlock()
	zt.nodeResolver = r
	return zt
}

// GetTargetZone returns the desired zone for the given target. The lookup
// is two-stage:
//  1. Read the configured targetZoneLabel from the target's labels. This
//     is the fast path used by Kubernetes EndpointSlice SD, EC2 SD, GCE
//     SD, etc., where the SD pipeline already populates a zone meta-label.
//  2. If the label is empty (or extraction is disabled) and a node-zone
//     resolver is attached, look up the target's node name via
//     target.Item.GetNodeName() and resolve that node's zone. This covers
//     Pod SD, classic Endpoints SD, and static configs that carry a node
//     label but no zone label, so zone awareness keeps working on those
//     paths instead of silently falling back to global allocation.
//
// Returns "" if neither stage produces a zone.
func (zt *ZoneTopology) GetTargetZone(item *target.Item) string {
	if item == nil {
		return ""
	}
	if zt.targetZoneLabel != "" {
		if z := item.Labels.Get(zt.targetZoneLabel); z != "" {
			return z
		}
	}
	zt.mu.RLock()
	resolver := zt.nodeResolver
	zt.mu.RUnlock()
	if resolver == nil {
		return ""
	}
	nodeName := item.GetNodeName()
	if nodeName == "" {
		return ""
	}
	return resolver.GetZone(nodeName)
}

// Reset wipes the per-zone target counts and clears any per-zone gauges the
// previous state had emitted. It intentionally leaves the collector index
// alone — callers driving Reset (notably the allocator's SetZoneTopology
// hydration path) follow it with SetCollectors anyway. Reset exists so a
// caller can safely re-hydrate target counts without double-counting.
func (zt *ZoneTopology) Reset() {
	zt.mu.Lock()
	defer zt.mu.Unlock()
	// Zero out the gauge for every zone we had targets in so existing
	// scrapers see the reset rather than stale values.
	for zone := range zt.targetsByZone {
		zt.targetsPerZone.Record(context.Background(), 0,
			metric.WithAttributes(attribute.String("zone", zone)))
	}
	zt.targetsByZone = make(map[string]int)
	zt.uncoveredZonesGauge.Record(context.Background(), 0)
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
	zt.maybeWarnHighCardinalityLocked()
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

// maybeWarnHighCardinalityLocked emits a one-time log warning the first
// time the count of distinct zones (collectors + targets) crosses
// cardinalityWarnThreshold. The intent is to catch
// `target_zone_label` misconfiguration that points at a high-cardinality
// label (pod IP, instance ID) before it explodes the targets_by_zone map
// and the Prometheus series count. The check runs under zt.mu.
func (zt *ZoneTopology) maybeWarnHighCardinalityLocked() {
	if zt.cardinalityWarningEmitted {
		return
	}
	distinct := len(zt.targetsByZone)
	if distinct > zt.distinctZoneHighWatermark {
		zt.distinctZoneHighWatermark = distinct
	}
	if zt.distinctZoneHighWatermark < cardinalityWarnThreshold {
		return
	}
	zt.cardinalityWarningEmitted = true
	zt.log.Info(
		"observed unusually high zone cardinality — verify topology.target_zone_label points at a low-cardinality SD label (zone names), not an instance or pod identifier",
		"distinctZones", zt.distinctZoneHighWatermark,
		"threshold", cardinalityWarnThreshold,
	)
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
