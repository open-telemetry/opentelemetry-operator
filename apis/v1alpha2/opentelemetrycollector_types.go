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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// OpenTelemetryCollectorSpec defines the desired state of OpenTelemetryCollector.
type OpenTelemetryCollectorSpec struct {
	// ManagementState defines if the CR should be managed by the operator or not.
	// Default is managed.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=managed
	ManagementState v1alpha1.ManagementStateType `json:"managementState,omitempty"`
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
	// Autoscaler specifies the pod autoscaling configuration to use
	// for the OpenTelemetryCollector workload.
	//
	// +optional
	Autoscaler *v1alpha1.AutoscalerSpec `json:"autoscaler,omitempty"`
	// PodDisruptionBudget specifies the pod disruption budget configuration to use
	// for the OpenTelemetryCollector workload.
	//
	// +optional
	PodDisruptionBudget *v1alpha1.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty"`
	// SecurityContext configures the container security context for
	// the opentelemetry-collector container.
	//
	// In deployment, daemonset, or statefulset mode, this controls
	// the security context settings for the primary application
	// container.
	//
	// In sidecar mode, this controls the security context for the
	// injected sidecar container.
	//
	// +optional
	SecurityContext *v1.SecurityContext `json:"securityContext,omitempty"`
	// PodSecurityContext configures the pod security context for the
	// opentelemetry-collector pod, when running as a deployment, daemonset,
	// or statefulset.
	//
	// In sidecar mode, the opentelemetry-operator will ignore this setting.
	//
	// +optional
	PodSecurityContext *v1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	// PodAnnotations is the set of annotations that will be attached to
	// Collector and Target Allocator pods.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// TargetAllocator indicates a value which determines whether to spawn a target allocation resource or not.
	// +optional
	TargetAllocator v1alpha1.OpenTelemetryTargetAllocator `json:"targetAllocator,omitempty"`
	// Mode represents how the collector should be deployed (deployment, daemonset, statefulset or sidecar)
	// +optional
	Mode v1alpha1.Mode `json:"mode,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance. When set,
	// the operator will not automatically create a ServiceAccount for the collector.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// Image indicates the container image to use for the OpenTelemetry Collector.
	// +optional
	Image string `json:"image,omitempty"`
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed
	// +optional
	UpgradeStrategy v1alpha1.UpgradeStrategy `json:"upgradeStrategy"`

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
	// used to open additional ports that can't be inferred by the operator, like for custom receivers.
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
	// Ingress is used to specify how OpenTelemetry Collector is exposed. This
	// functionality is only available if one of the valid modes is set.
	// Valid modes are: deployment, daemonset and statefulset.
	// +optional
	Ingress v1alpha1.Ingress `json:"ingress,omitempty"`
	// HostNetwork indicates if the pod should run in the host networking namespace.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty"`
	// If specified, indicates the pod's priority.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// If specified, indicates the pod's scheduling constraints
	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`
	// Actions that the management system should take in response to container lifecycle events. Cannot be updated.
	// +optional
	Lifecycle *v1.Lifecycle `json:"lifecycle,omitempty"`
	// Duration in seconds the pod needs to terminate gracefully upon probe failure.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
	// Liveness config for the OpenTelemetry Collector except the probe handler which is auto generated from the health extension of the collector.
	// It is only effective when healthcheckextension is configured in the OpenTelemetry Collector pipeline.
	// +optional
	LivenessProbe *v1alpha1.Probe `json:"livenessProbe,omitempty"`
	// InitContainers allows injecting initContainers to the Collector's pod definition.
	// These init containers can be used to fetch secrets for injection into the
	// configuration from external sources, run added checks, etc. Any errors during the execution of
	// an initContainer will lead to a restart of the Pod. More info:
	// https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	// +optional
	InitContainers []v1.Container `json:"initContainers,omitempty"`

	// AdditionalContainers allows injecting additional containers into the Collector's pod definition.
	// These sidecar containers can be used for authentication proxies, log shipping sidecars, agents for shipping
	// metrics to their cloud, or in general sidecars that do not support automatic injection. This option only
	// applies to Deployment, DaemonSet, and StatefulSet deployment modes of the collector. It does not apply to the sidecar
	// deployment mode. More info about sidecars:
	// https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/
	//
	// Container names managed by the operator:
	// * `otc-container`
	//
	// Overriding containers managed by the operator is outside the scope of what the maintainers will support and by
	// doing so, you wil accept the risk of it breaking things.
	//
	// +optional
	AdditionalContainers []v1.Container `json:"additionalContainers,omitempty"`

	// ObservabilitySpec defines how telemetry data gets handled.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Observability"
	Observability v1alpha1.ObservabilitySpec `json:"observability,omitempty"`

	// TopologySpreadConstraints embedded kubernetes pod configuration option,
	// controls how pods are spread across your cluster among failure-domains
	// such as regions, zones, nodes, and other user-defined topology domains
	// https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/
	// This is only relevant to statefulset, and deployment mode
	// +optional
	TopologySpreadConstraints []v1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`

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
