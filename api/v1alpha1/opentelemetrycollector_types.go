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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.
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

	// TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	TargetAllocator OpenTelemetryTargetAllocatorSpec `json:"targetAllocator,omitempty"`

	// Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar)
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Mode Mode `json:"mode,omitempty"`

	// ServiceAccount indicates the name of an existing service account to use with this instance.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// SecurityContext will be set as the container security context.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	SecurityContext *v1.SecurityContext `json:"securityContext,omitempty"`

	// HostNetwork indicates if the pod should run in the host networking namespace.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	HostNetwork bool `json:"hostNetwork,omitempty"`

	// VolumeClaimTemplates will provide stable storage using PersistentVolumes. Only available when the mode=statefulset.
	// +optional
	// +listType=atomic
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// VolumeMounts represents the mount points to use in the underlying collector deployment(s)
	// +optional
	// +listType=atomic
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`

	// Volumes represents which volumes to use in the underlying collector deployment(s).
	// +optional
	// +listType=atomic
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Volumes []v1.Volume `json:"volumes,omitempty"`

	// Ports allows a set of ports to be exposed by the underlying v1.Service. By default, the operator
	// will attempt to infer the required ports by parsing the .Spec.Config property but this property can be
	// used to open aditional ports that can't be inferred by the operator, like for custom receivers.
	// +optional
	// +listType=atomic
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Ports []v1.ServicePort `json:"ports,omitempty"`

	// ENV vars to set on the OpenTelemetry Collector's Pods. These can then in certain cases be
	// consumed in the config file for the Collector.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Env []v1.EnvVar `json:"env,omitempty"`

	// Resources to set on the OpenTelemetry Collector pods.
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// Toleration to schedule OpenTelemetry Collector pods.
	// This is only relevant to daemonsets, statefulsets and deployments
	// +optional
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.
type OpenTelemetryCollectorStatus struct {
	// Replicas is currently not being set and might be removed in the next version.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Version of the managed OpenTelemetry Collector (operand)
	// +optional
	Version string `json:"version,omitempty"`

	// Messages about actions performed by the operator on this resource.
	// +optional
	// +listType=atomic
	Messages []string `json:"messages,omitempty"`
}

// OpenTelemetryTargetAllocatorSpec defines the configurations for the Prometheus target allocator.
type OpenTelemetryTargetAllocatorSpec struct {
	// Enabled indicates whether to use a target allocation mechanism for Prometheus targets or not.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Image indicates the container image to use for the OpenTelemetry TargetAllocator.
	// +optional
	Image string `json:"image,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelcol;otelcols
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode",description="Deployment Mode"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="OpenTelemetry Version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Collector"

// OpenTelemetryCollector is the Schema for the opentelemetrycollectors API.
type OpenTelemetryCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenTelemetryCollectorSpec   `json:"spec,omitempty"`
	Status OpenTelemetryCollectorStatus `json:"status,omitempty"`
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
