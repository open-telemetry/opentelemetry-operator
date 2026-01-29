// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/sha256"
	"embed"
	"fmt"

	"dario.cat/mergo"
	"github.com/goccy/go-yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

//go:embed configs
var configFiles embed.FS

// CollectorType represents the type of collector configuration.
type CollectorType string

const (
	AgentCollectorType   CollectorType = "agent-collector"
	ClusterCollectorType CollectorType = "cluster-collector"
)

// DistroProvider represents a Kubernetes distribution and cloud provider combination.
type DistroProvider string

const (
	OpenShift DistroProvider = "openshift"
	// Future distros can be added here:
	// EKS       DistroProvider = "eks"
	// GKE       DistroProvider = "gke"
	// AKS       DistroProvider = "aks".
)

// CollectorConfigSpec represents the collector configuration structure from YAML.
type CollectorConfigSpec struct {
	Receivers   map[string]interface{} `yaml:"receivers"`
	Processors  map[string]interface{} `yaml:"processors"`
	Exporters   map[string]interface{} `yaml:"exporters"`
	Service     ServiceConfig          `yaml:"service"`
	Environment map[string]string      `yaml:"environment,omitempty"`
}

// ServiceConfig represents the service section of collector config.
type ServiceConfig struct {
	Pipelines map[string]PipelineConfig `yaml:"pipelines"`
}

// PipelineConfig represents a pipeline configuration.
type PipelineConfig struct {
	Receivers  []string `yaml:"receivers"`
	Processors []string `yaml:"processors"`
	Exporters  []string `yaml:"exporters"`
}

// ConfigLoader loads and merges collector configurations.
type ConfigLoader struct {
	fs embed.FS
}

// NewConfigLoader creates a new config loader.
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		fs: configFiles,
	}
}

// LoadCollectorConfig loads and merges collector configuration for the specified type and distro.
func (c *ConfigLoader) LoadCollectorConfig(collectorType CollectorType, distroProvider DistroProvider, spec v1alpha1.ClusterObservabilitySpec) (v1beta1.Config, error) {
	// Load base configuration
	baseConfig, err := c.loadBaseConfig(collectorType)
	if err != nil {
		return v1beta1.Config{}, fmt.Errorf("failed to load base config: %w", err)
	}

	// Load distro-specific overrides if they exist
	overrideConfig, err := c.loadDistroOverrides(collectorType, distroProvider)
	if err != nil {
		// Overrides are optional, continue without them
		overrideConfig = nil
	}

	// Merge configurations
	finalConfig := baseConfig
	if overrideConfig != nil {
		if mergeErr := mergo.Merge(&finalConfig, *overrideConfig, mergo.WithOverride); mergeErr != nil {
			return v1beta1.Config{}, fmt.Errorf("failed to merge configs: %w", mergeErr)
		}
	}

	// Build exporters configuration based on per-signal or default exporter
	exporters := c.buildExportersConfig(spec)
	finalConfig.Exporters = exporters

	// Build pipelines with all signals enabled
	pipelines := c.buildPipelinesWithExporters(collectorType)

	finalConfig.Service.Pipelines = pipelines

	// Convert to v1beta1.Config
	v1beta1Config := c.convertToV1Beta1Config(finalConfig)

	return v1beta1Config, nil
}

// loadBaseConfig loads the base configuration for a collector type.
func (c *ConfigLoader) loadBaseConfig(collectorType CollectorType) (CollectorConfigSpec, error) {
	filename := fmt.Sprintf("configs/%s-base.yaml", string(collectorType))

	data, err := c.fs.ReadFile(filename)
	if err != nil {
		return CollectorConfigSpec{}, fmt.Errorf("failed to read base config file %s: %w", filename, err)
	}

	var config CollectorConfigSpec
	if err := yaml.Unmarshal(data, &config); err != nil {
		return CollectorConfigSpec{}, fmt.Errorf("failed to unmarshal base config: %w", err)
	}

	return config, nil
}

