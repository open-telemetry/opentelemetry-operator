package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector
// +k8s:openapi-gen=true
type OpenTelemetryCollectorSpec struct {
	// +required
	Config string `json:"config,omitempty"`

	// +optional
	Args map[string]string `json:"args,omitempty"`

	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`
}

// OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector
// +k8s:openapi-gen=true
type OpenTelemetryCollectorStatus struct {
	Replicas int32  `json:"replicas"`
	Version  string `json:"version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenTelemetryCollector is the Schema for the opentelemetrycollectors API
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName=otelcol;otelcols
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
type OpenTelemetryCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenTelemetryCollectorSpec   `json:"spec,omitempty"`
	Status OpenTelemetryCollectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenTelemetryCollectorList contains a list of OpenTelemetryCollector
type OpenTelemetryCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
}
