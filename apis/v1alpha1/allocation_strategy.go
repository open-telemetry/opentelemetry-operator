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

package v1alpha1

type (
	// OpenTelemetryTargetAllocatorAllocationStrategy represent which strategy to distribute target to each collector
	// +kubebuilder:validation:Enum=least-weighted;consistent-hashing;per-node
	OpenTelemetryTargetAllocatorAllocationStrategy string
)

const (
	// OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted targets will be distributed to collector with fewer targets currently assigned.
	OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted OpenTelemetryTargetAllocatorAllocationStrategy = "least-weighted"

	// OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing targets will be consistently added to collectors, which allows a high-availability setup.
	OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing OpenTelemetryTargetAllocatorAllocationStrategy = "consistent-hashing"

	// OpenTelemetryTargetAllocatorAllocationStrategyPerNode targets will be assigned to the collector on the node they reside on (use only with daemon set).
	OpenTelemetryTargetAllocatorAllocationStrategyPerNode OpenTelemetryTargetAllocatorAllocationStrategy = "per-node"
)
