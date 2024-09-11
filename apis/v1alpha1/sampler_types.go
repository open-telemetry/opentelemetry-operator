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
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

type (
	// RoutingType describes how traffic to be sampled is distributed.
	// +kubebuilder:validation:Enum=traceid;service
	SamplerRoutingType string
)

const (
	// SamplerRoutingTypeTraceID specifies that traffic gets distributed based on the provided traceid.
	SamplerRoutingTypeTraceID SamplerRoutingType = "traceid"
	// SamplerRoutingTypeService specifies that traffic gets distributed based on the provided service name.
	SamplerRoutingTypeService SamplerRoutingType = "service"
)

// SamplerSpec defines the desired state of Sampler.
type SamplerSpec struct {
	// Policies taken into account when making sampling decision.
	//
	// +optional
	// +kubebuilder:validation:Optional
	Policies []SamplerPolicySpec `json:"policies,omitempty"`
	// RoutingKey describes how traffic to be sampled is distributed.
	//
	// +kubebuilder:default:=traceid
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Routing Key"
	RoutingKey SamplerRoutingType `json:"routingKey,omitempty"`
	// DecisionWait defines the time since the first span of a trace before
	// making a sampling decision.
	// Default is 30s.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=30000000000
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Time frame to make a dicision"
	DecisionWait time.Duration `json:"decision_wait" yaml:"decision_wait"`
	// NumTraces defines the number of traces kept in memory.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=5000
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Number of traces kept in memory"
	NumTraces uint64 `json:"num_traces" yaml:"num_traces"`
	// ExpectedNewTracesPerSec defines  the expected number of new traces.
	// It helps in allocating data structure.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=5000
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Expected new traces per second"
	ExpectedNewTracesPerSec uint64 `json:"expected_new_traces_per_sec" yaml:"expected_new_traces_per_sec"`
	// DecisionCache configuration.
	//
	// +optional
	DecisionCache SamplerPolicyDecisionCacheSpec `json:"decision_cache" yaml:"decision_cache"`
	// Template defines requirements for a set of setup components.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Sampling Component Templates"
	Components SamplerTemplateSpec `json:"components,omitempty"`
	// Telemetry defines the telemetry settings for the sampling system.
	//
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Telemetry Settings"
	Telemetry SamplerTelemetrySpec `json:"telemetry,omitempty"`
	// Exporter defines the exporter configuration.
	// NOTE: currently only otlp exporter settings are supported.
	//
	// +requiered
	Exporter v1beta1.AnyConfig `json:"exporter" yaml:"exporter"`
}

// SamplerPolicySpec describes a specific sampling policy.
type SamplerPolicySpec struct {
	// Name of policy.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Policy Name"
	Name string `json:"name" yaml:"name"`
	// NumTraces defines the number of traces kept in memory.
	//
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Policy Type"
	Type SamplerType `json:"type" yaml:"type"`
	// Config of a specific policy that will  be applied.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors="urn:alm:descriptor:io.kubernetes:Secret",displayName="Policy Configuration"
	Config v1beta1.AnyConfig `json:"-" yaml:",inline"`
}

// SamplerPolicyDecisionCacheSpec defines the settings for the decision cache.
type SamplerPolicyDecisionCacheSpec struct {
	// SampledCacheSize configures the amount of trace IDs to be kept in an LRU
	// cache, persisting the "keep" decisions for traces that may have already
	// been released from memory.
	// By default, the size is 0 and the cache is inactive.
	// If using, configure this as much higher than num_traces so decisions for
	// trace IDs are kept longer than the span data for the trace.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Sampled Cache Size"
	SampledCacheSize uint64 `json:"sampled_cache_size" yaml:"sampled_cache_size"`
}

// SamplerTelemetrySpec defines the telemetry settings for the sampling system.
type SamplerTelemetrySpec struct {
	// TODO(@frzifus): provide telemetry settings
	// e.g.: serviceMonitor, spanMetrics ...
}

// SamplerTemplateSpec defines the template of all requirements to configure
// scheduling of all components to be deployed.
type SamplerTemplateSpec struct {
	// Loadbalancer defines the loadbalancer component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Loadbalancer pods"
	Loadbalancer SamplerComponentSpec `json:"loadbalancer,omitempty"`
	// Sampler defines the sampler component spec.
	//
	// +optional
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Sampler pods"
	Sampler SamplerComponentSpec `json:"sampler,omitempty"`
}

// SamplerComponentSpec defines specific schedule settings for sampler components.
type SamplerComponentSpec struct {
	// ManagementState defines if the CR should be managed by the operator or not.
	// Default is managed.
	//
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:default:=managed
	ManagementState ManagementStateType `json:"managementState,omitempty"`
	// Resources to set on the OpenTelemetry Collector pods.
	// +optional
	Resources v1.ResourceRequirements `json:"resources,omitempty"`
	// NodeSelector to schedule OpenTelemetry Collector pods.
	// This is only relevant to daemonset, statefulset, and deployment mode
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// If specified, indicates the pod's scheduling constraints
	// +optional
	Affinity *v1.Affinity `json:"affinity,omitempty"`
	// Toleration to schedule OpenTelemetry Collector pods.
	// This is only relevant to daemonset, statefulset, and deployment mode
	//
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	// Replicas is the number of pod instances for the underlying OpenTelemetry Collector. Set this if your are not using autoscaling
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Autoscaler specifies the pod autoscaling configuration to use
	// for the OpenTelemetryCollector workload.
	//
	// +optional
	Autoscaler *AutoscalerSpec `json:"autoscaler,omitempty"`
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
	//
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// ServiceAccount indicates the name of an existing service account to use with this instance. When set,
	// the operator will not automatically create a ServiceAccount for the collector.
	//
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// SamplerStatus defines the observed state of Sampler.
type SamplerStatus struct {
	// TODO(@frzifus): add status fields.
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Sampler is the Schema for the samplers API.
type Sampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SamplerSpec   `json:"spec,omitempty"`
	Status SamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SamplerList contains a list of Sampler.
type SamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SamplerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Sampler{}, &SamplerList{})
}
