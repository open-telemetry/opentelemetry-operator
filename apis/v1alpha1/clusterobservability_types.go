// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OTLPHTTPExporter defines OTLP HTTP exporter configuration.
// This structure mirrors the official OpenTelemetry Collector otlphttpexporter configuration.
type OTLPHTTPExporter struct {
	// Endpoint is the target base URL to send data to (e.g., https://example.com:4318).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// TracesEndpoint is the target URL to send trace data to (e.g., https://example.com:4318/v1/traces).
	// If this setting is present the endpoint setting is ignored for traces.
	// +optional
	TracesEndpoint string `json:"traces_endpoint,omitempty"`

	// MetricsEndpoint is the target URL to send metric data to (e.g., https://example.com:4318/v1/metrics).
	// If this setting is present the endpoint setting is ignored for metrics.
	// +optional
	MetricsEndpoint string `json:"metrics_endpoint,omitempty"`

	// LogsEndpoint is the target URL to send log data to (e.g., https://example.com:4318/v1/logs).
	// If this setting is present the endpoint setting is ignored for logs.
	// +optional
	LogsEndpoint string `json:"logs_endpoint,omitempty"`

	// ProfilesEndpoint is the target URL to send profile data to (e.g., https://example.com:4318/v1/development/profiles).
	// If this setting is present the endpoint setting is ignored for profiles.
	// +optional
	ProfilesEndpoint string `json:"profiles_endpoint,omitempty"`

	// TLS defines TLS configuration for the exporter.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`

	// Timeout is the HTTP request time limit (e.g., "30s", "1m"). Default is 30s.
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// ReadBufferSize for HTTP client. Default is 0.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ReadBufferSize *int `json:"read_buffer_size,omitempty"`

	// WriteBufferSize for HTTP client. Default is 512 * 1024.
	// +optional
	// +kubebuilder:validation:Minimum=0
	WriteBufferSize *int `json:"write_buffer_size,omitempty"`

	// SendingQueue defines configuration for the sending queue.
	// +optional
	SendingQueue *SendingQueueConfig `json:"sending_queue,omitempty"`

	// RetryOnFailure defines retry configuration for failed requests.
	// +optional
	RetryOnFailure *RetryConfig `json:"retry_on_failure,omitempty"`

	// Encoding defines the encoding to use for the messages.
	// Valid options: proto, json. Default is proto.
	// +optional
	// +kubebuilder:validation:Enum=proto;json
	Encoding string `json:"encoding,omitempty"`

	// Compression defines the compression algorithm to use.
	// By default gzip compression is enabled. Use "none" to disable.
	// +optional
	// +kubebuilder:validation:Enum=gzip;none;""
	Compression string `json:"compression,omitempty"`

	// Headers defines additional headers to be sent with each request.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`
}

// TLSConfig defines TLS configuration for the OTLP HTTP exporter.
// This mirrors the OpenTelemetry Collector configtls settings.
type TLSConfig struct {
	// CAFile is the path to the CA certificate file for server verification.
	// +optional
	CAFile string `json:"ca_file,omitempty"`

	// CertFile is the path to the client certificate file for mutual TLS.
	// +optional
	CertFile string `json:"cert_file,omitempty"`

	// KeyFile is the path to the client private key file for mutual TLS.
	// +optional
	KeyFile string `json:"key_file,omitempty"`

	// Insecure controls whether to use insecure transport. Default is false.
	// +optional
	Insecure bool `json:"insecure,omitempty"`

	// ServerName for TLS handshake. If empty, uses the hostname from endpoint.
	// +optional
	ServerName string `json:"server_name,omitempty"`
}

// SendingQueueConfig defines configuration for the sending queue.
type SendingQueueConfig struct {
	// Enabled controls whether the queue is enabled. Default is true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// NumConsumers is the number of consumers that dequeue batches. Default is 10.
	// +optional
	// +kubebuilder:validation:Minimum=1
	NumConsumers *int `json:"num_consumers,omitempty"`

	// QueueSize is the maximum number of batches allowed in queue at a given time. Default is 1000.
	// +optional
	// +kubebuilder:validation:Minimum=1
	QueueSize *int `json:"queue_size,omitempty"`
}

