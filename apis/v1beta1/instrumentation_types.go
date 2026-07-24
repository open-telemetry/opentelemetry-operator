// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
)

// InstrumentationSpec defines the desired state of OpenTelemetry SDK and instrumentation.
type InstrumentationSpec struct {
	// EnvConfig defines the SDK configuration via environment variables.
	// This is the same configuration model as v1alpha1 (exporter, sampler, propagators).
	// +optional
	EnvConfig *EnvConfig `json:"envConfig,omitempty"`

	// Resource defines operator-level resource attribute configuration.
	// These settings control how the operator populates resource attributes.
	// +optional
	Resource Resource `json:"resource,omitempty"`

	// Env defines common env vars.
	// Precedence: original container env > language-specific env > common env > SDK config.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Java defines configuration for Java auto-instrumentation.
	// +optional
	Java Java `json:"java,omitempty"`

	// NodeJS defines configuration for NodeJS auto-instrumentation.
	// +optional
	NodeJS NodeJS `json:"nodejs,omitempty"`

	// Python defines configuration for Python auto-instrumentation.
	// +optional
	Python Python `json:"python,omitempty"`

	// DotNet defines configuration for DotNet auto-instrumentation.
	// +optional
	DotNet DotNet `json:"dotnet,omitempty"`

	// Go defines configuration for Go auto-instrumentation.
	// +optional
	Go Go `json:"go,omitempty"`

	// ApacheHttpd defines configuration for Apache HTTPD auto-instrumentation.
	// +optional
	ApacheHttpd ApacheHttpd `json:"apacheHttpd,omitempty"`

	// Nginx defines configuration for Nginx auto-instrumentation.
	// +optional
	Nginx Nginx `json:"nginx,omitempty"`

	// ImagePullPolicy defines the image pull policy for init containers.
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// InitContainerSecurityContext applied to auto-instrumentation init containers.
	// +optional
	InitContainerSecurityContext *corev1.SecurityContext `json:"initContainerSecurityContext,omitempty"`
}

// EnvConfig defines the env-var-based SDK configuration.
type EnvConfig struct {
	// Exporter defines exporter configuration.
	// +optional
	Exporter Exporter `json:"exporter,omitempty"`

	// Propagators defines inter-process context propagation configuration.
	// +optional
	Propagators []Propagator `json:"propagators,omitempty"`

	// Sampler defines sampling configuration.
	// +optional
	Sampler Sampler `json:"sampler,omitempty"`
}

// Exporter defines OTLP exporter configuration.
type Exporter struct {
	// Endpoint is address of the collector with OTLP endpoint.
	// If the endpoint defines https:// scheme TLS has to be specified.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Protocol defines the OTLP protocol to use.
	// Valid values are grpc, http/protobuf, and http/json.
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// TLS defines certificates for TLS.
	// TLS needs to be enabled by specifying https:// scheme in the Endpoint.
	// +optional
	TLS *TLS `json:"tls,omitempty"`
}

