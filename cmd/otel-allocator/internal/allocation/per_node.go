// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const perNodeStrategyName = "per-node"

var _ Strategy = &perNodeStrategy{}

type perNodeStrategy struct {
	collectorByNode  map[string]*Collector
	fallbackStrategy Strategy
}

// newPerNodeStrategy constructs a per-node strategy. The fallbackStrategy, which may be nil, is used to
// assign targets the per-node strategy can't assign on its own.
func newPerNodeStrategy(fallbackStrategy Strategy) Strategy {
	return &perNodeStrategy{
		collectorByNode:  make(map[string]*Collector),
		fallbackStrategy: fallbackStrategy,
	}
}

// buildPerNodeStrategy builds the configured fallback strategy and injects it into the per-node strategy.
func buildPerNodeStrategy(config StrategyConfig) (Strategy, error) {
	var fallbackStrategy Strategy
	if fallbackConfig := config.PerNode.FallbackStrategy; fallbackConfig != nil {
		// A per-node fallback would fail on exactly the targets the primary strategy failed on.
		if fallbackConfig.Name == perNodeStrategyName {
			return nil, errors.New("the per-node strategy can't be used as its own fallback")
		}
		fallback, err := buildFallbackStrategy(*fallbackConfig)
		if err != nil {
			return nil, fmt.Errorf("building per-node fallback strategy: %w", err)
		}
		fallbackStrategy = fallback
	}
	return newPerNodeStrategy(fallbackStrategy), nil
}

func (*perNodeStrategy) GetName() string {
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
