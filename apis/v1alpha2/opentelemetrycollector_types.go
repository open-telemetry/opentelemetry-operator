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

// +kubebuilder:skip

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.
type OpenTelemetryCollectorSpec struct {
	// Common defines fields that are common to all OpenTelemetry CRD workloads.
	Common OpenTelemetryCommonFields `json:",inline"`
	// TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.
	// +optional
	TargetAllocator v1alpha1.OpenTelemetryTargetAllocator `json:"targetAllocator,omitempty"`
	// Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar)
	// +optional
	Mode v1alpha1.Mode `json:"mode,omitempty"`
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed
	// +optional
	UpgradeStrategy v1alpha1.UpgradeStrategy `json:"upgradeStrategy"`
	// Config is the raw JSON to be used as the collector's configuration. Refer to the OpenTelemetry Collector documentation for details.
	// +required
	Config string `json:"config,omitempty"`
	// Ingress is used to specify how OpenTelemetry Collector is exposed. This
	// functionality is only available if one of the valid modes is set.
	// Valid modes are: deployment, daemonset and statefulset.
	// +optional
	Ingress v1alpha1.Ingress `json:"ingress,omitempty"`
	// Liveness config for the OpenTelemetry Collector except the probe handler which is auto generated from the health extension of the collector.
	// It is only effective when healthcheckextension is configured in the OpenTelemetry Collector pipeline.
	// +optional
	LivenessProbe *v1alpha1.Probe `json:"livenessProbe,omitempty"`

	// ObservabilitySpec defines how telemetry data gets handled.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Observability"
	Observability v1alpha1.ObservabilitySpec `json:"observability,omitempty"`

	// ConfigMaps is a list of ConfigMaps in the same namespace as the OpenTelemetryCollector
	// object, which shall be mounted into the Collector Pods.
	// Each ConfigMap will be added to the Collector's Deployments as a volume named `configmap-<configmap-name>`.
	ConfigMaps []v1alpha1.ConfigMapsSpec `json:"configmaps,omitempty"`
}

// OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.
type OpenTelemetryCollectorStatus struct {
	// Scale is the OpenTelemetryCollector's scale subresource status.
	// +optional
	Scale v1alpha1.ScaleSubresourceStatus `json:"scale,omitempty"`

	// Version of the managed OpenTelemetry Collector (operand)
	// +optional
	Version string `json:"version,omitempty"`

	// Image indicates the container image to use for the OpenTelemetry Collector.
	// +optional
	Image string `json:"image,omitempty"`

	// Messages about actions performed by the operator on this resource.
	// +optional
	// +listType=atomic
	// Deprecated: use Kubernetes events instead.
	Messages []string `json:"messages,omitempty"`

	// Replicas is currently not being set and might be removed in the next version.
	// +optional
	// Deprecated: use "OpenTelemetryCollector.Status.Scale.Replicas" instead.
	Replicas int32 `json:"replicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenTelemetryCollector is the Schema for the opentelemetrycollectors API.
type OpenTelemetryCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenTelemetryCollectorSpec   `json:"spec,omitempty"`
	Status OpenTelemetryCollectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenTelemetryCollectorList contains a list of OpenTelemetryCollector.
type OpenTelemetryCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
}
