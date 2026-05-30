// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const leastWeightedStrategyName = "least-weighted"

var _ Strategy = &leastWeightedStrategy{}

// leastWeightedStrategy assigns each target to the collector with the
// fewest currently-assigned targets, with deterministic tiebreakers on
// per-job count and collector name. With zone-aware allocation enabled,
// the search is restricted to collectors in the target's desired zone
// (when at least one exists) and a global maxSkew check decides whether
// to spill cross-zone.
//
// Performance: SetCollectors caches the collector set partitioned by
// zone (collectorsByZone). GetCollectorForTarget then reads the cached
// per-zone view directly instead of rescanning every collector per
// target. At 100k targets across N collectors this turns each
// assignment from O(N) scan + map allocation into O(N_zone) scan with
// zero per-target allocation.
//
// lastCollectors retains the most recent collector set so SetZoneAwareness
// (called when an operator attaches a topology after collectors are
// already loaded) can rebuild collectorsByZone without waiting for the
// next SetCollectors call. Without this, late topology activation would
// leave the cache nil and quietly degrade in-zone selection to the
// failover path until the next collector reconcile.
type leastWeightedStrategy struct {
	zoneAwareState

	// collectorsByZone is rebuilt on every SetCollectors call and on
	// SetZoneAwareness toggles. It maps zone -> collector-name ->
	// collector pointer. Read access from GetCollectorForTarget runs
	// under the allocator's read lock so no per-strategy lock is
	// needed.
	collectorsByZone map[string]map[string]*Collector
	// lastCollectors mirrors what consistent-hashing keeps under the
	// same name: the most recently seen collector set, used to rebuild
	// the per-zone cache when SetZoneAwareness flips zone awareness on
	// or off without a fresh SetCollectors call.
	lastCollectors map[string]*Collector
}

func newleastWeightedStrategy() Strategy {
	return &leastWeightedStrategy{}
}

func (*leastWeightedStrategy) GetName() string {
	return leastWeightedStrategyName
}

func (s *leastWeightedStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	// Reassignment fast-path: keep an existing assignment when it is still
	// valid, but with zone awareness on we must guard against "stuck
	// failover" — when a target was previously placed cross-zone (because
	// its desired zone had no collectors) and a same-zone collector has
	// since arrived, returning the old assignment unchanged would silently
	// leave the target on the wrong zone forever. So we keep the fast-path
	// for the common cases (zone awareness off, target zone-less, target
	// already on a correct-zone collector, or the desired zone is still
	// uncovered) and fall through to the full evaluation otherwise.
	if item.CollectorName != "" {
		if col, ok := collectors[item.CollectorName]; ok {
			if !s.enabled() {
				return col, nil
			}
			desiredZone := s.targetZone(item)
			switch {
			case desiredZone == "":
				return col, nil
			case col.Zone == desiredZone:
				return col, nil
			case len(s.collectorsByZone[desiredZone]) == 0:
				// Desired zone still uncovered — current cross-zone
				// assignment is the only option.
				return col, nil
			}
			// Otherwise the target is cross-zone but its zone now has
			// collectors. Fall through to re-evaluate.
		}
	}

	if !s.enabled() {
		return pickLeastLoaded(collectors, item.JobName), nil
	}

	desiredZone := s.targetZone(item)
	// Zone-agnostic targets pick from the global pool. They contribute to
	// the global skew like everything else, but they're not bound to any
	// particular zone.
	if desiredZone == "" {
		return pickLeastLoaded(collectors, item.JobName), nil
	}

	zoneSubset := s.collectorsByZone[desiredZone]
	if len(zoneSubset) == 0 {
		// Failover: zone has no collectors. Pick globally and tell the
		// topology so operators can see the gap.
		chosen := pickLeastLoaded(collectors, item.JobName)
		if s.zt != nil && chosen != nil {
			s.zt.RecordSpillover(desiredZone, chosen.Zone)
		}
		return chosen, nil
	}

	candidate := pickLeastLoaded(zoneSubset, item.JobName)
	if s.maxSkew > 0 && exceedsMaxSkew(candidate, collectors, s.maxSkew) {
		// Spillover: keeping this target in its zone would push the
		// global load skew over maxSkew. Pick the globally least-loaded
		// collector instead.
		globalPick := pickLeastLoaded(collectors, item.JobName)
		if s.zt != nil && globalPick != nil {
			s.zt.RecordSpillover(desiredZone, globalPick.Zone)
		}
		return globalPick, nil
	}
	return candidate, nil
}

// pickLeastLoaded returns the collector in the input set with the fewest
// targets, with deterministic tiebreakers: prefer fewer targets from the
// same job, then lexicographically smaller collector name. Returns nil
// when the input is empty.
func pickLeastLoaded(collectors map[string]*Collector, jobName string) *Collector {
	var col *Collector
	for _, v := range collectors {
		if col == nil || v.NumTargets < col.NumTargets {
			col = v
		} else if v.NumTargets == col.NumTargets {
			vPerJob := v.TargetsPerJob[jobName]
			colPerJob := col.TargetsPerJob[jobName]
			if vPerJob < colPerJob || (vPerJob == colPerJob && v.Name < col.Name) {
				col = v
			}
		}
	}
	return col
}

// SetCollectors rebuilds the per-zone collector index so
// GetCollectorForTarget does not have to rescan and allocate on every
// target assignment. The cache is only built when zone-aware allocation
// is active — pre-feature deployments pay zero cost here, preserving
// the existing no-op contract for the legacy code path. The collector
// set is also stashed in lastCollectors so SetZoneAwareness can rebuild
// the cache on a later toggle without requiring a fresh
// SetCollectors call.
func (s *leastWeightedStrategy) SetCollectors(collectors map[string]*Collector) {
	s.lastCollectors = collectors
	s.rebuildCacheFromLastCollectors()
}

// rebuildCacheFromLastCollectors recomputes collectorsByZone from
// lastCollectors. It is the single source of "how do we build the per-
// zone view" and is shared between SetCollectors (normal path) and
// SetZoneAwareness (late toggle path).
func (s *leastWeightedStrategy) rebuildCacheFromLastCollectors() {
	if !s.enabled() || len(s.lastCollectors) == 0 {
		s.collectorsByZone = nil
		return
	}
	byZone := make(map[string]map[string]*Collector)
	for name, c := range s.lastCollectors {
		if c.Zone == "" {
			continue
		}
		if byZone[c.Zone] == nil {
			byZone[c.Zone] = make(map[string]*Collector)
		}
		byZone[c.Zone][name] = c
	}
	s.collectorsByZone = byZone
}

func (*leastWeightedStrategy) SetFallbackStrategy(Strategy) {}

// SetZoneAwareness toggles zone-aware allocation. When zone awareness
// is being enabled after collectors already exist (a late-attach
// scenario the allocator API explicitly supports via SetZoneTopology),
// the per-zone cache must be rebuilt from the previously seen
// collector set — otherwise the next GetCollectorForTarget call would
// see an empty cache and incorrectly route same-zone targets through
// the failover path until the next collector reconcile. Disabling
// zone awareness clears the cache symmetrically.
func (s *leastWeightedStrategy) SetZoneAwareness(zt *ZoneTopology, maxSkew int) {
	s.setZoneAwareness(zt, maxSkew)
	s.rebuildCacheFromLastCollectors()
}
