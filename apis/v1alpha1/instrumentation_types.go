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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstrumentationSpec defines the desired state of OpenTelemetry SDK and instrumentation.
type InstrumentationSpec struct {
	// Exporter defines exporter configuration.
	// +optional
	Exporter `json:"exporter,omitempty"`

	// Resource defines the configuration for the resource attributes, as defined by the OpenTelemetry specification.
	// +optional
	Resource Resource `json:"resource,omitempty"`

	// Propagators defines inter-process context propagation configuration.
	// Values in this list will be set in the OTEL_PROPAGATORS env var.
	// Enum=tracecontext;baggage;b3;b3multi;jaeger;xray;ottrace;none
	// +optional
	Propagators []Propagator `json:"propagators,omitempty"`

	// Sampler defines sampling configuration.
	// +optional
	Sampler `json:"sampler,omitempty"`

	// Env defines common env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Java defines configuration for java auto-instrumentation.
	// +optional
	Java Java `json:"java,omitempty"`

	// NodeJS defines configuration for nodejs auto-instrumentation.
	// +optional
	NodeJS NodeJS `json:"nodejs,omitempty"`

	// Python defines configuration for python auto-instrumentation.
	// +optional
	Python Python `json:"python,omitempty"`

	// DotNet defines configuration for DotNet auto-instrumentation.
	// +optional
	DotNet DotNet `json:"dotnet,omitempty"`

	// Go defines configuration for Go auto-instrumentation.
	// When using Go auto-instrumentation you must provide a value for the OTEL_GO_AUTO_TARGET_EXE env var via the
	// Instrumentation env vars or via the instrumentation.opentelemetry.io/otel-go-auto-target-exe pod annotation.
	// Failure to set this value causes instrumentation injection to abort, leaving the original pod unchanged.
	// +optional
	Go Go `json:"go,omitempty"`

	// ApacheHttpd defines configuration for Apache HTTPD auto-instrumentation.
	// +optional
	ApacheHttpd ApacheHttpd `json:"apacheHttpd,omitempty"`

	// Nginx defines configuration for Nginx auto-instrumentation.
	// +optional
	Nginx Nginx `json:"nginx,omitempty"`
}

// Resource defines the configuration for the resource attributes, as defined by the OpenTelemetry specification.
// See also: https://github.com/open-telemetry/opentelemetry-specification/blob/v1.8.0/specification/overview.md#resources
type Resource struct {
	// Attributes defines attributes that are added to the resource.
	// For example environment: dev
	// +optional
	Attributes map[string]string `json:"resourceAttributes,omitempty"`

	// AddK8sUIDAttributes defines whether K8s UID attributes should be collected (e.g. k8s.deployment.uid).
	// +optional
	AddK8sUIDAttributes bool `json:"addK8sUIDAttributes,omitempty"`
}

// Exporter defines OTLP exporter configuration.
type Exporter struct {
	// Endpoint is address of the collector with OTLP endpoint.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}

// Sampler defines sampling configuration.
type Sampler struct {
	// Type defines sampler type.
	// The value will be set in the OTEL_TRACES_SAMPLER env var.
	// The value can be for instance parentbased_always_on, parentbased_always_off, parentbased_traceidratio...
	// +optional
	Type SamplerType `json:"type,omitempty"`

	// Argument defines sampler argument.
	// The value depends on the sampler type.
	// For instance for parentbased_traceidratio sampler type it is a number in range [0..1] e.g. 0.25.
	// The value will be set in the OTEL_TRACES_SAMPLER_ARG env var.
	// +optional
	Argument string `json:"argument,omitempty"`
}

// Java defines Java SDK and instrumentation configuration.
type Java struct {
	// Image is a container image with javaagent auto-instrumentation JAR.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines java specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// NodeJS defines NodeJS SDK and instrumentation configuration.
type NodeJS struct {
	// Image is a container image with NodeJS SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines nodejs specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// Python defines Python SDK and instrumentation configuration.
type Python struct {
	// Image is a container image with Python SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines python specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// DotNet defines DotNet SDK and instrumentation configuration.
type DotNet struct {
	// Image is a container image with DotNet SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines DotNet specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

type Go struct {
	// Image is a container image with Go SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines Go specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// ApacheHttpd defines Apache SDK and instrumentation configuration.
type ApacheHttpd struct {
	// Image is a container image with Apache SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines Apache HTTPD specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Attrs defines Apache HTTPD agent specific attributes. The precedence is:
	// `agent default attributes` > `instrument spec attributes` .
	// Attributes are documented at https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module
	// +optional
	Attrs []corev1.EnvVar `json:"attrs,omitempty"`

	// Apache HTTPD server version. One of 2.4 or 2.2. Default is 2.4
	// +optional
	Version string `json:"version,omitempty"`

	// Location of Apache HTTPD server configuration.
	// Needed only if different from default "/usr/local/apache2/conf"
	// +optional
	ConfigPath string `json:"configPath,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// Nginx defines Nginx SDK and instrumentation configuration.
type Nginx struct {
	// Image is a container image with Nginx SDK and auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeSizeLimit defines size limit for volume used for auto-instrumentation.
	// The default size is 200Mi.
	VolumeSizeLimit *resource.Quantity `json:"volumeLimitSize,omitempty"`

	// Env defines Nginx specific env vars. There are four layers for env vars' definitions and
	// the precedence order is: `original container env vars` > `language specific env vars` > `common env vars` > `instrument spec configs' vars`.
	// If the former var had been defined, then the other vars would be ignored.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Attrs defines Nginx agent specific attributes. The precedence order is:
	// `agent default attributes` > `instrument spec attributes` .
	// Attributes are documented at https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module
	// +optional
	Attrs []corev1.EnvVar `json:"attrs,omitempty"`

	// Location of Nginx configuration file.
	// Needed only if different from default "/etx/nginx/nginx.conf"
	// +optional
	ConfigFile string `json:"configFile,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resourceRequirements,omitempty"`
}

// InstrumentationStatus defines status of the instrumentation.
type InstrumentationStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelinst;otelinsts
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.exporter.endpoint"
// +kubebuilder:printcolumn:name="Sampler",type="string",JSONPath=".spec.sampler.type"
// +kubebuilder:printcolumn:name="Sampler Arg",type="string",JSONPath=".spec.sampler.argument"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Instrumentation"
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1}}

// Instrumentation is the spec for OpenTelemetry instrumentation.
type Instrumentation struct {
	Status            InstrumentationStatus `json:"status,omitempty"`
	metav1.TypeMeta   `json:",inline"`
	Spec              InstrumentationSpec `json:"spec,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
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
