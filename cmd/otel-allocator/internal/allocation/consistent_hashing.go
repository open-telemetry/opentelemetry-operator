// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"errors"
	"fmt"

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const consistentHashingStrategyName = "consistent-hashing"

type hasher struct{}

func (hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

var _ Strategy = &consistentHashingStrategy{}

// consistentHashingStrategy distributes targets across collectors via a
// consistent hash ring keyed on target URL. With zone-aware allocation
// enabled, a separate ring is maintained per zone so that hashing is
// constrained to the desired zone's collectors. A global ring across all
// collectors is also kept for two purposes:
//   - failover, when the target's desired zone has no collectors
//   - spillover, when the maxSkew check rejects a zone-local assignment
type consistentHashingStrategy struct {
	zoneAwareState

	config consistent.Config
	// globalHasher hashes across the full collector set. Used for
	// non-zone-aware allocation and for failover/spillover paths.
	globalHasher *consistent.Consistent
	// zoneHashers maps zone name -> ring containing only that zone's
	// collectors. Populated lazily on SetCollectors; only present when
	// zone awareness is enabled.
	zoneHashers map[string]*consistent.Consistent
	// lastCollectors retains the last collector set so we can rebuild
	// per-zone rings without re-receiving the map from the allocator.
	// This is needed because SetZoneAwareness can flip the strategy
	// between zone-aware and global modes after collectors have already
	// been loaded.
	lastCollectors map[string]*Collector
}

func newConsistentHashingStrategy() Strategy {
	config := consistent.Config{
		PartitionCount:    1061,
		ReplicationFactor: 5,
		Load:              1.1,
		Hasher:            hasher{},
	}
	return &consistentHashingStrategy{
		config:       config,
		globalHasher: consistent.New(nil, config),
	}
}

func (*consistentHashingStrategy) GetName() string {
	return consistentHashingStrategyName
}

func (s *consistentHashingStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	hashKey := []byte(item.TargetURL)

	// Fast path: zone awareness disabled, fall back to the original
	// single-ring behavior so unconfigured deployments see no change.
	if !s.enabled() {
		return s.lookupOnRing(s.globalHasher, hashKey, collectors)
	}

	desiredZone := s.targetZone(item)
	// Targets without zone metadata are zone-agnostic — they live on the
	// global ring just like in the non-zone-aware mode.
	if desiredZone == "" {
		return s.lookupOnRing(s.globalHasher, hashKey, collectors)
	}

	zoneRing, hasZoneRing := s.zoneHashers[desiredZone]
	if !hasZoneRing {
		// Failover: target wants a zone we have no collectors in. Fall
		// back to the global ring and surface the event so operators
		// can scale their zone coverage.
		chosen, err := s.lookupOnRing(s.globalHasher, hashKey, collectors)
		if err == nil && s.zt != nil {
			s.zt.RecordSpillover(desiredZone, chosen.Zone)
		}
		return chosen, err
	}

	candidate, err := s.lookupOnRing(zoneRing, hashKey, collectors)
	if err != nil {
		return nil, err
	}

	// maxSkew=0 means "never spill"; pure zone affinity.
	if s.maxSkew > 0 && exceedsMaxSkew(candidate, collectors, s.maxSkew) {
		// For spillover we intentionally drop the hash-based selection in
		// favor of the globally least-loaded collector. Reason: a hash on
		// the global ring can re-route the target right back to the
		// overloaded same-zone collector (since that collector still owns
		// some partitions on the global ring), which defeats the entire
		// point of opting in to maxSkew. Picking least-loaded gives the
		// load-balance guarantee the operator asked for. The cost is that
		// spillover assignments are not consistent across collector set
		// changes — but spillover already implies the cluster is under
		// stress, and reassignment churn is the lesser evil.
		globalPick := pickLeastLoaded(collectors, item.JobName)
		if globalPick == nil {
			return candidate, nil
		}
		if s.zt != nil {
			s.zt.RecordSpillover(desiredZone, globalPick.Zone)
		}
		return globalPick, nil
	}

	return candidate, nil
}

// lookupOnRing finds the collector that owns hashKey on the given ring and
// resolves the ring's member name to a live *Collector from the input map.
// Returns an error if the resolved name is no longer in the collector map
// — this normally only happens if SetCollectors and GetCollectorForTarget
// run with inconsistent state, but we surface it explicitly rather than
// crashing.
func (*consistentHashingStrategy) lookupOnRing(ring *consistent.Consistent, hashKey []byte, collectors map[string]*Collector) (*Collector, error) {
	if ring == nil {
		return nil, errors.New("consistent hash ring is not initialized")
	}
	member := ring.LocateKey(hashKey)
	if member == nil {
		return nil, errors.New("no collector available on consistent hash ring")
	}
	name := member.String()
	collector, ok := collectors[name]
	if !ok {
		return nil, fmt.Errorf("unknown collector %s", name)
	}
	return collector, nil
}

func (s *consistentHashingStrategy) SetCollectors(collectors map[string]*Collector) {
	// Cache the collector set so SetZoneAwareness can rebuild the per-zone
	// rings on its own, without the allocator having to re-invoke
	// SetCollectors after a zone-awareness flip.
	s.lastCollectors = collectors
	s.rebuildRings()
}

// SetZoneAwareness toggles zone-aware allocation. Flipping the flag forces
// a rebuild of the per-zone rings (when enabling) or discards them (when
// disabling) using the most recently seen collector set.
func (s *consistentHashingStrategy) SetZoneAwareness(zt *ZoneTopology, maxSkew int) {
	s.setZoneAwareness(zt, maxSkew)
	s.rebuildRings()
}

func (s *consistentHashingStrategy) rebuildRings() {
	// Global ring always reflects the full collector set.
	var members []consistent.Member
	if len(s.lastCollectors) > 0 {
		members = make([]consistent.Member, 0, len(s.lastCollectors))
		for _, c := range s.lastCollectors {
			members = append(members, c)
		}
	}
	s.globalHasher = consistent.New(members, s.config)

	// Per-zone rings only exist when zone-awareness is enabled. We drop
	// them on disable to free up memory and to keep state simple.
	if !s.enabled() {
		s.zoneHashers = nil
		return
	}
	byZone := make(map[string][]consistent.Member)
	for _, c := range s.lastCollectors {
		if c.Zone == "" {
			continue
		}
		byZone[c.Zone] = append(byZone[c.Zone], c)
	}
	s.zoneHashers = make(map[string]*consistent.Consistent, len(byZone))
	for zone, zoneMembers := range byZone {
		s.zoneHashers[zone] = consistent.New(zoneMembers, s.config)
	}
}

func (*consistentHashingStrategy) SetFallbackStrategy(Strategy) {}
