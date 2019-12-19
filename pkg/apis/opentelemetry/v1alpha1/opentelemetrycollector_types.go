package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector
// +k8s:openapi-gen=true
type OpenTelemetryCollectorSpec struct {
	// Config is the raw JSON to be used as the collector's configuration. Refer to the OpenTelemetry Collector documentation for details.
	// +required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Config string `json:"config,omitempty"`

	// Args is the set of arguments to pass to the OpenTelemetry Collector binary
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Args map[string]string `json:"args,omitempty"`

	// Replicas is the number of pod instances for the underlying OpenTelemetry Collector
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Replicas *int32 `json:"replicas,omitempty"`

	// Image indicates the container image to use for the OpenTelemetry Collector.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image,omitempty"`

	// Mode represents how the collector should be deployed (deployment vs. daemonset)
	// +optional
	// +kubebuilder:validation:Enum=daemonset;deployment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Mode opentelemetry.Mode `json:"mode,omitempty"`
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
// +genclient
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="OpenTelemetry Collector"
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

// AddToScheme is an alias to SchemeBuilder.AddToScheme, to please client-gen
var AddToScheme = SchemeBuilder.AddToScheme
