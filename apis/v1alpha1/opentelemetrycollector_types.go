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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Ingress is used to specify how OpenTelemetry Collector is exposed. This
// functionality is only available if one of the valid modes is set.
// Valid modes are: deployment, daemonset and statefulset.
// NOTE: If this feature is activated, all specified receivers are exposed.
// Currently this has a few limitations. Depending on the ingress controller
// there are problems with TLS and gRPC.
// SEE: https://github.com/open-telemetry/opentelemetry-operator/issues/1306.
// NOTE: As a workaround, port name and appProtocol could be specified directly
// in the CR.
// SEE: OpenTelemetryCollector.spec.ports[index].
type Ingress struct {
	// Type default value is: ""
	// Supported types are: ingress
	Type IngressType `json:"type,omitempty"`

	// Hostname by which the ingress proxy can be reached.
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// Annotations to add to ingress.
	// e.g. 'cert-manager.io/cluster-issuer: "letsencrypt"'
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS configuration.
	// +optional
	TLS []networkingv1.IngressTLS `json:"tls,omitempty"`

	// IngressClassName is the name of an IngressClass cluster resource. Ingress
	// controller implementations use this field to know whether they should be
	// serving this Ingress resource.
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// Route is an OpenShift specific section that is only considered when
	// type "route" is used.
	// +optional
	Route OpenShiftRoute `json:"route,omitempty"`
}

// OpenShiftRoute defines openshift route specific settings.
type OpenShiftRoute struct {
	// Termination indicates termination type. By default "edge" is used.
	Termination TLSRouteTerminationType `json:"termination,omitempty"`
}

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
	// Deprecated: use "OpenTelemetryCollector.Spec.Autoscaler.MinReplicas" instead.
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// MaxReplicas sets an upper bound to the autoscaling feature. If MaxReplicas is set autoscaling is enabled.
	// +optional
	// Deprecated: use "OpenTelemetryCollector.Spec.Autoscaler.MaxReplicas" instead.
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// Autoscaler specifies the pod autoscaling configuration to use
	// for the OpenTelemetryCollector workload.
	//
	// +optional
	Autoscaler *AutoscalerSpec `json:"autoscaler,omitempty"`
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
	// ServiceAccount indicates the name of an existing service account to use with this instance. When set,
	// the operator will not automatically create a ServiceAccount for the collector.
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
	Ingress Ingress `json:"ingress,omitempty"`
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
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`
}

// OpenTelemetryTargetAllocator defines the configurations for the Prometheus target allocator.
type OpenTelemetryTargetAllocator struct {
	// Replicas is the number of pod instances for the underlying TargetAllocator. This should only be set to a value
	// other than 1 if a strategy that allows for high availability is chosen. Currently, the only allocation strategy
	// that can be run in a high availability mode is consistent-hashing.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Resources to set on the OpenTelemetryTargetAllocator containers.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// AllocationStrategy determines which strategy the target allocator should use for allocation.
	// The current options are least-weighted and consistent-hashing. The default option is least-weighted
	// +optional
	AllocationStrategy OpenTelemetryTargetAllocatorAllocationStrategy `json:"allocationStrategy,omitempty"`
	// FilterStrategy determines how to filter targets before allocating them among the collectors.
	// The only current option is relabel-config (drops targets based on prom relabel_config).
	// Filtering is disabled by default.
	// +optional
	FilterStrategy string `json:"filterStrategy,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance. When set,
	// the operator will not automatically create a ServiceAccount for the TargetAllocator.
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
	// PodMonitors to be selected for target discovery.
	// This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a
	// PodMonitor's meta labels. The requirements are ANDed.
	// +optional
	PodMonitorSelector map[string]string `json:"podMonitorSelector,omitempty"`
	// ServiceMonitors to be selected for target discovery.
	// This is a map of {key,value} pairs. Each {key,value} in the map is going to exactly match a label in a
	// ServiceMonitor's meta labels. The requirements are ANDed.
	// +optional
	ServiceMonitorSelector map[string]string `json:"serviceMonitorSelector,omitempty"`
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

// AutoscalerSpec defines the OpenTelemetryCollector's pod autoscaling specification.
type AutoscalerSpec struct {
	// MinReplicas sets a lower bound to the autoscaling feature.  Set this if your are using autoscaling. It must be at least 1
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// MaxReplicas sets an upper bound to the autoscaling feature. If MaxReplicas is set autoscaling is enabled.
	// +optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	// +optional
	Behavior *autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
	// Metrics is meant to provide a customizable way to configure HPA metrics.
	// currently the only supported custom metrics is type=Pod.
	// Use TargetCPUUtilization or TargetMemoryUtilization instead if scaling on these common resource metrics.
	// +optional
	Metrics []MetricSpec `json:"metrics,omitempty"`
	// TargetCPUUtilization sets the target average CPU used across all replicas.
	// If average CPU exceeds this value, the HPA will scale up. Defaults to 90 percent.
	// +optional
	TargetCPUUtilization *int32 `json:"targetCPUUtilization,omitempty"`
	// +optional
	// TargetMemoryUtilization sets the target average memory utilization across all replicas
	TargetMemoryUtilization *int32 `json:"targetMemoryUtilization,omitempty"`
}

// Probe defines the OpenTelemetry's pod probe config. Only Liveness probe is supported currently.
type Probe struct {
	// Number of seconds after the container has started before liveness probes are initiated.
	// Defaults to 0 seconds. Minimum value is 0.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`
	// Number of seconds after which the probe times out.
	// Defaults to 1 second. Minimum value is 1.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
	// How often (in seconds) to perform the probe.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	// Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.
	// +optional
	SuccessThreshold *int32 `json:"successThreshold,omitempty"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	// Defaults to 3. Minimum value is 1.
	// +optional
	FailureThreshold *int32 `json:"failureThreshold,omitempty"`
	// Optional duration in seconds the pod needs to terminate gracefully upon probe failure.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this
	// value overrides the value provided by the pod spec.
	// Value must be non-negative integer. The value zero indicates stop immediately via
	// the kill signal (no opportunity to shut down).
	// This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.
	// Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`
}

// MetricSpec defines a subset of metrics to be defined for the HPA's metric array
// more metric type can be supported as needed.
// See https://pkg.go.dev/k8s.io/api/autoscaling/v2#MetricSpec for reference.
type MetricSpec struct {
	Type autoscalingv2.MetricSourceType  `json:"type"`
	Pods *autoscalingv2.PodsMetricSource `json:"pods,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollector{}, &OpenTelemetryCollectorList{})
}
