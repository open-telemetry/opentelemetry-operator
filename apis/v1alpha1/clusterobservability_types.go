// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// ClusterObservabilitySpec defines the desired state of ClusterObservability.
// This follows a simplified design using a single OTLP HTTP exporter for all signals.
type ClusterObservabilitySpec struct {
	// Exporter configures the OTLP HTTP exporter used by all generated pipelines.
	// Its fields are passed through unchanged.
	// +required
	// +kubebuilder:pruning:PreserveUnknownFields
	Exporter v1beta1.AnyConfig `json:"exporter"`
}

// ClusterObservabilityConditionType represents the type of condition.
type ClusterObservabilityConditionType string

const (
	// ClusterObservabilityConditionReady indicates whether the ClusterObservability is ready.
	ClusterObservabilityConditionReady ClusterObservabilityConditionType = "Ready"
	// ClusterObservabilityConditionConfigured indicates whether the ClusterObservability is configured.
	ClusterObservabilityConditionConfigured ClusterObservabilityConditionType = "Configured"
	// ClusterObservabilityConditionConflicted indicates that multiple ClusterObservability resources exist.
	ClusterObservabilityConditionConflicted ClusterObservabilityConditionType = "Conflicted"
)

const (
	// ClusterObservabilityFinalizer is the finalizer used for ClusterObservability resources.
	ClusterObservabilityFinalizer = "clusterobservability.opentelemetry.io/finalizer"
)

// ClusterObservabilityCondition represents a condition of a ClusterObservability.
type ClusterObservabilityCondition struct {
	// Type of condition.
	// +required
	Type ClusterObservabilityConditionType `json:"type"`

	// Status of the condition.
	// +required
	Status metav1.ConditionStatus `json:"status"`

	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ClusterObservabilityStatus defines the observed state of ClusterObservability.
type ClusterObservabilityStatus struct {
	// Conditions represent the latest available observations of the ClusterObservability state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []ClusterObservabilityCondition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed for this ClusterObservability.
	// It corresponds to the ClusterObservability's generation, which is updated on mutation
	// by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the ClusterObservability.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Message provides additional information about the current state.
	// +optional
	Message string `json:"message,omitempty"`

	// ComponentsStatus provides status information about individual observability components.
	// +optional
	ComponentsStatus map[string]ComponentStatus `json:"componentsStatus,omitempty"`

	// ConfigVersions tracks the version hashes of the configuration files used.
	// This enables detection of config changes when operator is upgraded.
	// +optional
	ConfigVersions map[string]string `json:"configVersions,omitempty"`
}

// ComponentStatus represents the status of an individual component.
type ComponentStatus struct {
	// Ready indicates whether the component is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Message provides additional information about the component status.
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdated is the last time this status was updated.
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.exporter.endpoint",description="OTLP exporter endpoint"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:displayName="Cluster Observability"
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1},{Deployment,apps/v1},{ConfigMap,v1},{Service,v1},{DaemonSet,apps/v1}}

// ClusterObservability is the Schema for the clusterobservabilities API.
type ClusterObservability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterObservabilitySpec   `json:"spec,omitempty"`
	Status ClusterObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterObservabilityList contains a list of ClusterObservability.
type ClusterObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterObservability `json:"items"`
}
