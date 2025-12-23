// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Instrumentation{}, &InstrumentationList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=otelinst;otelinsts
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Disabled",type="boolean",JSONPath=".spec.config.disabled"
// +operator-sdk:csv:customresourcedefinitions:displayName="OpenTelemetry Instrumentation"
// +operator-sdk:csv:customresourcedefinitions:resources={{Pod,v1}}

// Instrumentation is the Schema for the instrumentations API.
type Instrumentation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationSpec   `json:"spec,omitempty"`
	Status InstrumentationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstrumentationList contains a list of Instrumentation.
type InstrumentationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instrumentation `json:"items"`
}

// InstrumentationSpec defines the desired state of OpenTelemetry SDK configuration.
type InstrumentationSpec struct {
	// Config defines the OpenTelemetry SDK configuration based on the OpenTelemetry Configuration Schema.
	// See: https://github.com/open-telemetry/opentelemetry-configuration
	// +required
	// +kubebuilder:validation:Required
	Config SDKConfig `json:"config"`
}

// InstrumentationStatus defines the observed state of Instrumentation.
type InstrumentationStatus struct {
	// Conditions represent the latest available observations of an object's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SDKConfig represents the OpenTelemetry SDK configuration based on the OpenTelemetry Configuration Schema.
// Only stable fields are included (no experimental/development fields).
type SDKConfig struct {
	// FileFormat is the file format version. Represented as a string including the semver major and minor version numbers.
	// +required
	// +kubebuilder:validation:Required
	FileFormat string `json:"file_format"`

	// Disabled configures if the SDK is disabled or not. If omitted or null, false is used.
	// +optional
	Disabled *bool `json:"disabled,omitempty"`

	// AttributeLimits configures general attribute limits. See also tracer_provider.limits, logger_provider.limits.
	// +optional
	AttributeLimits *AttributeLimits `json:"attribute_limits,omitempty"`

	// Resource configures resource for all signals. If omitted, the default resource is used.
	// +optional
	Resource *Resource `json:"resource,omitempty"`

	// Propagator configures text map context propagators. If omitted, a noop propagator is used.
	// +optional
	Propagator *Propagator `json:"propagator,omitempty"`

	// TracerProvider configures the tracer provider. If omitted, a noop tracer provider is used.
	// +optional
	TracerProvider *TracerProvider `json:"tracer_provider,omitempty"`

	// MeterProvider configures the meter provider. If omitted, a noop meter provider is used.
	// +optional
	MeterProvider *MeterProvider `json:"meter_provider,omitempty"`

	// LoggerProvider configures the logger provider. If omitted, a noop logger provider is used.
	// +optional
	LoggerProvider *LoggerProvider `json:"logger_provider,omitempty"`
}

// ============================================
// Attribute Limits Types
// ============================================

// AttributeLimits configures general attribute limits.
type AttributeLimits struct {
	// AttributeValueLengthLimit configures max attribute value size. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty"`

	// AttributeCountLimit configures max attribute count. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty"`
}

// ============================================
// Resource Types
// ============================================

// Resource configures resource for all signals.
type Resource struct {
	// Attributes configures resource attributes. Entries have higher priority than entries from .resource.attributes_list.
	// +optional
	Attributes []AttributeNameValue `json:"attributes,omitempty"`

	// AttributesList is a string containing a comma-separated list of key=value pairs.
	// Entries have lower priority than entries from .resource.attributes.
	// +optional
	AttributesList *string `json:"attributes_list,omitempty"`

	// SchemaURL configures resource schema URL. If omitted or null, no schema URL is used.
	// +optional
	SchemaURL *string `json:"schema_url,omitempty"`

	// Detectors configures resource detectors.
	// +optional
	Detectors *Detectors `json:"detectors,omitempty"`
}

// AttributeNameValue represents a single attribute with name, type, and value.
type AttributeNameValue struct {
	// Name is the attribute key.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Value is the attribute value. Can be a string, number, boolean, or array.
	// +required
	// +kubebuilder:validation:Required
	Value apiextensionsv1.JSON `json:"value"`

	// Type specifies the attribute value type. Valid values are: string, bool, int, double, string_array, bool_array, int_array, double_array.
	// +optional
	// +kubebuilder:validation:Enum=string;bool;int;double;string_array;bool_array;int_array;double_array
	Type *string `json:"type,omitempty"`
}

// Detectors defines resource detector configuration.
type Detectors struct {
	// Attributes specifies which attributes to include or exclude from detectors.
	// +optional
	Attributes *DetectorAttributes `json:"attributes,omitempty"`
}

// DetectorAttributes specifies attribute inclusion/exclusion for detectors.
type DetectorAttributes struct {
	// Included lists the attributes to include.
	// +optional
	Included []string `json:"included,omitempty"`

	// Excluded lists the attributes to exclude.
	// +optional
	Excluded []string `json:"excluded,omitempty"`
}

// ============================================
// Propagator Types
// ============================================

// Propagator defines the context propagation configuration.
type Propagator struct {
	// Composite defines the list of propagators to use.
	// Valid values include: tracecontext, baggage, b3, b3multi, jaeger, xray, ottrace.
	// +optional
	Composite []TextMapPropagator `json:"composite,omitempty"`
}

// TextMapPropagator defines the configuration for a text map propagator.
// Only one propagator type should be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.tracecontext) ? 1 : 0) + (has(self.baggage) ? 1 : 0) + (has(self.b3) ? 1 : 0) + (has(self.b3multi) ? 1 : 0) <= 1",message="only one propagator type should be specified"
// +kubebuilder:validation:XValidation:rule="(has(self.jaeger) ? 1 : 0) + (has(self.ottrace) ? 1 : 0) + (has(self.xray) ? 1 : 0) <= 1",message="only one propagator type should be specified"
// +kubebuilder:validation:XValidation:rule="!((has(self.tracecontext) || has(self.baggage) || has(self.b3) || has(self.b3multi)) && (has(self.jaeger) || has(self.ottrace) || has(self.xray)))",message="only one propagator type should be specified"
type TextMapPropagator struct {
	// TraceContext configures the tracecontext propagator. If omitted, ignore.
	// +optional
	TraceContext *TraceContextPropagator `json:"tracecontext,omitempty"`

	// Baggage configures the baggage propagator. If omitted, ignore.
	// +optional
	Baggage *BaggagePropagator `json:"baggage,omitempty"`

	// B3 configures the b3 propagator. If omitted, ignore.
	// +optional
	B3 *B3Propagator `json:"b3,omitempty"`

	// B3Multi configures the b3multi propagator. If omitted, ignore.
	// +optional
	B3Multi *B3MultiPropagator `json:"b3multi,omitempty"`

	// Jaeger configures the jaeger propagator. If omitted, ignore.
	// +optional
	Jaeger *JaegerPropagator `json:"jaeger,omitempty"`

	// OTTrace configures the ottrace propagator. If omitted, ignore.
	// +optional
	OTTrace *OTTracePropagator `json:"ottrace,omitempty"`

	// XRay configures the xray propagator. If omitted, ignore.
	// +optional
	XRay *XRayPropagator `json:"xray,omitempty"`
}

// TraceContextPropagator configures the tracecontext propagator.
type TraceContextPropagator struct{}

// BaggagePropagator configures the baggage propagator.
type BaggagePropagator struct{}

// B3Propagator configures the b3 propagator.
type B3Propagator struct{}

// B3MultiPropagator configures the b3multi propagator.
type B3MultiPropagator struct{}

// JaegerPropagator configures the jaeger propagator.
type JaegerPropagator struct{}

// OTTracePropagator configures the ottrace propagator.
type OTTracePropagator struct{}

// XRayPropagator configures the xray propagator.
type XRayPropagator struct{}

// ============================================
// TracerProvider Types
// ============================================

// TracerProvider configures the tracer provider.
type TracerProvider struct {
	// Processors configures span processors.
	// +optional
	Processors []SpanProcessor `json:"processors,omitempty"`

	// Limits configures span limits. See also attribute_limits.
	// +optional
	Limits *SpanLimits `json:"limits,omitempty"`

	// Sampler configures the sampler. If omitted, parent based sampler with a root of always_on is used.
	// +optional
	Sampler *Sampler `json:"sampler,omitempty"`
}

// SpanProcessor configures a span processor.
// Only one of batch or simple should be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.batch) ? 1 : 0) + (has(self.simple) ? 1 : 0) <= 1",message="only one of batch or simple can be specified"
type SpanProcessor struct {
	// Batch configures a batch span processor. If omitted, ignore.
	// +optional
	Batch *BatchSpanProcessor `json:"batch,omitempty"`

	// Simple configures a simple span processor. If omitted, ignore.
	// +optional
	Simple *SimpleSpanProcessor `json:"simple,omitempty"`
}

// BatchSpanProcessor configures a batch span processor.
type BatchSpanProcessor struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter SpanExporter `json:"exporter"`

	// ScheduleDelay configures delay interval (in milliseconds) between two consecutive exports.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ScheduleDelay *int `json:"schedule_delay,omitempty"`

	// ExportTimeout configures maximum allowed time (in milliseconds) to export data.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ExportTimeout *int `json:"export_timeout,omitempty"`

	// MaxQueueSize configures maximum queue size. Value must be positive.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxQueueSize *int `json:"max_queue_size,omitempty"`

	// MaxExportBatchSize configures maximum batch size. Value must be positive.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxExportBatchSize *int `json:"max_export_batch_size,omitempty"`
}

