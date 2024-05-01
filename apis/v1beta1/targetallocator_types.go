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

package v1beta1

import (
	"github.com/open-telemetry/opentelemetry-operator/internal/api/common"
)

type (
	// TargetAllocatorPrometheusCR configures Prometheus CustomResource handling in the Target Allocator.
	TargetAllocatorPrometheusCR common.TargetAllocatorPrometheusCR
	// TargetAllocatorAllocationStrategy represent a strategy Target Allocator uses to distribute targets to each collector.
	TargetAllocatorAllocationStrategy common.TargetAllocatorAllocationStrategy
	// TargetAllocatorFilterStrategy represent a filtering strategy for targets before they are assigned to collectors.
	TargetAllocatorFilterStrategy common.TargetAllocatorFilterStrategy
)

const (
	// TargetAllocatorAllocationStrategyLeastWeighted targets will be distributed to collector with fewer targets currently assigned.
	TargetAllocatorAllocationStrategyLeastWeighted = TargetAllocatorAllocationStrategy(common.TargetAllocatorAllocationStrategyLeastWeighted)

	// TargetAllocatorAllocationStrategyConsistentHashing targets will be consistently added to collectors, which allows a high-availability setup.
	TargetAllocatorAllocationStrategyConsistentHashing = TargetAllocatorAllocationStrategy(common.TargetAllocatorAllocationStrategyConsistentHashing)

	// TargetAllocatorAllocationStrategyPerNode targets will be assigned to the collector on the node they reside on (use only with daemon set).
	TargetAllocatorAllocationStrategyPerNode TargetAllocatorAllocationStrategy = TargetAllocatorAllocationStrategy(common.TargetAllocatorAllocationStrategyPerNode)

	// TargetAllocatorFilterStrategyRelabelConfig targets will be consistently drops targets based on the relabel_config.
	TargetAllocatorFilterStrategyRelabelConfig = TargetAllocatorFilterStrategy(common.TargetAllocatorFilterStrategyRelabelConfig)
)
