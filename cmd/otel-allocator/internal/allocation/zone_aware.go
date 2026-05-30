// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

// zoneAwareState holds the per-strategy zone-aware configuration. It is
// embedded into individual strategies (consistent-hashing, least-weighted)
// so they share a single source of truth for "are we zone-aware right now?"
// without duplicating the SetZoneAwareness boilerplate. Strategies use the
// helpers below (subsetForZone, exceedsMaxSkew) to keep their assignment
// code uniform across implementations.
//
// Concurrency: zoneAwareState is mutated only via SetZoneAwareness and
// SetCollectors, both of which are called by the allocator while holding
// its write lock. Read access from GetCollectorForTarget runs under the
// allocator's read lock. There is therefore no internal synchronization.
type zoneAwareState struct {
	// zt is non-nil exactly when zone-aware allocation is enabled. The
	// strategy queries zt for GetTargetZone, CollectorsInZone, and
	// RecordSpillover.
	zt *ZoneTopology
	// maxSkew is the cross-zone spillover threshold. 0 disables the check
	// entirely (pure zone affinity). Negative values are not possible
	// because the config layer rejects them.
	maxSkew int
}

// setZoneAwareness updates the zone-aware config. Passing zt=nil disables
// zone awareness.
func (z *zoneAwareState) setZoneAwareness(zt *ZoneTopology, maxSkew int) {
	z.zt = zt
	z.maxSkew = maxSkew
}

// enabled reports whether zone-aware allocation is currently active.
func (z *zoneAwareState) enabled() bool {
	return z.zt != nil
}

// targetZone returns the target's desired zone, or "" if zone awareness is
// disabled, the target has no zone label, or the target is nil.
func (z *zoneAwareState) targetZone(item *target.Item) string {
	if z.zt == nil {
		return ""
	}
	return z.zt.GetTargetZone(item)
}

// exceedsMaxSkew reports whether assigning one additional target to
// `candidate` would push the global load skew over `maxSkew`. Skew is
// defined as max(NumTargets) - min(NumTargets) across all collectors. A
// maxSkew of 0 disables the check (returns false unconditionally).
//
// The cost is O(N) over collectors. We only run it on the assignment hot
// path when maxSkew > 0, so the typical zone-affinity-only configuration
// pays nothing.
func exceedsMaxSkew(candidate *Collector, allCollectors map[string]*Collector, maxSkew int) bool {
	if maxSkew <= 0 || candidate == nil {
		return false
	}
	// We only need the global minimum, not the max. After assignment,
	// candidate.NumTargets + 1 is the new max contribution from this
	// collector; comparing it against the current global min gives the
	// post-assignment skew lower bound. If that lower bound already
	// exceeds maxSkew, spill.
	globalMin := candidate.NumTargets // candidate counts toward the min calculation
	for _, c := range allCollectors {
		if c.NumTargets < globalMin {
			globalMin = c.NumTargets
		}
	}
	return (candidate.NumTargets + 1 - globalMin) > maxSkew
}