// SimpleSpanProcessor configures a simple span processor.
type SimpleSpanProcessor struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter SpanExporter `json:"exporter"`
}

// SpanExporter configures span exporter.
// Only one exporter type should be specified.
type SpanExporter struct {
	// OTLP configures exporter to be OTLP. If omitted, ignore.
	// +optional
	OTLP *OTLP `json:"otlp,omitempty"`

	// Console configures exporter to be console. If omitted, ignore.
	// +optional
	Console *Console `json:"console,omitempty"`
}

// SpanLimits configures span limits. See also attribute_limits.
type SpanLimits struct {
	// AttributeValueLengthLimit configures max attribute value size. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty"`

	// AttributeCountLimit configures max attribute count. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty"`

	// EventCountLimit configures max span event count. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	EventCountLimit *int `json:"event_count_limit,omitempty"`

	// LinkCountLimit configures max span link count. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	LinkCountLimit *int `json:"link_count_limit,omitempty"`

	// EventAttributeCountLimit configures max attributes per span event. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	EventAttributeCountLimit *int `json:"event_attribute_count_limit,omitempty"`

	// LinkAttributeCountLimit configures max attributes per span link. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	LinkAttributeCountLimit *int `json:"link_attribute_count_limit,omitempty"`
}

// ============================================
// Sampler Types
// ============================================

