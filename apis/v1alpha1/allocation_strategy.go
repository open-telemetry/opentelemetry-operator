// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type (
	// OpenTelemetryTargetAllocatorAllocationStrategy represent which strategy to distribute target to each collector
	// +kubebuilder:validation:Enum=least-weighted;consistent-hashing;consistent-hashing-endpoint;per-node
	OpenTelemetryTargetAllocatorAllocationStrategy string
)

const (
	// OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted targets will be distributed to collector with fewer targets currently assigned.
	OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted OpenTelemetryTargetAllocatorAllocationStrategy = "least-weighted"

	// OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing targets will be consistently added to collectors, which allows a high-availability setup.
	OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing OpenTelemetryTargetAllocatorAllocationStrategy = "consistent-hashing"

	// OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashingEndpoint is like consistent-hashing, but keys the
	// hash on the target's scrape endpoint (address, scheme, metrics path, and query params) instead of on the
	// address alone, so targets that share a host:port but differ by path or params are spread across collectors.
	OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashingEndpoint OpenTelemetryTargetAllocatorAllocationStrategy = "consistent-hashing-endpoint"

	// OpenTelemetryTargetAllocatorAllocationStrategyPerNode targets will be assigned to the collector on the node they reside on (use only with daemon set).
	OpenTelemetryTargetAllocatorAllocationStrategyPerNode OpenTelemetryTargetAllocatorAllocationStrategy = "per-node"
)