// TLS defines TLS configuration for exporter.
type TLS struct {
	// SecretName defines secret name that will be used to configure TLS on the exporter.
	// It is user responsibility to create the secret in the namespace of the workload.
	// The secret must contain client certificate (Cert) and private key (Key).
	// The CA certificate might be defined in the secret or in the config map.
	SecretName string `json:"secretName,omitempty"`

	// ConfigMapName defines configmap name with CA certificate. If it is not defined CA certificate will be
	// used from the secret defined in SecretName.
	ConfigMapName string `json:"configMapName,omitempty"`

	// CA defines the key of certificate (e.g. ca.crt) in the configmap map, secret or absolute path to a certificate.
	// The absolute path can be used when certificate is already present on the workload filesystem e.g.
	// /var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt
	CA string `json:"ca_file,omitempty"`
	// Cert defines the key (e.g. tls.crt) of the client certificate in the secret or absolute path to a certificate.
	// The absolute path can be used when certificate is already present on the workload filesystem.
	Cert string `json:"cert_file,omitempty"`
	// Key defines a key (e.g. tls.key) of the private key in the secret or absolute path to a certificate.
	// The absolute path can be used when certificate is already present on the workload filesystem.
	Key string `json:"key_file,omitempty"`
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

// Resource defines operator-level resource attribute configuration.
// These fields control how the operator populates resource attributes and
// are independent of the SDK configuration mode.
type Resource struct {
	// Attributes defines resource attributes to inject into the workload.
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// K8sMetadata controls K8s resource attribute injection (k8s.pod.name, k8s.namespace.name, etc.).
	// +optional
	K8sMetadata *K8sMetadataConfig `json:"k8sMetadata,omitempty"`

	// ServiceMetadata controls service identity attribute derivation (service.name, service.version, etc.).
	// +optional
	ServiceMetadata *ServiceMetadataConfig `json:"serviceMetadata,omitempty"`
}

// K8sMetadataConfig defines how Kubernetes resource attributes are injected.
type K8sMetadataConfig struct {
	// Enabled controls whether K8s resource attributes are automatically injected.
	// When false, no k8s.* attributes are added. Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// IncludeUIDs defines whether K8s UID attributes should be collected
	// (e.g. k8s.deployment.uid, k8s.replicaset.uid). Only applies when Enabled is true.
	// +optional
	IncludeUIDs bool `json:"includeUIDs,omitempty"`
}

// ServiceMetadataConfig defines how service identity attributes are derived from K8s metadata.
// Controls attributes: service.name, service.version, service.namespace, service.instance.id.
type ServiceMetadataConfig struct {
	// Enabled controls whether service identity attributes are automatically derived.
	// When false, no service.* attributes are added by the operator. Defaults to true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// CommonLanguageSpec contains fields shared by all language-specific configurations.
type CommonLanguageSpec struct {
	// Image is a container image with auto-instrumentation.
	// +optional
	Image string `json:"image,omitempty"`

	// VolumeClaimTemplate defines an ephemeral volume used for auto-instrumentation.
	// If omitted, an emptyDir is used with a default size limit.
	// +optional
	VolumeClaimTemplate corev1.PersistentVolumeClaimTemplate `json:"volumeClaimTemplate,omitempty"`

	// Env defines language-specific env vars.
	// Precedence: original container env > language-specific env > common env > SDK config.
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources describes the compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Java defines Java SDK and instrumentation configuration.
type Java struct {
	CommonLanguageSpec `json:",inline"`

	// Extensions defines java specific extensions.
	// All extensions are copied to a single directory; if a JAR with the same name exists, it will be overwritten.
	// +optional
	Extensions []Extensions `json:"extensions,omitempty"`
}

// Extensions defines a container image and directory for Java instrumentation extensions.
type Extensions struct {
	// Image is a container image with extensions auto-instrumentation JAR.
	Image string `json:"image"`

	// Dir is a directory with extensions auto-instrumentation JAR.
	Dir string `json:"dir"`
}

// NodeJS defines NodeJS SDK and instrumentation configuration.
type NodeJS struct {
	CommonLanguageSpec `json:",inline"`
}

// Python defines Python SDK and instrumentation configuration.
type Python struct {
	CommonLanguageSpec `json:",inline"`
}

// DotNet defines DotNet SDK and instrumentation configuration.
type DotNet struct {
	CommonLanguageSpec `json:",inline"`
}

// Go defines Go SDK and instrumentation configuration.
type Go struct {
	CommonLanguageSpec `json:",inline"`

	// SecurityContext applied to the Go auto-instrumentation sidecar.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
}

// ApacheHttpd defines Apache HTTPD SDK and instrumentation configuration.
type ApacheHttpd struct {
	CommonLanguageSpec `json:",inline"`

	// Attrs defines Apache HTTPD agent specific attributes. The precedence is:
	// `agent default attributes` > `instrument spec attributes` .
	// Attributes are documented at https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module
	// +optional
	Attrs []corev1.EnvVar `json:"attrs,omitempty"`

	// Version is the Apache HTTPD server version. One of 2.4 or 2.2. Default is 2.4.
	// +optional
	Version string `json:"version,omitempty"`

	// ConfigPath is the location of Apache HTTPD server configuration.
	// Needed only if different from default "/usr/local/apache2/conf".
	// +optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9._/-]*$`
	// +kubebuilder:validation:MaxLength=256
	ConfigPath string `json:"configPath,omitempty"`
}

// Nginx defines Nginx SDK and instrumentation configuration.
type Nginx struct {
	CommonLanguageSpec `json:",inline"`

	// Attrs defines Nginx agent specific attributes. The precedence order is:
	// `agent default attributes` > `instrument spec attributes` .
	// Attributes are documented at https://github.com/open-telemetry/opentelemetry-cpp-contrib/tree/main/instrumentation/otel-webserver-module
	// +optional
	Attrs []corev1.EnvVar `json:"attrs,omitempty"`

	// ConfigFile is the location of Nginx configuration file.
	// Needed only if different from default "/etc/nginx/nginx.conf".
	// +optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9._/-]*$`
	// +kubebuilder:validation:MaxLength=256
	ConfigFile string `json:"configFile,omitempty"`
}

// InstrumentationStatus defines status of the instrumentation.
type InstrumentationStatus struct {
	// UpgradeBlockedVersions contains instrumentation language images whose
	// versions could not be automatically upgraded, mapped to a message
	// explaining why.
	// +optional
	UpgradeBlockedVersions map[string]string `json:"upgradeBlockedVersions,omitempty"`
}