// Sampler configures the sampler.
// Only one sampler type should be specified.
// +kubebuilder:pruning:PreserveUnknownFields
// +kubebuilder:validation:XValidation:rule="(has(self.always_on) ? 1 : 0) + (has(self.always_off) ? 1 : 0) + (has(self.trace_id_ratio_based) ? 1 : 0) + (has(self.parent_based) ? 1 : 0) + (has(self.jaeger_remote) ? 1 : 0) <= 1",message="only one sampler type can be specified"
type Sampler struct {
	// AlwaysOn configures sampler to be always_on. If omitted, ignore.
	// +optional
	AlwaysOn *AlwaysOnSampler `json:"always_on,omitempty"`

	// AlwaysOff configures sampler to be always_off. If omitted, ignore.
	// +optional
	AlwaysOff *AlwaysOffSampler `json:"always_off,omitempty"`

	// TraceIDRatioBased configures sampler to be trace_id_ratio_based. If omitted, ignore.
	// +optional
	TraceIDRatioBased *TraceIDRatioBasedSampler `json:"trace_id_ratio_based,omitempty"`

	// ParentBased configures sampler to be parent_based. If omitted, ignore.
	// +optional
	ParentBased *ParentBasedSampler `json:"parent_based,omitempty"`

	// JaegerRemote configures sampler to be jaeger_remote. If omitted, ignore.
	// +optional
	JaegerRemote *JaegerRemoteSampler `json:"jaeger_remote,omitempty"`
}

// AlwaysOnSampler configures sampler to be always_on.
type AlwaysOnSampler struct{}

// AlwaysOffSampler configures sampler to be always_off.
type AlwaysOffSampler struct{}

// TraceIDRatioBasedSampler configures sampler to be trace_id_ratio_based.
type TraceIDRatioBasedSampler struct {
	// Ratio configures trace_id_ratio. If omitted or null, 1.0 is used.
	// Must be a value between 0.0 and 1.0.
	// +optional
	// +kubebuilder:validation:Minimum=0.0
	// +kubebuilder:validation:Maximum=1.0
	// +kubebuilder:default=1.0
	Ratio *float64 `json:"ratio,omitempty"`
}

