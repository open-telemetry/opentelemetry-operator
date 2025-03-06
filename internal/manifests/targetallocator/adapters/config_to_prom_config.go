// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

type TAOption func(targetAllocatorCfg map[interface{}]interface{}) error

func errorNoComponent(component string) error {
	return fmt.Errorf("no %s available as part of the configuration", component)
}

func errorNotAMapAtIndex(component string, index int) error {
	return fmt.Errorf("index %d: %s property in the configuration doesn't contain a valid map: %s", index, component, component)
}

func errorNotAMap(component string) error {
	return fmt.Errorf("%s property in the configuration doesn't contain valid %s", component, component)
}

func errorNotAList(component string) error {
	return fmt.Errorf("%s must be a list in the config", component)
}

func errorNotAListAtIndex(component string, index int) error {
	return fmt.Errorf("index %d: %s property in the configuration doesn't contain a valid index: %s", index, component, component)
}

func errorNotAStringAtIndex(component string, index int) error {
	return fmt.Errorf("index %d: %s property in the configuration doesn't contain a valid string: %s", index, component, component)
}

// GetScrapeConfigsFromPromConfig extracts the scrapeConfig array from prometheus receiver config.
func GetScrapeConfigsFromPromConfig(promReceiverConfig map[interface{}]interface{}) ([]map[string]interface{}, error) {
	prometheusConfigProperty, ok := promReceiverConfig["config"]
	if !ok {
		return nil, errorNoComponent("prometheusConfig")
	}

	prometheusConfig, ok := prometheusConfigProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("prometheusConfig")
	}

	scrapeConfigsProperty, ok := prometheusConfig["scrape_configs"]
	if !ok {
		return nil, errorNoComponent("scrape_configs")
	}

	scrapeConfigs, ok := scrapeConfigsProperty.([]interface{})
	if !ok {
		return nil, errorNotAList("scrape_configs")
	}

	scrapeConfigMaps := make([]map[string]interface{}, len(scrapeConfigs))
	for i, scrapeConfig := range scrapeConfigs {
		scrapeConfigMapInterface, ok := scrapeConfig.(map[interface{}]interface{})
		if !ok {
			return nil, errorNotAMap("scrape_config")
		}
		scrapeConfigMap := make(map[string]interface{})
		for k, v := range scrapeConfigMapInterface {
			k, ok := k.(string)
			if !ok {
				return nil, errorNotAMap("scrape_config")
			}
			scrapeConfigMap[k] = v
		}
		scrapeConfigMaps[i] = scrapeConfigMap
	}

	return scrapeConfigMaps, nil
}

// ConfigToPromConfig converts the incoming configuration object into the Prometheus receiver config.
func ConfigToPromConfig(cfg string) (map[interface{}]interface{}, error) {
	config, err := adapters.ConfigFromString(cfg)
	if err != nil {
		return nil, err
	}

	receiversProperty, ok := config["receivers"]
	if !ok {
		return nil, errorNoComponent("receivers")
	}

	receivers, ok := receiversProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("receivers")
	}

	prometheusProperty, ok := receivers["prometheus"]
	if !ok {
		return nil, errorNoComponent("prometheus")
	}

	prometheus, ok := prometheusProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("prometheus")
	}

	return prometheus, nil
}

// UnescapeDollarSignsInPromConfig replaces "$$" with "$" in the "replacement" fields of
// both "relabel_configs" and "metric_relabel_configs" in a Prometheus configuration file.
func UnescapeDollarSignsInPromConfig(cfg string) (map[interface{}]interface{}, error) {
	prometheus, err := ConfigToPromConfig(cfg)
	if err != nil {
		return nil, err
	}

	scrapeConfigs, err := GetScrapeConfigsFromPromConfig(prometheus)
	if err != nil {
		return nil, err
	}

	for i, scrapeConfig := range scrapeConfigs {

		relabelConfigsProperty, found := scrapeConfig["relabel_configs"]
		if found {
			relabelConfigs, ok := relabelConfigsProperty.([]interface{})
			if !ok {
				return nil, errorNotAListAtIndex("relabel_configs", i)
			}

			for i, rc := range relabelConfigs {
				relabelConfig, rcErr := rc.(map[interface{}]interface{})
				if !rcErr {
					return nil, errorNotAMapAtIndex("relabel_config", i)
				}

				replacementProperty, rcErr := relabelConfig["replacement"]
				if !rcErr {
					continue
				}

				replacement, rcErr := replacementProperty.(string)
				if !rcErr {
					return nil, errorNotAStringAtIndex("replacement", i)
				}

				relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "$")
			}
		}

		metricRelabelConfigsProperty, found := scrapeConfig["metric_relabel_configs"]
		if found {
			metricRelabelConfigs, ok := metricRelabelConfigsProperty.([]interface{})
			if !ok {
				return nil, errorNotAListAtIndex("metric_relabel_configs", i)
			}

			for i, rc := range metricRelabelConfigs {
				relabelConfig, ok := rc.(map[interface{}]interface{})
				if !ok {
					return nil, errorNotAMapAtIndex("metric_relabel_config", i)
				}

				replacementProperty, ok := relabelConfig["replacement"]
				if !ok {
					continue
				}

				replacement, ok := replacementProperty.(string)
				if !ok {
					return nil, errorNotAStringAtIndex("replacement", i)
				}

				relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "$")
			}
		}
	}

	return prometheus, nil
}

