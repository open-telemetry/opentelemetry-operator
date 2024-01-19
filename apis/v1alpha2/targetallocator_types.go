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

// +kubebuilder:skip

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TargetAllocatorSpec defines the desired state of TargetAllocator.
type TargetAllocatorSpec struct {
	// Common defines fields that are common to all OpenTelemetry CRD workloads.
	OpenTelemetryCommonFields `json:",inline"`
	// AllocationStrategy determines which strategy the target allocator should use for allocation.
	// The current options are least-weighted and consistent-hashing. The default option is consistent-hashing
	// +optional
	// +kubebuilder:default:=consistent-hashing
	AllocationStrategy TargetAllocatorAllocationStrategy `json:"allocationStrategy,omitempty"`
	// FilterStrategy determines how to filter targets before allocating them among the collectors.
	// The only current option is relabel-config (drops targets based on prom relabel_config).
	// The default is relabel-config.
	// +optional
	// +kubebuilder:default:=relabel-config
	FilterStrategy TargetAllocatorFilterStrategy `json:"filterStrategy,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance. When set,
	// the operator will not automatically create a ServiceAccount for the TargetAllocator.
	// +optional
	PrometheusCR TargetAllocatorPrometheusCR `json:"prometheusCR,omitempty"`
}

// TargetAllocatorPrometheusCR configures Prometheus CustomResource handling in the Target Allocator.
type TargetAllocatorPrometheusCR struct {
	// Enabled indicates whether to use a PrometheusOperator custom resources as targets or not.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// Default interval between consecutive scrapes. Intervals set in ServiceMonitors and PodMonitors override it.
	//Equivalent to the same setting on the Prometheus CR.
	//
	// Default: "30s"
	// +kubebuilder:default:="30s"
	// +kubebuilder:validation:Format:=duration
	ScrapeInterval *metav1.Duration `json:"scrapeInterval,omitempty"`
	// PodMonitors to be selected for target discovery.
	// This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a
	// PodMonitor's meta labels. The requirements are ANDed.
	// +optional
	PodMonitorSelector *metav1.LabelSelector `json:"podMonitorSelector,omitempty"`
	// ServiceMonitors to be selected for target discovery.
	// This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a
	// ServiceMonitor's meta labels. The requirements are ANDed.
	// +optional
	ServiceMonitorSelector *metav1.LabelSelector `json:"serviceMonitorSelector,omitempty"`
}

// TargetAllocatorStatus defines the observed state of Target Allocator.
type TargetAllocatorStatus struct {
	// Version of the managed Target Allocator (operand)
	// +optional
	Version string `json:"version,omitempty"`

	// Image indicates the container image to use for the Target Allocator.
	// +optional
	Image string `json:"image,omitempty"`

	// Messages about actions performed by the operator on this resource.
	// +optional
	// +listType=atomic
	// Deprecated: use Kubernetes events instead.
	Messages []string `json:"messages,omitempty"`
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

func init() {
	SchemeBuilder.Register(&TargetAllocator{}, &TargetAllocatorList{})
}