// ParentBasedSampler configures sampler to be parent_based.
type ParentBasedSampler struct {
	// Root configures root sampler. If omitted, always_on is used.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Root *Sampler `json:"root,omitempty"`

	// RemoteParentSampled configures remote_parent_sampled sampler. If omitted, always_on is used.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	RemoteParentSampled *Sampler `json:"remote_parent_sampled,omitempty"`

	// RemoteParentNotSampled configures remote_parent_not_sampled sampler. If omitted, always_off is used.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	RemoteParentNotSampled *Sampler `json:"remote_parent_not_sampled,omitempty"`

	// LocalParentSampled configures local_parent_sampled sampler. If omitted, always_on is used.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	LocalParentSampled *Sampler `json:"local_parent_sampled,omitempty"`

	// LocalParentNotSampled configures local_parent_not_sampled sampler. If omitted, always_off is used.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	LocalParentNotSampled *Sampler `json:"local_parent_not_sampled,omitempty"`
}

// JaegerRemoteSampler configures sampler to be jaeger_remote.
type JaegerRemoteSampler struct {
	// Endpoint configures the endpoint of the jaeger remote sampling service.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// PollingInterval configures the polling interval (in milliseconds) to fetch from the remote sampling service.
	// +optional
	// +kubebuilder:validation:Minimum=0
	PollingInterval *int `json:"polling_interval,omitempty"`

	// InitialSampler configures the initial sampler used before first configuration is fetched.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	InitialSampler *Sampler `json:"initial_sampler,omitempty"`
}

// ============================================
// MeterProvider Types
// ============================================

// MeterProvider configures the meter provider.
type MeterProvider struct {
	// Readers configures metric readers.
	// +optional
	Readers []MetricReader `json:"readers,omitempty"`

	// Views configures views. Each view has a selector which determines the instrument(s) it applies to.
	// +optional
	// +kubebuilder:validation:MaxItems=128
	Views []View `json:"views,omitempty"`
}

// MetricReader configures metric reader.
// Only one of pull or periodic should be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.pull) ? 1 : 0) + (has(self.periodic) ? 1 : 0) <= 1",message="only one of pull or periodic can be specified"
type MetricReader struct {
	// Pull configures a pull based metric reader. If omitted, ignore.
	// +optional
	Pull *PullMetricReader `json:"pull,omitempty"`

	// Periodic configures a periodic metric reader. If omitted, ignore.
	// +optional
	Periodic *PeriodicMetricReader `json:"periodic,omitempty"`
}

// PullMetricReader configures a pull based metric reader.
type PullMetricReader struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter PullMetricExporter `json:"exporter"`
}

// PullMetricExporter configures pull metric exporter.
type PullMetricExporter struct {
	// Prometheus configures exporter to be prometheus. If omitted, ignore.
	// +optional
	Prometheus *Prometheus `json:"prometheus,omitempty"`
}

// Prometheus configures Prometheus exporter.
type Prometheus struct {
	// Host configures host. If omitted or null, localhost is used.
	// +optional
	Host *string `json:"host,omitempty"`

	// Port configures port. If omitted or null, 9464 is used.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port *int `json:"port,omitempty"`

	// WithoutUnits configures Prometheus Exporter to produce metrics without unit suffixes.
	// +optional
	WithoutUnits *bool `json:"without_units,omitempty"`

	// WithoutTypeSuffix configures Prometheus Exporter to produce metrics without type suffixes.
	// +optional
	WithoutTypeSuffix *bool `json:"without_type_suffix,omitempty"`

	// WithoutScopeInfo configures Prometheus Exporter to produce metrics without a scope info metric.
	// +optional
	WithoutScopeInfo *bool `json:"without_scope_info,omitempty"`

	// WithResourceConstantLabels configures resource constant labels.
	// +optional
	WithResourceConstantLabels *IncludeExclude `json:"with_resource_constant_labels,omitempty"`
}

// IncludeExclude defines inclusion/exclusion lists.
type IncludeExclude struct {
	// Included lists the items to include.
	// +optional
	Included []string `json:"included,omitempty"`

	// Excluded lists the items to exclude.
	// +optional
	Excluded []string `json:"excluded,omitempty"`
}