// loadDistroOverrides loads distro-specific configuration overrides.
func (c *ConfigLoader) loadDistroOverrides(collectorType CollectorType, distroProvider DistroProvider) (*CollectorConfigSpec, error) {
	filename := fmt.Sprintf("configs/distros/%s/%s-overrides.yaml", string(distroProvider), string(collectorType))

	data, err := c.fs.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read distro override file %s: %w", filename, err)
	}

	var config CollectorConfigSpec
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal distro override config: %w", err)
	}

	return &config, nil
}

// buildExportersConfig builds exporters configuration from the single OTLP HTTP exporter spec.
func (c *ConfigLoader) buildExportersConfig(spec v1alpha1.ClusterObservabilitySpec) map[string]interface{} {
	exporters := make(map[string]interface{})

	// Build the otlphttp exporter configuration - map exactly to collector fields
	otlpConfig := map[string]interface{}{}

	// Handle endpoints - either base endpoint or per-signal endpoints
	if spec.Exporter.Endpoint != "" {
		otlpConfig["endpoint"] = spec.Exporter.Endpoint
	}
	if spec.Exporter.TracesEndpoint != "" {
		otlpConfig["traces_endpoint"] = spec.Exporter.TracesEndpoint
	}
	if spec.Exporter.MetricsEndpoint != "" {
		otlpConfig["metrics_endpoint"] = spec.Exporter.MetricsEndpoint
	}
	if spec.Exporter.LogsEndpoint != "" {
		otlpConfig["logs_endpoint"] = spec.Exporter.LogsEndpoint
	}
	if spec.Exporter.ProfilesEndpoint != "" {
		otlpConfig["profiles_endpoint"] = spec.Exporter.ProfilesEndpoint
	}

	// TODO: We do not really handle taking the CA/Cert/Key.
	if spec.Exporter.TLS != nil {
		tlsConfig := map[string]interface{}{}
		if spec.Exporter.TLS.CAFile != "" {
			tlsConfig["ca_file"] = spec.Exporter.TLS.CAFile
		}
		if spec.Exporter.TLS.CertFile != "" {
			tlsConfig["cert_file"] = spec.Exporter.TLS.CertFile
		}
		if spec.Exporter.TLS.KeyFile != "" {
			tlsConfig["key_file"] = spec.Exporter.TLS.KeyFile
		}
		if spec.Exporter.TLS.Insecure {
			tlsConfig["insecure"] = spec.Exporter.TLS.Insecure
		}
		if spec.Exporter.TLS.ServerName != "" {
			tlsConfig["server_name"] = spec.Exporter.TLS.ServerName
		}
		if len(tlsConfig) > 0 {
			otlpConfig["tls"] = tlsConfig
		}
	}

	if spec.Exporter.Timeout != "" {
		otlpConfig["timeout"] = spec.Exporter.Timeout
	}
	if spec.Exporter.ReadBufferSize != nil {
		otlpConfig["read_buffer_size"] = *spec.Exporter.ReadBufferSize
	}
	if spec.Exporter.WriteBufferSize != nil {
		otlpConfig["write_buffer_size"] = *spec.Exporter.WriteBufferSize
	}
	if spec.Exporter.Encoding != "" {
		otlpConfig["encoding"] = spec.Exporter.Encoding
	}
	if spec.Exporter.Compression != "" {
		otlpConfig["compression"] = spec.Exporter.Compression
	}
	if len(spec.Exporter.Headers) > 0 {
		otlpConfig["headers"] = spec.Exporter.Headers
	}

	if spec.Exporter.SendingQueue != nil {
		queueConfig := map[string]interface{}{}
		if spec.Exporter.SendingQueue.Enabled != nil {
			queueConfig["enabled"] = *spec.Exporter.SendingQueue.Enabled
		}
		if spec.Exporter.SendingQueue.NumConsumers != nil {
			queueConfig["num_consumers"] = *spec.Exporter.SendingQueue.NumConsumers
		}
		if spec.Exporter.SendingQueue.QueueSize != nil {
			queueConfig["queue_size"] = *spec.Exporter.SendingQueue.QueueSize
		}
		if len(queueConfig) > 0 {
			otlpConfig["sending_queue"] = queueConfig
		}
	}

	if spec.Exporter.RetryOnFailure != nil {
		retryConfig := map[string]interface{}{}
		if spec.Exporter.RetryOnFailure.Enabled != nil {
			retryConfig["enabled"] = *spec.Exporter.RetryOnFailure.Enabled
		}
		if spec.Exporter.RetryOnFailure.InitialInterval != "" {
			retryConfig["initial_interval"] = spec.Exporter.RetryOnFailure.InitialInterval
		}
		if spec.Exporter.RetryOnFailure.RandomizationFactor != "" {
			retryConfig["randomization_factor"] = spec.Exporter.RetryOnFailure.RandomizationFactor
		}
		if spec.Exporter.RetryOnFailure.Multiplier != "" {
			retryConfig["multiplier"] = spec.Exporter.RetryOnFailure.Multiplier
		}
		if spec.Exporter.RetryOnFailure.MaxInterval != "" {
			retryConfig["max_interval"] = spec.Exporter.RetryOnFailure.MaxInterval
		}
		if spec.Exporter.RetryOnFailure.MaxElapsedTime != "" {
			retryConfig["max_elapsed_time"] = spec.Exporter.RetryOnFailure.MaxElapsedTime
		}
		if len(retryConfig) > 0 {
			otlpConfig["retry_on_failure"] = retryConfig
		}
	}

	exporters["otlphttp"] = otlpConfig
	return exporters
}

