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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=otelcol;otelcols
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.scale.replicas,selectorpath=.status.scale.selector
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode",description="Deployment Mode"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="OpenTelemetry Version"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.scale.statusReplicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".status.image"
// +kubebuilder:printcolumn:name="Management",type="string",JSONPath=".spec.managementState",description="Management State"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Collector"
// This annotation provides a hint for OLM which resources are managed by OpenTelemetryCollector kind.
// It's not mandatory to list all resources.
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1},{Deployment,apps/v1},{DaemonSets,apps/v1},{StatefulSets,apps/v1},{ConfigMaps,v1},{Service,v1}}

// OpenTelemetryCollector is the Schema for the opentelemetrycollectors API.
type OpenTelemetryCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   v1alpha1.OpenTelemetryCollectorSpec   `json:"spec,omitempty"`
	Status v1alpha1.OpenTelemetryCollectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OpenTelemetryCollectorList contains a list of OpenTelemetryCollector.
type OpenTelemetryCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
}
