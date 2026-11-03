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
	// AllowNamespaces Namespaces to scope the interaction of the Target Allocator and the apiserver (allow list). This is mutually exclusive with DenyNamespaces.
	// +optional
	AllowNamespaces []string `json:"allowNamespaces,omitempty"`
	// DenyNamespaces Namespaces to scope the interaction of the Target Allocator and the apiserver (deny list). This is mutually exclusive with AllowNamespaces.
	// +optional
	DenyNamespaces []string `json:"denyNamespaces,omitempty"`
	// SecretNamespaces Namespaces to scope the watching of secrets for the Target Allocator.
	// If not configured, defaults to the target allocator's own namespace.
	// +optional
	SecretNamespaces []string `json:"secretNamespaces,omitempty"`
	// DenyFSAccessThroughSMs causes the Target Allocator to drop ServiceMonitor and
	// PodMonitor endpoints that reference arbitrary files on the file system. When
	// enabled, endpoints with bearerTokenFile, tlsConfig.caFile, tlsConfig.certFile,
	// or tlsConfig.keyFile are dropped from the produced scrape configuration while
	// the remaining endpoints are kept. This prevents tenants from stealing the
	// Collector's service account token via ServiceMonitor bearerTokenFile
	// references. This is the equivalent of ArbitraryFSAccessThroughSMs.Deny from
	// the Prometheus Operator.
	// +optional
	DenyFSAccessThroughSMs bool `json:"denyFSAccessThroughSMs,omitempty"`
	// Default interval between consecutive scrapes. Intervals set in ServiceMonitors and PodMonitors override it.
	//
	// Default: "30s"
	// +kubebuilder:default:="30s"
	// +kubebuilder:validation:Format:=duration
	ScrapeInterval *metav1.Duration `json:"scrapeInterval,omitempty"`
	// Default interval between rule evaluations.
	//
	// Default: "30s"
	// +kubebuilder:default:="30s"
	// +kubebuilder:validation:Format:=duration
	// +optional
	EvaluationInterval *metav1.Duration `json:"evaluationInterval,omitempty"`
	// ScrapeProtocols define the protocols to negotiate during a scrape. It tells clients the
	// protocols supported by Prometheus in order of preference (from most to least preferred).
	// +optional
	ScrapeProtocols []string `json:"scrapeProtocols,omitempty"`
	// ScrapeClasses to be referenced by PodMonitors and ServiceMonitors to include common configuration.
	// If specified, expects an array of ScrapeClass objects as specified by https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.ScrapeClass.
	// +optional
	// +listType=atomic
	// +kubebuilder:pruning:PreserveUnknownFields
	ScrapeClasses []AnyConfig `json:"scrapeClasses,omitempty"`
	// PodMonitors to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	PodMonitorSelector *metav1.LabelSelector `json:"podMonitorSelector,omitempty"`
	// Namespaces to be selected for PodMonitor discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	// +kubebuilder:default:={}
	PodMonitorNamespaceSelector *metav1.LabelSelector `json:"podMonitorNamespaceSelector,omitempty"`
	// ServiceMonitors to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ServiceMonitorSelector *metav1.LabelSelector `json:"serviceMonitorSelector,omitempty"`
	// Namespaces to be selected for ServiceMonitor discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	// +kubebuilder:default:={}
	ServiceMonitorNamespaceSelector *metav1.LabelSelector `json:"serviceMonitorNamespaceSelector,omitempty"`
	// ScrapeConfigs to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ScrapeConfigSelector *metav1.LabelSelector `json:"scrapeConfigSelector,omitempty"`
	// Namespaces to be selected for ScrapeConfig discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	// +kubebuilder:default:={}
	ScrapeConfigNamespaceSelector *metav1.LabelSelector `json:"scrapeConfigNamespaceSelector,omitempty"`
	// Probes to be selected for target discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	ProbeSelector *metav1.LabelSelector `json:"probeSelector,omitempty"`
	// Namespaces to be selected for Probe discovery.
	// A label selector is a label query over a set of resources. The result of matchLabels and
	// matchExpressions are ANDed. An empty label selector matches all objects. A null
	// label selector matches no objects.
	// +optional
	// +kubebuilder:default:={}
	ProbeNamespaceSelector *metav1.LabelSelector `json:"probeNamespaceSelector,omitempty"`
}

type (
	// TargetAllocatorAllocationStrategy represent a strategy Target Allocator uses to distribute targets to each collector
	// +kubebuilder:validation:Enum=least-weighted;consistent-hashing;per-node
	TargetAllocatorAllocationStrategy string
	// TargetAllocatorFilterStrategy represent a filtering strategy for targets before they are assigned to collectors
	// +kubebuilder:validation:Enum="";none;relabel-config
	TargetAllocatorFilterStrategy string
	// TargetAllocatorFallbackAllocationStrategy represents a strategy which can be used as the fallback for
	// another allocation strategy. It is the set of allocation strategies minus per-node: the Target Allocator
	// rejects per-node as a fallback, because a per-node fallback would fail on exactly the targets the primary
	// strategy failed to assign.
	// +kubebuilder:validation:Enum=least-weighted;consistent-hashing
	TargetAllocatorFallbackAllocationStrategy string
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

	// TargetAllocatorFilterStrategyNone disables filtering of targets before they are assigned to collectors.
	TargetAllocatorFilterStrategyNone TargetAllocatorFilterStrategy = "none"

	// TargetAllocatorFallbackAllocationStrategyLeastWeighted uses the least-weighted strategy as the fallback.
	TargetAllocatorFallbackAllocationStrategyLeastWeighted TargetAllocatorFallbackAllocationStrategy = "least-weighted"

	// TargetAllocatorFallbackAllocationStrategyConsistentHashing uses the consistent-hashing strategy as the fallback.
	TargetAllocatorFallbackAllocationStrategyConsistentHashing TargetAllocatorFallbackAllocationStrategy = "consistent-hashing"
)

// TargetAllocatorAllocationStrategyConfig holds per-strategy configuration for the allocation strategies.
// Each allocation strategy has its own section because strategies accept different configuration options.
type TargetAllocatorAllocationStrategyConfig struct {
	// PerNode holds the configuration options for the per-node allocation strategy.
	// +optional
	PerNode TargetAllocatorPerNodeStrategyConfig `json:"perNode,omitempty"`
}

// TargetAllocatorPerNodeStrategyConfig holds the configuration options for the per-node allocation strategy.
type TargetAllocatorPerNodeStrategyConfig struct {
	// FallbackStrategy configures the allocation strategy used for targets the per-node strategy can't assign
	// on their own, for example targets which don't reside on a Node. If unset, such targets are left
	// unassigned.
	// +optional
	FallbackStrategy *TargetAllocatorFallbackStrategyConfig `json:"fallbackStrategy,omitempty"`
}

// TargetAllocatorFallbackStrategyConfig configures an allocation strategy used as a fallback: the strategy
// name plus the options for that strategy. It mirrors TargetAllocatorAllocationStrategyConfig, except that
// strategies used as fallbacks can't have fallbacks of their own, which keeps fallback chains bounded to a
// single level.
type TargetAllocatorFallbackStrategyConfig struct {
	// Name is the name of the allocation strategy to use as the fallback. The per-node strategy can't be
	// used as a fallback, as it would fail on exactly the targets the primary strategy failed to assign.
	// +kubebuilder:validation:Required
	Name TargetAllocatorFallbackAllocationStrategy `json:"name"`
}
