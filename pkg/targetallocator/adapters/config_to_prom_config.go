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

package adapters

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func errorNoComponent(component string) error {
	return fmt.Errorf("no %s available as part of the configuration", component)
}

func errorNotAMap(component string) error {
	return fmt.Errorf("%s property in the configuration doesn't contain valid %s", component, component)
}

func errorNotAList(component string) error {
	return fmt.Errorf("%s must be a list in the config", component)
}

func errorNotAString(component string) error {
	return fmt.Errorf("%s must be a string", component)
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

// extractScrapeConfigs takes a Prometheus config as a map and returns a list of the "scrape_configs" sections.
func extractScrapeConfigs(prometheus map[interface{}]interface{}) ([]interface{}, error) {
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

	return scrapeConfigs, nil
}

// UnescapeDollarSignsInPromConfig replaces "$$" with "$" in the "replacement" fields of
// both "relabel_configs" and "metric_relabel_configs" in a Prometheus configuration file.
func UnescapeDollarSignsInPromConfig(cfg string) (map[interface{}]interface{}, error) {
	prometheus, err := ConfigToPromConfig(cfg)
	if err != nil {
		return nil, err
	}

	scrapeConfigs, err := extractScrapeConfigs(prometheus)
	if err != nil {
		return nil, err
	}

	for _, config := range scrapeConfigs {
		scrapeConfig, ok := config.(map[interface{}]interface{})
		if !ok {
			return nil, errorNotAMap("scrape_config")
		}

		relabelConfigsProperty, ok := scrapeConfig["relabel_configs"]
		if !ok {
			continue
		}

		relabelConfigs, ok := relabelConfigsProperty.([]interface{})
		if !ok {
			return nil, errorNotAList("relabel_configs")
		}

		for _, rc := range relabelConfigs {
			relabelConfig, rcok := rc.(map[interface{}]interface{})
			if !rcok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, rcok := relabelConfig["replacement"]
			if !rcok {
				continue
			}

			replacement, rcok := replacementProperty.(string)
			if !rcok {
				return nil, errorNotAString("replacement")
			}

			relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "$")
		}

		metricRelabelConfigsProperty, ok := scrapeConfig["metric_relabel_configs"]
		if !ok {
			continue
		}

		metricRelabelConfigs, ok := metricRelabelConfigsProperty.([]interface{})
		if !ok {
			return nil, errorNotAList("metric_relabel_configs")
		}

		for _, rc := range metricRelabelConfigs {
			relabelConfig, mrcok := rc.(map[interface{}]interface{})
			if !mrcok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, mrcok := relabelConfig["replacement"]
			if !mrcok {
				continue
			}

			replacement, mrcok := replacementProperty.(string)
			if !mrcok {
				return nil, errorNotAString("replacement")
			}

			relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "$")
		}
	}

	return prometheus, nil
}

// AddHTTPSDConfigToPromConfig takes a Prometheus configuration in YAML format and
// adds service discovery configurations for each scrape job that specifies a "job_name"
// property. The service discovery configuration is added as an "http_sd_configs" property,
// with the URL pointing to a service endpoint that provides the list of targets for the
// given job.
func AddHTTPSDConfigToPromConfig(cfg string, taServiceName string) (map[interface{}]interface{}, error) {
	prometheus, err := ConfigToPromConfig(cfg)
	if err != nil {
		return nil, err
	}

	scrapeConfigs, err := extractScrapeConfigs(prometheus)
	if err != nil {
		return nil, err
	}

	for _, config := range scrapeConfigs {
		scrapeConfig, ok := config.(map[interface{}]interface{})
		if !ok {
			return nil, errorNotAMap("scrape_config")
		}

		jobNameProperty, ok := scrapeConfig["job_name"]
		if !ok {
			return nil, errorNotAString("job_name")
		}

		jobName, ok := jobNameProperty.(string)
		if !ok {
			return nil, errorNotAString("job_name is not a string")
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
