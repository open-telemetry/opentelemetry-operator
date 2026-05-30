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
type leastWeightedStrategy struct {
	zoneAwareState
}

func newleastWeightedStrategy() Strategy {
	return &leastWeightedStrategy{}
}

func (*leastWeightedStrategy) GetName() string {
	return leastWeightedStrategyName
}

func (s *leastWeightedStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	// Reassignment fast-path: if the target already names a collector and
	// that collector still exists, keep the assignment. This matches the
	// pre-zone-aware behavior and avoids needlessly thrashing assignments
	// on cluster reconciles.
	if item.CollectorName != "" {
		if col, ok := collectors[item.CollectorName]; ok {
			return col, nil
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

	zoneSubset := subsetForZone(collectors, desiredZone)
	if zoneSubset == nil {
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

func (*leastWeightedStrategy) SetCollectors(map[string]*Collector) {}

func (*leastWeightedStrategy) SetFallbackStrategy(Strategy) {}

func (s *leastWeightedStrategy) SetZoneAwareness(zt *ZoneTopology, maxSkew int) {
	s.setZoneAwareness(zt, maxSkew)
}
