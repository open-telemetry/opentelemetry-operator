// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"
	"strings"

	"github.com/buraksezer/consistent"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const consistentHashingEndpointStrategyName = "consistent-hashing-endpoint"

// endpointKeySeparator delimits the components of the scrape-endpoint hash
// key. It matches the separator Prometheus uses for label hashing and won't
// appear in label names or values.
const endpointKeySeparator = '\xff'

var _ Strategy = &consistentHashingEndpointStrategy{}

// consistentHashingEndpointStrategy is a variant of consistent-hashing that
// keys the ring on the target's scrape endpoint - address, scheme, metrics
// path, and query params - instead of on __address__ alone.
//
// consistent-hashing hashes only __address__, so any targets that share a
// host:port but are distinguished by metrics path or query params (for
// example, multiple exporters behind the same proxy, or one host scraped
// under several paths) always land on the same collector. This strategy
// spreads those endpoints across collectors, while staying insensitive to
// labels that don't affect the scrape URL, so mutable metadata such as
// service-discovery annotations or instance labels doesn't reshuffle
// assignments.
type consistentHashingEndpointStrategy struct {
	config           consistent.Config
	consistentHasher *consistent.Consistent
}

func newConsistentHashingEndpointStrategy() Strategy {
	config := consistent.Config{
		PartitionCount:    1061,
		ReplicationFactor: 5,
		Load:              1.1,
		Hasher:            hasher{},
	}
	return &consistentHashingEndpointStrategy{
		consistentHasher: consistent.New(nil, config),
		config:           config,
	}
}

func (*consistentHashingEndpointStrategy) GetName() string {
	return consistentHashingEndpointStrategyName
}

func (s *consistentHashingEndpointStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	member := s.consistentHasher.LocateKey([]byte(endpointHashKey(item)))
	collectorName := member.String()
	collector, ok := collectors[collectorName]
	if !ok {
		return nil, fmt.Errorf("unknown collector %s", collectorName)
	}
	return collector, nil
}

func (s *consistentHashingEndpointStrategy) SetCollectors(collectors map[string]*Collector) {
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

func (*consistentHashingEndpointStrategy) SetFallbackStrategy(Strategy) {}

// endpointHashKey builds a stable key from the parts that make up a target's
// scrape endpoint: its address (item.TargetURL, the same value the
// consistent-hashing strategy hashes), scheme, metrics path, and query
// params. Labels are read in their (sorted) order, so the key is
// deterministic for a given endpoint.
func endpointHashKey(item *target.Item) string {
	ls := item.Labels
	var sb strings.Builder
	sb.WriteString(item.TargetURL)
	sb.WriteByte(endpointKeySeparator)
	sb.WriteString(ls.Get(model.SchemeLabel))
	sb.WriteByte(endpointKeySeparator)
	sb.WriteString(ls.Get(model.MetricsPathLabel))
	ls.Range(func(l labels.Label) {
		if strings.HasPrefix(l.Name, model.ParamLabelPrefix) {
			sb.WriteByte(endpointKeySeparator)
			sb.WriteString(l.Name)
			sb.WriteByte('=')
			sb.WriteString(l.Value)
		}
	})
	return sb.String()
}
