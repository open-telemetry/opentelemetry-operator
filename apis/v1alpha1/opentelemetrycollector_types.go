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
	// Resources to set on the OpenTelemetry Collector pods.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// NodeSelector to schedule OpenTelemetry Collector pods.
	// This is only relevant to daemonset, statefulset, and deployment mode
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Args is the set of arguments to pass to the OpenTelemetry Collector binary
	// +optional
	Args map[string]string `json:"args,omitempty"`
	// Replicas is the number of pod instances for the underlying OpenTelemetry Collector. Set this if your are not using autoscaling
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// MinReplicas sets a lower bound to the autoscaling feature.  Set this if your are using autoscaling. It must be at least 1
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// MaxReplicas sets an upper bound to the autoscaling feature. If MaxReplicas is set autoscaling is enabled.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// TargetCPUUtilization sets the target average CPU used across all replicas.
	// If average CPU exceeds this value, the HPA will scale up. Defaults to 90 percent.
	// +optional
	TargetCPUUtilization *int32 `json:"targetCPUUtilization,omitempty"`
	// SecurityContext will be set as the container security context.
	// +optional
	SecurityContext *v1.SecurityContext `json:"securityContext,omitempty"`

	PodSecurityContext *v1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// PodAnnotations is the set of annotations that will be attached to
	// Collector and Target Allocator pods.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.
	// +optional
	TargetAllocator OpenTelemetryTargetAllocator `json:"targetAllocator,omitempty"`
	// Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar)
	// +optional
	Mode Mode `json:"mode,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// Image indicates the container image to use for the OpenTelemetry Collector.
	// +optional
	Image string `json:"image,omitempty"`
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed
	// +optional
	UpgradeStrategy UpgradeStrategy `json:"upgradeStrategy"`

	// ImagePullPolicy indicates the pull policy to be used for retrieving the container image (Always, Never, IfNotPresent)
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Config is the raw JSON to be used as the collector's configuration. Refer to the OpenTelemetry Collector documentation for details.
	// +required
	Config string `json:"config,omitempty"`
	// VolumeMounts represents the mount points to use in the underlying collector deployment(s)
	// +optional
	// +listType=atomic
	VolumeMounts []v1.VolumeMount `json:"volumeMounts,omitempty"`
	// Ports allows a set of ports to be exposed by the underlying v1.Service. By default, the operator
	// will attempt to infer the required ports by parsing the .Spec.Config property but this property can be
	// used to open aditional ports that can't be inferred by the operator, like for custom receivers.
	// +optional
	// +listType=atomic
	Ports []v1.ServicePort `json:"ports,omitempty"`
	// ENV vars to set on the OpenTelemetry Collector's Pods. These can then in certain cases be
	// consumed in the config file for the Collector.
	// +optional
	Env []v1.EnvVar `json:"env,omitempty"`
	// List of sources to populate environment variables on the OpenTelemetry Collector's Pods.
	// These can then in certain cases be consumed in the config file for the Collector.
	// +optional
	EnvFrom []v1.EnvFromSource `json:"envFrom,omitempty"`
	// VolumeClaimTemplates will provide stable storage using PersistentVolumes. Only available when the mode=statefulset.
	// +optional
	// +listType=atomic
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	// Toleration to schedule OpenTelemetry Collector pods.
	// This is only relevant to daemonset, statefulset, and deployment mode
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	// Volumes represents which volumes to use in the underlying collector deployment(s).
	// +optional
	// +listType=atomic
	Volumes []v1.Volume `json:"volumes,omitempty"`
	// HostNetwork indicates if the pod should run in the host networking namespace.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`
}

// OpenTelemetryTargetAllocator defines the configurations for the Prometheus target allocator.
type OpenTelemetryTargetAllocator struct {
	// ServiceAccount indicates the name of an existing service account to use with this instance.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// Image indicates the container image to use for the OpenTelemetry TargetAllocator.
	// +optional
	Image string `json:"image,omitempty"`
	// Enabled indicates whether to use a target allocation mechanism for Prometheus targets or not.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
	// PrometheusCR defines the configuration for the retrieval of PrometheusOperator CRDs ( servicemonitor.monitoring.coreos.com/v1 and podmonitor.monitoring.coreos.com/v1 )  retrieval.
	// All CR instances which the ServiceAccount has access to will be retrieved. This includes other namespaces.
	// +optional
	PrometheusCR OpenTelemetryTargetAllocatorPrometheusCR `json:"prometheusCR,omitempty"`
}

type OpenTelemetryTargetAllocatorPrometheusCR struct {
	// Enabled indicates whether to use a PrometheusOperator custom resources as targets or not.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// ScaleSubresourceStatus defines the observed state of the OpenTelemetryCollector's
// scale subresource.
type ScaleSubresourceStatus struct {
	// The selector used to match the OpenTelemetryCollector's
	// deployment or statefulSet pods.
	// +optional
	Selector string `json:"selector,omitempty"`

	// The total number non-terminated pods targeted by this
	// OpenTelemetryCollector's deployment or statefulSet.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
}

// OpenTelemetryCollectorStatus defines the observed state of OpenTelemetryCollector.
type OpenTelemetryCollectorStatus struct {
	// Scale is the OpenTelemetryCollector's scale subresource status.
	// +optional
	Scale ScaleSubresourceStatus `json:"scale,omitempty"`

	// Version of the managed OpenTelemetry Collector (operand)
	// +optional
	Version string `json:"version,omitempty"`

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

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelcol;otelcols
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.scale.replicas,selectorpath=.status.scale.selector
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode",description="Deployment Mode"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="OpenTelemetry Version"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Collector"
// This annotation provides a hint for OLM which resources are managed by OpenTelemetryCollector kind.
// It's not mandatory to list all resources.
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1},{Deployment,apps/v1},{DaemonSets,apps/v1},{StatefulSets,apps/v1},{ConfigMaps,v1},{Service,v1}}

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