// PeriodicMetricReader configures a periodic metric reader.
type PeriodicMetricReader struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter PushMetricExporter `json:"exporter"`

	// Interval configures delay interval (in milliseconds) between start of two consecutive exports.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Interval *int `json:"interval,omitempty"`

	// Timeout configures maximum allowed time (in milliseconds) to export data.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Timeout *int `json:"timeout,omitempty"`
}

// PushMetricExporter configures push metric exporter.
type PushMetricExporter struct {
	// OTLP configures exporter to be OTLP. If omitted, ignore.
	// +optional
	OTLP *OTLPMetric `json:"otlp,omitempty"`

	// Console configures exporter to be console. If omitted, ignore.
	// +optional
	Console *Console `json:"console,omitempty"`
}

// OTLPMetric defines OTLP metric exporter configuration.
type OTLPMetric struct {
	// Protocol is the OTLP transport protocol. Valid values: grpc, http/protobuf.
	// +optional
	// +kubebuilder:validation:Enum=grpc;http/protobuf
	Protocol *string `json:"protocol,omitempty"`

	// Endpoint is the target URL to send telemetry to.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// Certificate is the path to the TLS certificate.
	// +optional
	Certificate *string `json:"certificate,omitempty"`

	// ClientKey is the path to the TLS client key.
	// +optional
	ClientKey *string `json:"client_key,omitempty"`

	// ClientCertificate is the path to the TLS client certificate.
	// +optional
	ClientCertificate *string `json:"client_certificate,omitempty"`

	// Headers are additional headers to send with requests.
	// +optional
	Headers []NameStringValuePair `json:"headers,omitempty"`

	// HeadersList is a comma-separated list of headers.
	// +optional
	HeadersList *string `json:"headers_list,omitempty"`

	// Compression is the compression type. Valid values: gzip, none.
	// +optional
	// +kubebuilder:validation:Enum=gzip;none
	Compression *string `json:"compression,omitempty"`

	// Timeout is the export timeout in milliseconds.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Timeout *int `json:"timeout,omitempty"`

	// Insecure disables TLS.
	// +optional
	Insecure *bool `json:"insecure,omitempty"`

	// TemporalityPreference is the temporality preference for metrics.
	// Valid values: cumulative, delta, lowmemory.
	// +optional
	// +kubebuilder:validation:Enum=cumulative;delta;lowmemory
	TemporalityPreference *string `json:"temporality_preference,omitempty"`

	// DefaultHistogramAggregation is the default histogram aggregation.
	// Valid values: explicit_bucket_histogram, base2_exponential_bucket_histogram.
	// +optional
	// +kubebuilder:validation:Enum=explicit_bucket_histogram;base2_exponential_bucket_histogram
	DefaultHistogramAggregation *string `json:"default_histogram_aggregation,omitempty"`
}

// ============================================
// View Types
// ============================================

// View configures a metric view.
type View struct {
	// Selector configures view selector. Selection criteria is additive.
	// +optional
	Selector *ViewSelector `json:"selector,omitempty"`

	// Stream configures view stream.
	// +optional
	Stream *ViewStream `json:"stream,omitempty"`
}

// ViewSelector configures selection criteria for a metric view.
type ViewSelector struct {
	// InstrumentName configures instrument name selection criteria.
	// +optional
	InstrumentName *string `json:"instrument_name,omitempty"`

	// InstrumentType configures instrument type selection criteria.
	// +optional
	// +kubebuilder:validation:Enum=counter;histogram;observable_counter;observable_gauge;observable_up_down_counter;up_down_counter
	InstrumentType *string `json:"instrument_type,omitempty"`

	// MeterName configures meter name selection criteria.
	// +optional
	MeterName *string `json:"meter_name,omitempty"`

	// MeterVersion configures meter version selection criteria.
	// +optional
	MeterVersion *string `json:"meter_version,omitempty"`

	// MeterSchemaURL configures meter schema URL selection criteria.
	// +optional
	MeterSchemaURL *string `json:"meter_schema_url,omitempty"`

	// Unit configures the instrument unit selection criteria.
	// +optional
	Unit *string `json:"unit,omitempty"`
}

