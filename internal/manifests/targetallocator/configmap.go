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
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
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

	taConfig := make(map[interface{}]interface{})
	// Set config if global or scrape configs set
	config := map[string]interface{}{}
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
		prometheusCRConfig := map[interface{}]interface{}{
			"enabled": true,
		}
		if taSpec.PrometheusCR.ScrapeInterval.Size() > 0 {
			prometheusCRConfig["scrape_interval"] = taSpec.PrometheusCR.ScrapeInterval.Duration
		}

		prometheusCRConfig["service_monitor_selector"] = taSpec.PrometheusCR.ServiceMonitorSelector

		prometheusCRConfig["pod_monitor_selector"] = taSpec.PrometheusCR.PodMonitorSelector

		prometheusCRConfig["scrape_config_selector"] = taSpec.PrometheusCR.ScrapeConfigSelector

		prometheusCRConfig["probe_selector"] = taSpec.PrometheusCR.ProbeSelector

		taConfig["prometheus_cr"] = prometheusCRConfig
	}

	if params.Config.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		taConfig["https"] = map[string]interface{}{
			"enabled":            true,
			"listen_addr":        ":8443",
			"ca_file_path":       filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorCAFileName),
			"tls_cert_file_path": filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSCertFileName),
			"tls_key_file_path":  filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSKeyFileName),
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
			Annotations: instance.Annotations,
		},
		Data: map[string]string{
			targetAllocatorFilename: string(taConfigYAML),
		},
	}, nil
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
				Global map[string]interface{} `mapstructure:"global"`
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