// AddHTTPSDConfigToPromConfig adds HTTP SD (Service Discovery) configuration to the Prometheus configuration.
// This function removes any existing service discovery configurations (e.g., `sd_configs`, `dns_sd_configs`, `file_sd_configs`, etc.)
// from the `scrape_configs` section and adds a single `http_sd_configs` configuration.
// The `http_sd_configs` points to the TA (Target Allocator) endpoint that provides the list of targets for the given job.
func AddHTTPSDConfigToPromConfig(prometheus map[interface{}]interface{}, taServiceName string) (map[interface{}]interface{}, error) {
	prometheusConfigProperty, ok := prometheus["config"]
	if !ok {
		return nil, errorNoComponent("prometheusConfig")
	}

	prometheusConfig, ok := prometheusConfigProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("prometheusConfig")
	}

	scrapeConfigsProperty, ok := prometheusConfig["scrape_configs"]
	if !ok {
		return nil, errorNoComponent("scrape_configs")
	}

	scrapeConfigs, ok := scrapeConfigsProperty.([]interface{})
	if !ok {
		return nil, errorNotAList("scrape_configs")
	}

	sdRegex := regexp.MustCompile(`^.*(sd|static)_configs$`)

	for i, config := range scrapeConfigs {
		scrapeConfig, ok := config.(map[interface{}]interface{})
		if !ok {
			return nil, errorNotAMapAtIndex("scrape_config", i)
		}

		// Check for other types of service discovery configs (e.g. dns_sd_configs, file_sd_configs, etc.)
		for key := range scrapeConfig {
			keyStr, keyErr := key.(string)
			if !keyErr {
				continue
			}
			if sdRegex.MatchString(keyStr) {
				delete(scrapeConfig, key)
			}
		}

		jobNameProperty, ok := scrapeConfig["job_name"]
		if !ok {
			return nil, errorNotAStringAtIndex("job_name", i)
		}

		jobName, ok := jobNameProperty.(string)
		if !ok {
			return nil, errorNotAStringAtIndex("job_name is not a string", i)
		}

		escapedJob := url.QueryEscape(jobName)
		scrapeConfig["http_sd_configs"] = []interface{}{
			map[string]interface{}{
				"url": fmt.Sprintf("http://%s:80/jobs/%s/targets?collector_id=$POD_NAME", taServiceName, escapedJob),
			},
		}
	}

	return prometheus, nil
}

func WithTLSConfig(caFile, certFile, keyFile, taServiceName string) TAOption {
	return func(targetAllocatorCfg map[interface{}]interface{}) error {
		if _, exists := targetAllocatorCfg["tls"]; !exists {
			targetAllocatorCfg["tls"] = make(map[interface{}]interface{})
		}

		tlsCfg, ok := targetAllocatorCfg["tls"].(map[interface{}]interface{})
		if !ok {
			return errorNotAMap("tls")
		}

		tlsCfg["ca_file"] = caFile
		tlsCfg["cert_file"] = certFile
		tlsCfg["key_file"] = keyFile

		targetAllocatorCfg["endpoint"] = fmt.Sprintf("https://%s:443", taServiceName)

		return nil
	}
}

// AddTAConfigToPromConfig adds or updates the target_allocator configuration in the Prometheus configuration.
// If the `EnableTargetAllocatorRewrite` feature flag for the target allocator is enabled, this function
// removes the existing scrape_configs from the collector's Prometheus configuration as it's not required.
func AddTAConfigToPromConfig(prometheus map[interface{}]interface{}, taServiceName string, taOpts ...TAOption) (map[interface{}]interface{}, error) {
	prometheusConfigProperty, ok := prometheus["config"]
	if !ok {
		return nil, errorNoComponent("prometheusConfig")
	}

	prometheusCfg, ok := prometheusConfigProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("prometheusConfig")
	}

	// Create the TargetAllocConfig dynamically if it doesn't exist
	if prometheus["target_allocator"] == nil {
		prometheus["target_allocator"] = make(map[interface{}]interface{})
	}

	targetAllocatorCfg, ok := prometheus["target_allocator"].(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("target_allocator")
	}

	targetAllocatorCfg["endpoint"] = fmt.Sprintf("http://%s:80", taServiceName)
	targetAllocatorCfg["interval"] = "30s"
	targetAllocatorCfg["collector_id"] = "${POD_NAME}"

	for _, opt := range taOpts {
		err := opt(targetAllocatorCfg)
		if err != nil {
			return nil, err
		}
	}

	// Remove the scrape_configs key from the map
	delete(prometheusCfg, "scrape_configs")

	return prometheus, nil
}

// ValidatePromConfig checks if the prometheus receiver config is valid given other collector-level settings.
func ValidatePromConfig(config map[interface{}]interface{}, targetAllocatorEnabled bool) error {
	// TODO: Rethink this validation, now that target allocator rewrite is enabled permanently.

	_, promConfigExists := config["config"]

	if targetAllocatorEnabled {
		return nil
	}
	// if target allocator isn't enabled, we need a config section
	if !promConfigExists {
		return errorNoComponent("prometheusConfig")
	}

	return nil
}

// ValidateTargetAllocatorConfig checks if the Target Allocator config is valid
// In order for Target Allocator to do anything useful, at least one of the following has to be true:
//   - at least one scrape config has to be defined in Prometheus receiver configuration
//   - PrometheusCR has to be enabled in target allocator settings
func ValidateTargetAllocatorConfig(targetAllocatorPrometheusCR bool, promReceiverConfig map[interface{}]interface{}) error {

	if targetAllocatorPrometheusCR {
		return nil
	}
	// if PrometheusCR isn't enabled, we need at least one scrape config
	scrapeConfigs, err := GetScrapeConfigsFromPromConfig(promReceiverConfig)
	if err != nil {
		return err
	}

	if len(scrapeConfigs) == 0 {
		return fmt.Errorf("either at least one scrape config needs to be defined or PrometheusCR needs to be enabled")
	}

	return nil
}
