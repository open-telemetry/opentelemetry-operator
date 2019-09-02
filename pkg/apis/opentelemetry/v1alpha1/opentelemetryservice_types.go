package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenTelemetryServiceSpec defines the desired state of OpenTelemetryService
// +k8s:openapi-gen=true
type OpenTelemetryServiceSpec struct {
	// +required
	Config string `json:"config,omitempty"`

	// +optional
	Args map[string]string `json:"args,omitempty"`

	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +optional
	Image string `json:"image,omitempty"`
}

// OpenTelemetryServiceStatus defines the observed state of OpenTelemetryService
// +k8s:openapi-gen=true
type OpenTelemetryServiceStatus struct {
	Replicas int32 `json:"replicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenTelemetryService is the Schema for the opentelemetryservices API
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName=otelsvc;otelsvcs
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
type OpenTelemetryService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenTelemetryServiceSpec   `json:"spec,omitempty"`
	Status OpenTelemetryServiceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OpenTelemetryServiceList contains a list of OpenTelemetryService
type OpenTelemetryServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryService{}, &OpenTelemetryServiceList{})
}
