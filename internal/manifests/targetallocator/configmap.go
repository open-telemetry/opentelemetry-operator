// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	targetAllocatorFilename = "targetallocator.yaml"
)

func ConfigMap(params Params) (*corev1.ConfigMap, error) {
	instance := params.TargetAllocator
	name := naming.TAConfigMap(instance.Name)
	labels := manifestutils.Labels(instance.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)
	taSpec := instance.Spec

	taConfig := make(map[any]any)
	// Set config if global or scrape configs set
	config := map[string]any{}
	var (
		globalConfig      map[string]any
		scrapeConfigs     []v1beta1.AnyConfig
		collectorSelector *metav1.LabelSelector
		err               error
	)
	if params.Collector != nil {
		collectorSelector = &metav1.LabelSelector{
			MatchLabels: manifestutils.SelectorLabels(params.Collector.ObjectMeta, collector.ComponentOpenTelemetryCollector),
		}

		globalConfig, err = getGlobalConfig(taSpec.GlobalConfig, params.Collector.Spec.Config)
		if err != nil {
			return nil, err
		}

		scrapeConfigs, err = getScrapeConfigs(taSpec.ScrapeConfigs, params.Collector.Spec.Config)
		if err != nil {
			return nil, err
		}
	} else { // if there's no collector, just use what's in the TargetAllocator CR
		collectorSelector = nil
		globalConfig = taSpec.GlobalConfig.Object
		scrapeConfigs = taSpec.ScrapeConfigs
	}

	if len(globalConfig) > 0 {
		config["global"] = globalConfig
	}

	if len(scrapeConfigs) > 0 {
		config["scrape_configs"] = scrapeConfigs
	}

	if len(config) != 0 {
		taConfig["config"] = config
	}

	taConfig["collector_selector"] = collectorSelector

	if len(taSpec.AllocationStrategy) > 0 {
		taConfig["allocation_strategy"] = taSpec.AllocationStrategy
	} else {
		taConfig["allocation_strategy"] = v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	}

	if featuregate.EnableTargetAllocatorFallbackStrategy.IsEnabled() {
		taConfig["allocation_fallback_strategy"] = v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	}

	taConfig["filter_strategy"] = taSpec.FilterStrategy

	if taSpec.PrometheusCR.Enabled {
		prometheusCRConfig := map[any]any{
			"enabled": true,
		}
		if taSpec.PrometheusCR.ScrapeInterval.Size() > 0 {
			prometheusCRConfig["scrape_interval"] = taSpec.PrometheusCR.ScrapeInterval.Duration
		}
		if taSpec.PrometheusCR.EvaluationInterval.Size() > 0 {
			prometheusCRConfig["evaluation_interval"] = taSpec.PrometheusCR.EvaluationInterval.Duration
		}
		if taSpec.PrometheusCR.ScrapeProtocols != nil {
			prometheusCRConfig["scrape_protocols"] = taSpec.PrometheusCR.ScrapeProtocols
		}

		if taSpec.PrometheusCR.ScrapeClasses != nil {
			prometheusCRConfig["scrape_classes"] = taSpec.PrometheusCR.ScrapeClasses
		}

		if taSpec.PrometheusCR.AllowNamespaces != nil {
			prometheusCRConfig["allow_namespaces"] = taSpec.PrometheusCR.AllowNamespaces
		}

		if taSpec.PrometheusCR.DenyNamespaces != nil {
			prometheusCRConfig["deny_namespaces"] = taSpec.PrometheusCR.DenyNamespaces
		}

		if taSpec.PrometheusCR.SecretNamespaces != nil {
			prometheusCRConfig["secret_namespaces"] = taSpec.PrometheusCR.SecretNamespaces
		}

		if taSpec.PrometheusCR.DenyFSAccessThroughSMs {
			prometheusCRConfig["deny_fs_access_through_sms"] = true
		}

		prometheusCRConfig["service_monitor_namespace_selector"] = taSpec.PrometheusCR.ServiceMonitorNamespaceSelector
		prometheusCRConfig["service_monitor_selector"] = taSpec.PrometheusCR.ServiceMonitorSelector

		prometheusCRConfig["pod_monitor_namespace_selector"] = taSpec.PrometheusCR.PodMonitorNamespaceSelector
		prometheusCRConfig["pod_monitor_selector"] = taSpec.PrometheusCR.PodMonitorSelector

		prometheusCRConfig["scrape_config_namespace_selector"] = taSpec.PrometheusCR.ScrapeConfigNamespaceSelector
		prometheusCRConfig["scrape_config_selector"] = taSpec.PrometheusCR.ScrapeConfigSelector

		prometheusCRConfig["probe_namespace_selector"] = taSpec.PrometheusCR.ProbeNamespaceSelector
		prometheusCRConfig["probe_selector"] = taSpec.PrometheusCR.ProbeSelector

		taConfig["prometheus_cr"] = prometheusCRConfig
	}

	if manifestutils.IsTAMTLSEnabled(params.TargetAllocator.Spec.Mtls) {
		taConfig["https"] = map[string]any{
			"enabled":            true,
			"listen_addr":        ":8443",
			"ca_file_path":       filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorCAFileName),
			"tls_cert_file_path": filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSCertFileName),
			"tls_key_file_path":  filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSKeyFileName),
		}
	}

	if taSpec.AllowInsecureAuthSecrets {
		taConfig["allow_insecure_auth_secrets"] = true
	}

	if taSpec.CollectorNotReadyGracePeriod.Size() > 0 {
		taConfig["collector_not_ready_grace_period"] = taSpec.CollectorNotReadyGracePeriod.Duration
	}

	if len(taSpec.Telemetry.Metrics.Readers) > 0 {
		readers := make([]any, 0, len(taSpec.Telemetry.Metrics.Readers))
		for _, r := range taSpec.Telemetry.Metrics.Readers {
			if r.Periodic == nil {
				continue
			}
			p := r.Periodic
			exporter := map[string]any{}
			if g := p.Exporter.OtlpGrpc; g != nil {
				grpcMap := buildOTLPExporterMap(g.TAOTLPCommonConfig)
				if g.Tls != nil && g.Tls.Insecure {
					grpcMap["tls"] = map[string]any{"insecure": true}
				}
				exporter["otlp_grpc"] = grpcMap
			} else if h := p.Exporter.OtlpHttp; h != nil {
				exporter["otlp_http"] = buildOTLPExporterMap(h.TAOTLPCommonConfig)
			}
			// interval and timeout are in milliseconds per the OTel declarative config spec.
			periodic := map[string]any{"exporter": exporter}
			if p.Interval != nil {
				periodic["interval"] = int(p.Interval.Milliseconds())
			}
			if p.Timeout != nil {
				periodic["timeout"] = int(p.Timeout.Milliseconds())
			}
			readers = append(readers, map[string]any{"periodic": periodic})
		}
		if len(readers) > 0 {
			taConfig["telemetry"] = map[string]any{
				"metrics": map[string]any{
					"readers": readers,
				},
			}
		}
	}

	taConfigYAML, err := yaml.Marshal(taConfig)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: ResourceAnnotations(params.TargetAllocator, params.Config.AnnotationsFilter),
		},
		Data: map[string]string{
			targetAllocatorFilename: string(taConfigYAML),
		},
	}, nil
}