// RetryConfig defines retry configuration for failed requests.
type RetryConfig struct {
	// Enabled controls whether retry is enabled. Default is true.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// InitialInterval is the initial retry interval (e.g., "5s"). Default is 5s.
	// +optional
	InitialInterval string `json:"initial_interval,omitempty"`

	// RandomizationFactor is the randomization factor for retry intervals (e.g., "0.5"). Default is 0.5.
	// +optional
	RandomizationFactor string `json:"randomization_factor,omitempty"`

	// Multiplier is the multiplier for retry intervals (e.g., "1.5"). Default is 1.5.
	// +optional
	Multiplier string `json:"multiplier,omitempty"`

	// MaxInterval is the maximum retry interval (e.g., "30s"). Default is 30s.
	// +optional
	MaxInterval string `json:"max_interval,omitempty"`

	// MaxElapsedTime is the maximum elapsed time for retries (e.g., "5m"). Default is 5m.
	// +optional
	MaxElapsedTime string `json:"max_elapsed_time,omitempty"`
}

// ClusterObservabilitySpec defines the desired state of ClusterObservability.
// This follows a simplified design using a single OTLP HTTP exporter for all signals.
type ClusterObservabilitySpec struct {
	// Exporter defines the OTLP HTTP exporter configuration for all signals.
	// The collector will automatically append appropriate paths for each signal type.
	// +required
	Exporter OTLPHTTPExporter `json:"exporter"`
}

// ClusterObservabilityConditionType represents the type of condition.
type ClusterObservabilityConditionType string

const (
	// ClusterObservabilityConditionReady indicates whether the ClusterObservability is ready.
	ClusterObservabilityConditionReady ClusterObservabilityConditionType = "Ready"
	// ClusterObservabilityConditionConfigured indicates whether the ClusterObservability is configured.
	ClusterObservabilityConditionConfigured ClusterObservabilityConditionType = "Configured"
	// ClusterObservabilityConditionConflicted indicates that multiple ClusterObservability resources exist.
	ClusterObservabilityConditionConflicted ClusterObservabilityConditionType = "Conflicted"
)

const (
	// ClusterObservabilityFinalizer is the finalizer used for ClusterObservability resources.
	ClusterObservabilityFinalizer = "clusterobservability.opentelemetry.io/finalizer"
)

// ClusterObservabilityCondition represents a condition of a ClusterObservability.
type ClusterObservabilityCondition struct {
	// Type of condition.
	// +required
	Type ClusterObservabilityConditionType `json:"type"`

	// Status of the condition.
	// +required
	Status metav1.ConditionStatus `json:"status"`

	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ClusterObservabilityStatus defines the observed state of ClusterObservability.
type ClusterObservabilityStatus struct {
	// Conditions represent the latest available observations of the ClusterObservability state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []ClusterObservabilityCondition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed for this ClusterObservability.
	// It corresponds to the ClusterObservability's generation, which is updated on mutation
	// by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Phase represents the current phase of the ClusterObservability.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Message provides additional information about the current state.
	// +optional
	Message string `json:"message,omitempty"`

	// ComponentsStatus provides status information about individual observability components.
	// +optional
	ComponentsStatus map[string]ComponentStatus `json:"componentsStatus,omitempty"`

	// ConfigVersions tracks the version hashes of the configuration files used.
	// This enables detection of config changes when operator is upgraded.
	// +optional
	ConfigVersions map[string]string `json:"configVersions,omitempty"`
}

// ComponentStatus represents the status of an individual component.
type ComponentStatus struct {
	// Ready indicates whether the component is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Message provides additional information about the component status.
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdated is the last time this status was updated.
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.exporter.endpoint",description="OTLP exporter endpoint"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:displayName="Cluster Observability"
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1},{Deployment,apps/v1},{ConfigMap,v1},{Service,v1},{DaemonSet,apps/v1}}

// ClusterObservability is the Schema for the clusterobservabilities API.
type ClusterObservability struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterObservabilitySpec   `json:"spec,omitempty"`
	Status ClusterObservabilityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterObservabilityList contains a list of ClusterObservability.
type ClusterObservabilityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterObservability `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterObservability{}, &ClusterObservabilityList{})
}
