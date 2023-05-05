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

// ConfigToPromConfig converts the incoming configuration object into a the Prometheus receiver config.
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

	prometheusConfigProperty, ok := prometheus["config"]
	if !ok {
		return nil, errorNoComponent("prometheusConfig")
	}

	prometheusConfig, ok := prometheusConfigProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("prometheusConfig")
	}

	return prometheusConfig, nil
}

// ReplaceDollarSignInPromConfig replaces "$$" with "__DOUBLE_DOLLAR__" and "$" with "__SINGLE_DOLLAR__" in
// the "replacement" fields of both "relabel_configs" and "metric_relabel_configs" in a Prometheus configuration file.
func ReplaceDollarSignInPromConfig(cfg string) (map[interface{}]interface{}, error) {
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
			relabelConfig, ok := rc.(map[interface{}]interface{})
			if !ok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, ok := relabelConfig["replacement"]
			if !ok {
				continue
			}

			replacement, ok := replacementProperty.(string)
			if !ok {
				return nil, errorNotAString("replacement")
			}

			relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "__DOUBLE_DOLLAR__")
			relabelConfig["replacement"] = strings.ReplaceAll(relabelConfig["replacement"].(string), "$", "__SINGLE_DOLLAR__")
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
			relabelConfig, ok := rc.(map[interface{}]interface{})
			if !ok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, ok := relabelConfig["replacement"]
			if !ok {
				continue
			}

			replacement, ok := replacementProperty.(string)
			if !ok {
				return nil, errorNotAString("replacement")
			}

			relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "__DOUBLE_DOLLAR__")
			relabelConfig["replacement"] = strings.ReplaceAll(relabelConfig["replacement"].(string), "$", "__SINGLE_DOLLAR__")
		}
	}

	return prometheusConfig, nil
}

// ReplaceDollarSignInTAPromConfig replaces "$$" with "$" in the "replacement" fields of
// both "relabel_configs" and "metric_relabel_configs" in a Prometheus configuration file.
func ReplaceDollarSignInTAPromConfig(cfg string) (map[interface{}]interface{}, error) {
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
			relabelConfig, ok := rc.(map[interface{}]interface{})
			if !ok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, ok := relabelConfig["replacement"]
			if !ok {
				continue
			}

			replacement, ok := replacementProperty.(string)
			if !ok {
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
			relabelConfig, ok := rc.(map[interface{}]interface{})
			if !ok {
				return nil, errorNotAMap("relabel_config")
			}

			replacementProperty, ok := relabelConfig["replacement"]
			if !ok {
				continue
			}

			replacement, ok := replacementProperty.(string)
			if !ok {
				return nil, errorNotAString("replacement")
			}

			relabelConfig["replacement"] = strings.ReplaceAll(replacement, "$$", "$")
		}
	}

	return prometheusConfig, nil
}
