// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
