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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/api/common"
)

func init() {
	SchemeBuilder.Register(&TargetAllocator{}, &TargetAllocatorList{})
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TargetAllocator is the Schema for the targetallocators API.
type TargetAllocator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TargetAllocatorSpec   `json:"spec,omitempty"`
	Status TargetAllocatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TargetAllocatorList contains a list of TargetAllocator.
type TargetAllocatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollector `json:"items"`
}

// TargetAllocatorStatus defines the observed state of Target Allocator.
type TargetAllocatorStatus struct {
	// Version of the managed Target Allocator (operand)
	// +optional
	Version string `json:"version,omitempty"`

	// Image indicates the container image to use for the Target Allocator.
	// +optional
	Image string `json:"image,omitempty"`
}

// TargetAllocatorSpec defines the desired state of TargetAllocator.
type TargetAllocatorSpec struct {
	// Common defines fields that are common to all OpenTelemetry CRD workloads.
	common.OpenTelemetryCommonFields `json:",inline"`
	// CollectorSelector is the selector for Collector Pods the target allocator will allocate targets to.
	CollectorSelector metav1.LabelSelector `json:"collectorSelector,omitempty"`
	// AllocationStrategy determines which strategy the target allocator should use for allocation.
	// The current options are least-weighted, consistent-hashing and per-node. The default is
	// consistent-hashing.
	// WARNING: The per-node strategy currently ignores targets without a Node, like control plane components.
	// +optional
	// +kubebuilder:default:=consistent-hashing
	AllocationStrategy TargetAllocatorAllocationStrategy `json:"allocationStrategy,omitempty"`
	// FilterStrategy determines how to filter targets before allocating them among the collectors.
	// The only current option is relabel-config (drops targets based on prom relabel_config).
	// The default is relabel-config.
	// +optional
	// +kubebuilder:default:=relabel-config
	FilterStrategy TargetAllocatorFilterStrategy `json:"filterStrategy,omitempty"`
	// ScrapeConfigs define static Prometheus scrape configurations for the target allocator.
	// To use dynamic configurations from ServiceMonitors and PodMonitors, see the PrometheusCR section.
	// For the exact format, see https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config.
	// +optional
	// +listType=atomic
	// +kubebuilder:pruning:PreserveUnknownFields
	ScrapeConfigs []common.AnyConfig `json:"scrapeConfigs,omitempty"`
	// PrometheusCR defines the configuration for the retrieval of PrometheusOperator CRDs ( servicemonitor.monitoring.coreos.com/v1 and podmonitor.monitoring.coreos.com/v1 ).
	// +optional
	PrometheusCR TargetAllocatorPrometheusCR `json:"prometheusCR,omitempty"`
	// ObservabilitySpec defines how telemetry data gets handled.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Observability"
	Observability common.ObservabilitySpec `json:"observability,omitempty"`
}

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
