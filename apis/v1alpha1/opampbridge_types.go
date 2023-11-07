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

// OpAMPBridgeSpec defines the desired state of OpAMPBridge.
type OpAMPBridgeSpec struct {
	// Common defines fields that are common to all OpenTelemetry CRD workloads.
	Common OpenTelemetryCommonFields `json:",inline"`
	// OpAMP backend Server endpoint
	// +required
	Endpoint string `json:"endpoint"`
	// Capabilities supported by the OpAMP Bridge
	// +required
	Capabilities map[OpAMPBridgeCapability]bool `json:"capabilities"`
	// ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)
	// +optional
	ComponentsAllowed map[string][]string `json:"componentsAllowed,omitempty"`
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed
	// +optional
	UpgradeStrategy UpgradeStrategy `json:"upgradeStrategy"`
}

// OpAMPBridgeStatus defines the observed state of OpAMPBridge.
type OpAMPBridgeStatus struct {
	// Version of the managed OpAMP Bridge (operand)
	// +optional
	Version string `json:"version,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="OpenTelemetry Version"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.endpoint"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpAMP Bridge"
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1},{Deployment,apps/v1},{ConfigMaps,v1},{Service,v1}}

// OpAMPBridge is the Schema for the opampbridges API.
type OpAMPBridge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpAMPBridgeSpec   `json:"spec,omitempty"`
	Status OpAMPBridgeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpAMPBridgeList contains a list of OpAMPBridge.
type OpAMPBridgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpAMPBridge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpAMPBridge{}, &OpAMPBridgeList{})
}
