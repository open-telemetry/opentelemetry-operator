// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"

	"github.com/buraksezer/consistent"
	"github.com/cespare/xxhash/v2"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const consistentHashingStrategyName = "consistent-hashing"

type hasher struct{}

func (h hasher) Sum64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

var _ Strategy = &consistentHashingStrategy{}

type consistentHashingStrategy struct {
	config           consistent.Config
	consistentHasher *consistent.Consistent
}

func newConsistentHashingStrategy() Strategy {
	config := consistent.Config{
		PartitionCount:    1061,
		ReplicationFactor: 5,
		Load:              1.1,
		Hasher:            hasher{},
	}
	consistentHasher := consistent.New(nil, config)
	chStrategy := &consistentHashingStrategy{
		consistentHasher: consistentHasher,
		config:           config,
	}
	return chStrategy
}

func (s *consistentHashingStrategy) GetName() string {
	return consistentHashingStrategyName
}

func (s *consistentHashingStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	hashKey := item.TargetURL
	member := s.consistentHasher.LocateKey([]byte(hashKey))
	collectorName := member.String()
	collector, ok := collectors[collectorName]
	if !ok {
		return nil, fmt.Errorf("unknown collector %s", collectorName)
	}
	return collector, nil
}

func (s *consistentHashingStrategy) SetCollectors(collectors map[string]*Collector) {
	// we simply recreate the hasher with the new member set
	// this isn't any more expensive than doing a diff and then applying the change
	var members []consistent.Member

	if len(collectors) > 0 {
		members = make([]consistent.Member, 0, len(collectors))
		for _, collector := range collectors {
			members = append(members, collector)
		}
	}

	s.consistentHasher = consistent.New(members, s.config)

}

func (s *consistentHashingStrategy) SetFallbackStrategy(fallbackStrategy Strategy) {}
