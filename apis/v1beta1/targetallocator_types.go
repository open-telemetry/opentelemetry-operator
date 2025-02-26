// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	PodMonitorSelector *metav1.LabelSelector `json:"podMonitorSelector,omitempty"`
	// ServiceMonitors to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ServiceMonitorSelector *metav1.LabelSelector `json:"serviceMonitorSelector,omitempty"`
	// ScrapeConfigs to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ScrapeConfigSelector *metav1.LabelSelector `json:"scrapeConfigSelector,omitempty"`
	// Probes to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ProbeSelector *metav1.LabelSelector `json:"probeSelector,omitempty"`
}

type (
	// TargetAllocatorAllocationStrategy represent a strategy Target Allocator uses to distribute targets to each collector
	// +kubebuilder:validation:Enum=least-weighted;consistent-hashing;per-node
	TargetAllocatorAllocationStrategy string
	// TargetAllocatorFilterStrategy represent a filtering strategy for targets before they are assigned to collectors
	// +kubebuilder:validation:Enum="";relabel-config
	TargetAllocatorFilterStrategy string
)

const (
	// TargetAllocatorAllocationStrategyLeastWeighted targets will be distributed to collector with fewer targets currently assigned.
	TargetAllocatorAllocationStrategyLeastWeighted TargetAllocatorAllocationStrategy = "least-weighted"

	// TargetAllocatorAllocationStrategyConsistentHashing targets will be consistently added to collectors, which allows a high-availability setup.
	TargetAllocatorAllocationStrategyConsistentHashing TargetAllocatorAllocationStrategy = "consistent-hashing"

	// TargetAllocatorAllocationStrategyPerNode targets will be assigned to the collector on the node they reside on (use only with daemon set).
	TargetAllocatorAllocationStrategyPerNode TargetAllocatorAllocationStrategy = "per-node"

	// TargetAllocatorFilterStrategyRelabelConfig targets will be consistently drops targets based on the relabel_config.
	TargetAllocatorFilterStrategyRelabelConfig TargetAllocatorFilterStrategy = "relabel-config"
)