// buildOTLPExporterMap constructs the common fields shared by otlp_grpc and otlp_http exporters.
func buildOTLPExporterMap(base v1beta1.TAOTLPCommonConfig) map[string]any {
	m := map[string]any{"endpoint": base.Endpoint}
	if len(base.Headers) > 0 {
		h := make([]map[string]any, len(base.Headers))
		for i, hdr := range base.Headers {
			h[i] = map[string]any{"name": hdr.Name, "value": hdr.Value}
		}
		m["headers"] = h
	}
	if base.TemporalityPreference != "" {
		m["temporality_preference"] = base.TemporalityPreference
	}
	return m
}

func getGlobalConfig(taGlobalConfig v1beta1.AnyConfig, collectorConfig v1beta1.Config) (map[string]any, error) {
	// global config from the target allocator has priority
	if len(taGlobalConfig.Object) > 0 {
		return taGlobalConfig.Object, nil
	}

	collectorGlobalConfig, err := getGlobalConfigFromOtelConfig(collectorConfig)
	if err != nil {
		return nil, err
	}
	return collectorGlobalConfig.Object, nil
}

func getScrapeConfigs(taScrapeConfigs []v1beta1.AnyConfig, collectorConfig v1beta1.Config) ([]v1beta1.AnyConfig, error) {
	scrapeConfigs := []v1beta1.AnyConfig{}

	// we take scrape configs from both the target allocator spec and the collector config
	if len(taScrapeConfigs) > 0 {
		scrapeConfigs = append(scrapeConfigs, taScrapeConfigs...)
	}

	configStr, err := collectorConfig.Yaml()
	if err != nil {
		return nil, err
	}

	collectorScrapeConfigs, err := getScrapeConfigsFromOtelConfig(configStr)
	if err != nil {
		return nil, err
	}

	return append(scrapeConfigs, collectorScrapeConfigs...), nil
}

func getGlobalConfigFromOtelConfig(otelConfig v1beta1.Config) (v1beta1.AnyConfig, error) {
	// TODO: Eventually we should figure out a way to pull this in to the main specification for the TA
	type promReceiverConfig struct {
		Prometheus struct {
			Config struct {
				Global map[string]any `mapstructure:"global"`
			} `mapstructure:"config"`
		} `mapstructure:"prometheus"`
	}
	decodedConfig := &promReceiverConfig{}
	if err := mapstructure.Decode(otelConfig.Receivers.Object, decodedConfig); err != nil {
		return v1beta1.AnyConfig{}, err
	}
	return v1beta1.AnyConfig{
		Object: decodedConfig.Prometheus.Config.Global,
	}, nil
}

func getScrapeConfigsFromOtelConfig(otelcolConfig string) ([]v1beta1.AnyConfig, error) {
	// Collector supports environment variable substitution, but the TA does not.
	// TA Scrape Configs should have a single "$", as it does not support env var substitution
	promConfig, err := adapters.ConfigToPromConfig(otelcolConfig)
	if err != nil {
		return nil, err
	}
	if _, hasConfig := promConfig["config"]; !hasConfig {
		return []v1beta1.AnyConfig{}, nil
	}
	prometheusReceiverConfig, err := adapters.UnescapeDollarSignsInPromConfig(otelcolConfig)
	if err != nil {
		return nil, err
	}

	scrapeConfigs, err := adapters.GetScrapeConfigsFromPromConfig(prometheusReceiverConfig)
	if err != nil {
		return nil, err
	}

	v1beta1scrapeConfigs := make([]v1beta1.AnyConfig, len(scrapeConfigs))

	for i, config := range scrapeConfigs {
		v1beta1scrapeConfigs[i] = v1beta1.AnyConfig{Object: config}
	}

	return v1beta1scrapeConfigs, nil
}
