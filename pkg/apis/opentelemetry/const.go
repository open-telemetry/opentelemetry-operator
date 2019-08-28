package opentelemetry

// ContextEntry represents a key in the context map
type ContextEntry string

// ConfigMapEntry represents an entry in a config map
type ConfigMapEntry string

// Instance is the OpenTelemetryService CR (instance) that is the current target of the reconciliation
const Instance ContextEntry = "__instance"

// CollectorConfigMapEntry represents the configuration file name for the collector
const CollectorConfigMapEntry = "collector.yaml"
