// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const perNodeStrategyName = "per-node"

var _ Strategy = &perNodeStrategy{}

type perNodeStrategy struct {
	collectorByNode  map[string]*Collector
	fallbackStrategy Strategy
}

func newPerNodeStrategy() Strategy {
	return &perNodeStrategy{
		collectorByNode:  make(map[string]*Collector),
		fallbackStrategy: nil,
	}
}

func (s *perNodeStrategy) SetFallbackStrategy(fallbackStrategy Strategy) {
	s.fallbackStrategy = fallbackStrategy
}

func (s *perNodeStrategy) GetName() string {
	return perNodeStrategyName
}

func (s *perNodeStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	targetNodeName := item.GetNodeName()
	if targetNodeName == "" && s.fallbackStrategy != nil {
		return s.fallbackStrategy.GetCollectorForTarget(collectors, item)
	}

	collector, ok := s.collectorByNode[targetNodeName]
	if !ok {
		return nil, fmt.Errorf("could not find collector for node %s", targetNodeName)
	}
	return collectors[collector.Name], nil
}

func (s *perNodeStrategy) SetCollectors(collectors map[string]*Collector) {
	clear(s.collectorByNode)
	for _, collector := range collectors {
		if collector.NodeName != "" {
			s.collectorByNode[collector.NodeName] = collector
		}
	}

	if s.fallbackStrategy != nil {
		s.fallbackStrategy.SetCollectors(collectors)
	}
}
