// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const leastWeightedStrategyName = "least-weighted"

var _ Strategy = &leastWeightedStrategy{}

type leastWeightedStrategy struct{}

func newleastWeightedStrategy() Strategy {
	return &leastWeightedStrategy{}
}

func (s *leastWeightedStrategy) GetName() string {
	return leastWeightedStrategyName
}

func (s *leastWeightedStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	// if a collector is already assigned, do nothing
	// TODO: track this in a separate map
	if item.CollectorName != "" {
		if col, ok := collectors[item.CollectorName]; ok {
			return col, nil
		}
	}

	var col *Collector
	for _, v := range collectors {
		// If the initial collector is empty, set the initial collector to the first element of map
		if col == nil {
			col = v
		} else if v.NumTargets < col.NumTargets {
			col = v
		}
	}
	return col, nil
}

func (s *leastWeightedStrategy) SetCollectors(_ map[string]*Collector) {}

func (s *leastWeightedStrategy) SetFallbackStrategy(fallbackStrategy Strategy) {}