// ViewStream configures output stream for a view.
type ViewStream struct {
	// Name configures metric name of the resulting stream(s).
	// +optional
	Name *string `json:"name,omitempty"`

	// Description configures metric description of the resulting stream(s).
	// +optional
	Description *string `json:"description,omitempty"`

	// AttributeKeys configures attribute keys retained in the resulting stream(s).
	// +optional
	AttributeKeys *IncludeExclude `json:"attribute_keys,omitempty"`

	// Aggregation configures aggregation of the resulting stream(s). If omitted, default is used.
	// +optional
	Aggregation *ViewAggregation `json:"aggregation,omitempty"`
}

// ViewAggregation configures aggregation for a view.
// Only one aggregation type should be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.default) ? 1 : 0) + (has(self.drop) ? 1 : 0) + (has(self.sum) ? 1 : 0) + (has(self.last_value) ? 1 : 0) + (has(self.explicit_bucket_histogram) ? 1 : 0) + (has(self.base2_exponential_bucket_histogram) ? 1 : 0) <= 1",message="only one aggregation type can be specified"
type ViewAggregation struct {
	// Default configures the stream to use the instrument kind to select an aggregation.
	// +optional
	Default *DefaultAggregation `json:"default,omitempty"`

	// Drop configures the stream to ignore/drop all instrument measurements.
	// +optional
	Drop *DropAggregation `json:"drop,omitempty"`

	// Sum configures the stream to collect the arithmetic sum of measurement values.
	// +optional
	Sum *SumAggregation `json:"sum,omitempty"`

	// LastValue configures the stream to collect data using the last measurement.
	// +optional
	LastValue *LastValueAggregation `json:"last_value,omitempty"`

	// ExplicitBucketHistogram configures the stream to collect data for the histogram metric point
	// using a set of explicit boundary values.
	// +optional
	ExplicitBucketHistogram *ExplicitBucketHistogramAggregation `json:"explicit_bucket_histogram,omitempty"`

	// Base2ExponentialBucketHistogram configures the stream to collect data for the exponential histogram metric point.
	// +optional
	Base2ExponentialBucketHistogram *Base2ExponentialBucketHistogramAggregation `json:"base2_exponential_bucket_histogram,omitempty"`
}

// DefaultAggregation configures the stream to use the instrument kind to select an aggregation.
type DefaultAggregation struct{}

// DropAggregation configures the stream to ignore/drop all instrument measurements.
type DropAggregation struct{}

// SumAggregation configures the stream to collect the arithmetic sum of measurement values.
type SumAggregation struct{}

// LastValueAggregation configures the stream to collect data using the last measurement.
type LastValueAggregation struct{}

// ExplicitBucketHistogramAggregation configures the stream to collect data for the histogram metric point
// using a set of explicit boundary values.
type ExplicitBucketHistogramAggregation struct {
	// Boundaries configures bucket boundaries.
	// +optional
	Boundaries []float64 `json:"boundaries,omitempty"`

	// RecordMinMax configures record min and max. If omitted or null, true is used.
	// +optional
	// +kubebuilder:default=true
	RecordMinMax *bool `json:"record_min_max,omitempty"`
}

// Base2ExponentialBucketHistogramAggregation configures the stream to collect data for the exponential histogram metric point.
type Base2ExponentialBucketHistogramAggregation struct {
	// MaxScale configures the max scale factor. If omitted or null, 20 is used.
	// +optional
	// +kubebuilder:validation:Minimum=-10
	// +kubebuilder:validation:Maximum=20
	// +kubebuilder:default=20
	MaxScale *int `json:"max_scale,omitempty"`

	// MaxSize configures the maximum number of buckets in each of the positive and negative ranges.
	// +optional
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:default=160
	MaxSize *int `json:"max_size,omitempty"`

	// RecordMinMax configures record min and max. If omitted or null, true is used.
	// +optional
	// +kubebuilder:default=true
	RecordMinMax *bool `json:"record_min_max,omitempty"`
}

// ============================================
// LoggerProvider Types
// ============================================

// LoggerProvider configures the logger provider.
type LoggerProvider struct {
	// Processors configures log record processors.
	// +optional
	Processors []LogRecordProcessor `json:"processors,omitempty"`

	// Limits configures log record limits. See also attribute_limits.
	// +optional
	Limits *LogRecordLimits `json:"limits,omitempty"`
}

