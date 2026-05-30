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

// TargetAllocatorTopology configures availability-zone aware target
// allocation. Forwarded verbatim into the allocator's runtime config so
// every field below behaves exactly like the matching key documented
// for the cmd/otel-allocator config file.
//
// When ZoneAware is false (the default) the rest of this section has no
// effect and allocation behaves identically to releases without this
// feature.
type TargetAllocatorTopology struct {
	// ZoneAware enables zone-aware allocation for the consistent-hashing
	// and least-weighted strategies. Targets are preferentially assigned
	// to collectors running in the same topology zone, reducing
	// cross-AZ scrape traffic and the associated cloud egress costs.
	// +optional
	ZoneAware bool `json:"zoneAware,omitempty"`
	// ZoneLabel is the node label used to look up the zone each
	// collector pod runs in. Defaults to the standard
	// "topology.kubernetes.io/zone" set by kubelets on every major cloud
	// provider. The legacy "failure-domain.beta.kubernetes.io/zone"
	// label is used automatically as a fallback.
	// +optional
	ZoneLabel string `json:"zoneLabel,omitempty"`
	// TargetZoneLabel is the Prometheus service-discovery meta-label
	// used to read a target's desired zone. Defaults to
	// "__meta_kubernetes_endpointslice_endpoint_zone" which is populated
	// automatically when scraping EndpointSlice resources. For EC2 SD
	// use "__meta_ec2_availability_zone", for GCE SD use
	// "__meta_gce_zone".
	//
	// IMPORTANT: this label MUST be low-cardinality. The allocator keeps
	// an in-memory map and a Prometheus metric label per distinct value
	// it sees, so pointing this at a high-cardinality label (instance
	// IDs, pod names, IP addresses) will grow memory and the
	// `opentelemetry_allocator_*_zone*` series count linearly with
	// target count. Real cloud topologies have a handful of zones per
	// region. The allocator emits a one-time warning when distinct-zone
	// cardinality crosses 64 to surface misconfiguration, but it does
	// not enforce a hard cap — picking a sensible label is the
	// operator's responsibility.
	// +optional
	TargetZoneLabel string `json:"targetZoneLabel,omitempty"`
	// MaxSkew controls cross-zone "spillover". When a same-zone
	// assignment would push the global target-count skew (max minus min
	// across all collectors) above this value, the target is assigned
	// to the globally least-loaded collector instead. 0 (the default)
	// disables the check entirely — pure zone affinity. Values of 5–20
	// are practical for production setups with uneven workloads.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxSkew int32 `json:"maxSkew,omitempty"`
	// NodeSyncInterval controls how often the allocator re-reads node
	// zone labels from the Kubernetes API so new or relabeled nodes are
	// picked up without restarting the allocator. Defaults to 5m. Set
	// to 0 to disable periodic re-sync (sync only on startup); minimum
	// valid non-zero value is 30s.
	// +optional
	// +kubebuilder:validation:Format:=duration
	NodeSyncInterval *metav1.Duration `json:"nodeSyncInterval,omitempty"`
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
