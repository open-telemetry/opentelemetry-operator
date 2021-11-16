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
)

// InstrumentationSpec defines the desired state of OpenTelemetry SDK and instrumentation.
type InstrumentationSpec struct {
	// Exporter defines exporter configuration.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Exporter `json:"exporter,omitempty"`

	// ResourceAttributes defines attributes that are added to resource.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ResourceAttributes map[string]string `json:"resourceAttributes,omitempty"`

	// Propagators defines inter-process context propagation configuration.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Propagators []Propagator `json:"propagators,omitempty"`

	// Sampler defines sampling configuration.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Sampler `json:"sampler,omitempty"`

	// Java defines configuration for java auto-instrumentation.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Java JavaSpec `json:"java,omitempty"`

	// NodeJS defines configuration for nodejs auto-instrumentation.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	NodeJS NodeJSSpec `json:"nodejs,omitempty"`
}

// JavaSpec defines Java SDK and instrumentation configuration.
type JavaSpec struct {
	// Image is a container image with javaagent JAR.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image,omitempty"`
}

// NodeJSSpec defines NodeJS SDK and instrumentation configuration.
type NodeJSSpec struct {
	// Image is a container image with NodeJS SDK and autoinstrumentation.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image,omitempty"`
}

// Exporter defines OTLP exporter configuration.
type Exporter struct {
	// Endpoint is address of the collector with OTLP endpoint.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Endpoint string `json:"endpoint,omitempty"`
}

// Sampler defines sampling configuration.
type Sampler struct {
	// Type defines sampler type.
	// The value can be for instance parentbased_always_on, parentbased_always_off, parentbased_traceidratio...
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Type SamplerType `json:"type,omitempty"`

	// Argument defines sampler argument.
	// The value depends on the sampler type.
	// For instance for parentbased_traceidratio sampler type it is a number in range [0..1] e.g. 0.25.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Argument string `json:"argument,omitempty"`
}

// InstrumentationStatus defines status of the instrumentation.
type InstrumentationStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelinst;otelinsts
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Instrumentation"

// Instrumentation is the spec for OpenTelemetry instrumentation.
type Instrumentation struct {
	Status            InstrumentationStatus `json:"status,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstrumentationSpec `json:"spec,omitempty"`
	metav1.TypeMeta   `json:",inline"`
}

// +kubebuilder:object:root=true

// InstrumentationList contains a list of Instrumentation.
type InstrumentationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instrumentation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instrumentation{}, &InstrumentationList{})
}
