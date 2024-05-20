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
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const perNodeStrategyName = "per-node"

var _ Strategy = &perNodeStrategy{}

type perNodeStrategy struct {
	collectorByNode map[string]*Collector
}

func newPerNodeStrategy() Strategy {
	return &perNodeStrategy{
		collectorByNode: make(map[string]*Collector),
	}
}

func (s *perNodeStrategy) GetName() string {
	return perNodeStrategyName
}

func (s *perNodeStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	targetNodeName := item.GetNodeName()
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
}