// buildPipelinesWithExporters creates service pipelines based on collector type.
func (c *ConfigLoader) buildPipelinesWithExporters(collectorType CollectorType) map[string]PipelineConfig {
	pipelines := make(map[string]PipelineConfig)

	// We only use the otlphttp exporter for now
	exporterName := "otlphttp"

	if collectorType == AgentCollectorType {
		// Agent collector: metrics, logs, traces
		pipelines["metrics"] = PipelineConfig{
			Receivers:  []string{"otlp", "kubeletstats"},
			Processors: []string{"resourcedetection", "k8sattributes", "batch"},
			Exporters:  []string{exporterName},
		}
		pipelines["logs"] = PipelineConfig{
			Receivers:  []string{"filelog"},
			Processors: []string{"k8sattributes", "batch"},
			Exporters:  []string{exporterName},
		}
		pipelines["traces"] = PipelineConfig{
			Receivers:  []string{"otlp"},
			Processors: []string{"resourcedetection", "k8sattributes", "batch"},
			Exporters:  []string{exporterName},
		}
	} else if collectorType == ClusterCollectorType {
		// Cluster collector: metrics, logs (k8s events)
		pipelines["metrics"] = PipelineConfig{
			Receivers:  []string{"k8s_cluster"},
			Processors: []string{"resourcedetection", "batch"},
			Exporters:  []string{exporterName},
		}
		pipelines["logs"] = PipelineConfig{
			Receivers:  []string{"k8s_events"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	return pipelines
}

// convertToV1Beta1Config converts our internal config to v1beta1.Config.
func (c *ConfigLoader) convertToV1Beta1Config(config CollectorConfigSpec) v1beta1.Config {
	// Convert pipelines
	v1beta1Pipelines := make(map[string]*v1beta1.Pipeline)
	for name, pipeline := range config.Service.Pipelines {
		v1beta1Pipelines[name] = &v1beta1.Pipeline{
			Receivers:  pipeline.Receivers,
			Processors: pipeline.Processors,
			Exporters:  pipeline.Exporters,
		}
	}

	return v1beta1.Config{
		Receivers: v1beta1.AnyConfig{
			Object: config.Receivers,
		},
		Processors: &v1beta1.AnyConfig{
			Object: config.Processors,
		},
		Exporters: v1beta1.AnyConfig{
			Object: config.Exporters,
		},
		Service: v1beta1.Service{
			Pipelines: v1beta1Pipelines,
		},
	}
}

// GetAvailableDistroProviders returns a list of available distro/provider combinations.
func (c *ConfigLoader) GetAvailableDistroProviders() []DistroProvider {
	return []DistroProvider{
		OpenShift,
		// Add more as they become available
	}
}

// DetectDistroProvider detects the Kubernetes distribution.
func (c *ConfigLoader) DetectDistroProvider(cfg config.Config) DistroProvider {
	if cfg.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		return OpenShift
	}

	// Default to no specific distro (use base configs only)
	return ""
}

// ValidateConfig validates that a configuration is valid.
func (c *ConfigLoader) ValidateConfig(config v1beta1.Config) error {
	if len(config.Receivers.Object) == 0 {
		return fmt.Errorf("no receivers configured")
	}

	if len(config.Exporters.Object) == 0 {
		return fmt.Errorf("no exporters configured")
	}

	if len(config.Service.Pipelines) == 0 {
		return fmt.Errorf("no pipelines configured")
	}

	// Validate that pipeline components exist
	for pipelineName, pipeline := range config.Service.Pipelines {
		if pipeline == nil {
			continue
		}

		// Check receivers exist
		for _, receiver := range pipeline.Receivers {
			if _, exists := config.Receivers.Object[receiver]; !exists {
				return fmt.Errorf("pipeline %s references non-existent receiver %s", pipelineName, receiver)
			}
		}

		// Check processors exist (if any configured)
		if config.Processors != nil {
			for _, processor := range pipeline.Processors {
				if _, exists := config.Processors.Object[processor]; !exists {
					return fmt.Errorf("pipeline %s references non-existent processor %s", pipelineName, processor)
				}
			}
		}

		// Check exporters exist
		for _, exporter := range pipeline.Exporters {
			if _, exists := config.Exporters.Object[exporter]; !exists {
				return fmt.Errorf("pipeline %s references non-existent exporter %s", pipelineName, exporter)
			}
		}
	}

	return nil
}

// GetConfigVersion returns a version hash representing the current embedded configs.
// This can be used to detect when config files have changed between operator versions.
func (c *ConfigLoader) GetConfigVersion(collectorType CollectorType, distroProvider DistroProvider) (string, error) {
	hasher := sha256.New()

	// Hash the base config
	baseFilename := fmt.Sprintf("configs/%s-base.yaml", string(collectorType))
	baseData, err := c.fs.ReadFile(baseFilename)
	if err != nil {
		return "", fmt.Errorf("failed to read base config for version: %w", err)
	}
	hasher.Write(baseData)

	// Hash the distro override config if it exists
	overrideFilename := fmt.Sprintf("configs/distros/%s/%s-overrides.yaml", string(distroProvider), string(collectorType))
	overrideData, err := c.fs.ReadFile(overrideFilename)
	if err == nil {
		hasher.Write(overrideData)
	} // Ignore missing override files - they're optional

	// Include the distro provider in the hash
	hasher.Write([]byte(string(distroProvider)))

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// CompareConfigVersions compares two config versions and returns true if they differ.
func (c *ConfigLoader) CompareConfigVersions(version1, version2 string) bool {
	return version1 != version2
}

// GetAllConfigVersions returns version hashes for all supported collector types and distros.
func (c *ConfigLoader) GetAllConfigVersions() (map[string]string, error) {
	versions := make(map[string]string)

	collectorTypes := []CollectorType{AgentCollectorType, ClusterCollectorType}
	distroProviders := c.GetAvailableDistroProviders()

	for _, collectorType := range collectorTypes {
		for _, distroProvider := range distroProviders {
			versionKey := fmt.Sprintf("%s-%s", string(collectorType), string(distroProvider))
			version, err := c.GetConfigVersion(collectorType, distroProvider)
			if err != nil {
				return nil, fmt.Errorf("failed to get version for %s: %w", versionKey, err)
			}
			versions[versionKey] = version
		}
	}

	return versions, nil
}