// LogRecordProcessor configures log record processor.
// Only one of batch or simple should be specified.
// +kubebuilder:validation:XValidation:rule="(has(self.batch) ? 1 : 0) + (has(self.simple) ? 1 : 0) <= 1",message="only one of batch or simple can be specified"
type LogRecordProcessor struct {
	// Batch configures a batch log record processor. If omitted, ignore.
	// +optional
	Batch *BatchLogRecordProcessor `json:"batch,omitempty"`

	// Simple configures a simple log record processor. If omitted, ignore.
	// +optional
	Simple *SimpleLogRecordProcessor `json:"simple,omitempty"`
}

// BatchLogRecordProcessor configures a batch log record processor.
type BatchLogRecordProcessor struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter LogRecordExporter `json:"exporter"`

	// ScheduleDelay configures delay interval (in milliseconds) between two consecutive exports.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ScheduleDelay *int `json:"schedule_delay,omitempty"`

	// ExportTimeout configures maximum allowed time (in milliseconds) to export data.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ExportTimeout *int `json:"export_timeout,omitempty"`

	// MaxQueueSize configures maximum queue size. Value must be positive.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxQueueSize *int `json:"max_queue_size,omitempty"`

	// MaxExportBatchSize configures maximum batch size. Value must be positive.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxExportBatchSize *int `json:"max_export_batch_size,omitempty"`
}

// SimpleLogRecordProcessor configures a simple log record processor.
type SimpleLogRecordProcessor struct {
	// Exporter configures exporter. Property is required and must be non-null.
	// +required
	// +kubebuilder:validation:Required
	Exporter LogRecordExporter `json:"exporter"`
}

// LogRecordExporter configures log record exporter.
type LogRecordExporter struct {
	// OTLP configures exporter to be OTLP. If omitted, ignore.
	// +optional
	OTLP *OTLP `json:"otlp,omitempty"`

	// Console configures exporter to be console. If omitted, ignore.
	// +optional
	Console *Console `json:"console,omitempty"`
}

// LogRecordLimits configures log record limits. See also attribute_limits.
type LogRecordLimits struct {
	// AttributeValueLengthLimit configures max attribute value size. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeValueLengthLimit *int `json:"attribute_value_length_limit,omitempty"`

	// AttributeCountLimit configures max attribute count. Value must be non-negative.
	// +optional
	// +kubebuilder:validation:Minimum=0
	AttributeCountLimit *int `json:"attribute_count_limit,omitempty"`
}

// ============================================
// Common Exporter Types
// ============================================

// OTLP configures OTLP exporter (for traces and logs).
type OTLP struct {
	// Protocol configures the OTLP transport protocol. Known values include: grpc, http/protobuf.
	// +optional
	// +kubebuilder:validation:Enum=grpc;http/protobuf
	Protocol *string `json:"protocol,omitempty"`

	// Endpoint configures endpoint.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// Certificate configures the path to the TLS certificate.
	// +optional
	Certificate *string `json:"certificate,omitempty"`

	// ClientKey configures the path to the TLS client key.
	// +optional
	ClientKey *string `json:"client_key,omitempty"`

	// ClientCertificate configures the path to the TLS client certificate.
	// +optional
	ClientCertificate *string `json:"client_certificate,omitempty"`

	// Headers configures headers. Entries have higher priority than entries from .headers_list.
	// +optional
	Headers []NameStringValuePair `json:"headers,omitempty"`

	// HeadersList configures headers. Entries have lower priority than entries from .headers.
	// +optional
	HeadersList *string `json:"headers_list,omitempty"`

	// Compression configures compression. Known values include: gzip, none.
	// +optional
	// +kubebuilder:validation:Enum=gzip;none
	Compression *string `json:"compression,omitempty"`

	// Timeout configures max time (in milliseconds) to wait for each export.
	// +optional
	// +kubebuilder:validation:Minimum=0
	Timeout *int `json:"timeout,omitempty"`

	// Insecure disables TLS.
	// +optional
	Insecure *bool `json:"insecure,omitempty"`
}

// Console configures exporter to be console.
type Console struct{}

// NameStringValuePair represents a name-value pair for headers.
type NameStringValuePair struct {
	// Name is the header name.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Value is the header value.
	// +optional
	Value *string `json:"value"`
}
