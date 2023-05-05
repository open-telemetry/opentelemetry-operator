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

package reconcile

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/http"
	_ "github.com/prometheus/prometheus/discovery/install" // Package install has the side-effect of registering all builtin.
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

type Config struct {
	PromConfig *promconfig.Config `yaml:"config"`
}

func ReplaceConfig(instance v1alpha1.OpenTelemetryCollector) (string, error) {
	// Check if TargetAllocator is enabled, if not, return the original config
	if !instance.Spec.TargetAllocator.Enabled {
		return instance.Spec.Config, nil
	}

	config, getStringErr := adapters.ConfigFromString(instance.Spec.Config)
	if getStringErr != nil {
		return "", getStringErr
	}

	// Get the Prometheus config map with dollar signs replaced to ensure successful validation of Prometheus regex
	promCfgMap, getCfgPromErr := ta.ReplaceDollarSignInPromConfig(instance.Spec.Config)
	if getCfgPromErr != nil {
		return "", getCfgPromErr
	}

	// yaml marshaling/unsmarshaling is preferred because of the problems associated with the conversion of map to a struct using mapstructure
	promCfg, marshalErr := yaml.Marshal(map[string]interface{}{
		"config": promCfgMap,
	})
	if marshalErr != nil {
		return "", marshalErr
	}

	var cfg Config
	if marshalErr = yaml.UnmarshalStrict(promCfg, &cfg); marshalErr != nil {
		return "", fmt.Errorf("error unmarshaling YAML: %w", marshalErr)
	}

	// Escapes default replacement key added by prometheus and resets all other replacement keys to their original values
	replacer := strings.NewReplacer(
		"$", "$$",
		"__DOUBLE_DOLLAR__", "$$",
		"__SINGLE_DOLLAR__", "$",
	)

	// Loop through the scrape configs and escape "$" characters in relabel_configs and metric_relabel_configs
	for _, scrapeCfg := range cfg.PromConfig.ScrapeConfigs {
		for _, relabelCfg := range scrapeCfg.RelabelConfigs {
			relabelCfg.Replacement = replacer.Replace(relabelCfg.Replacement)
		}
		for _, metricRelabelCfg := range scrapeCfg.MetricRelabelConfigs {
			metricRelabelCfg.Replacement = replacer.Replace(metricRelabelCfg.Replacement)
		}
		// Escape the job name for use in the URL
		escapedJob := url.QueryEscape(scrapeCfg.JobName)
		// Create the service discovery configuration with the formatted URL
		sdConfig := &http.SDConfig{
			URL: fmt.Sprintf("http://%s:80/jobs/%s/targets?collector_id=$POD_NAME", naming.TAService(instance), escapedJob),
		}
		scrapeCfg.ServiceDiscoveryConfigs = discovery.Configs{sdConfig}
	}

	// Decode the Config struct back to a map and update the Prometheus config map
	updPromCfgMap := make(map[string]interface{})
	if err := mapstructure.Decode(cfg, &updPromCfgMap); err != nil {
		return "", err
	}

	// Update the OpenTelemetryCollector instance's config with the updated Prometheus config map
	// type coercion checks are handled in the ReplaceDollarSignInPromConfig method above
	config["receivers"].(map[interface{}]interface{})["prometheus"].(map[interface{}]interface{})["config"] = updPromCfgMap["PromConfig"]

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
