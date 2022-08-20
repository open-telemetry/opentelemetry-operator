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

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func errorNoComponent(component string) error {
	return fmt.Errorf("no %s available as part of the configuration", component)
}

func errorNotAMap(component string) error {
	return fmt.Errorf("%s property in the configuration doesn't contain valid %s", component, component)
}

func configToPromReceiverConfig(cfg string) (map[interface{}]interface{}, error) {
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

// ConfigToPromConfig converts the incoming configuration object into a the Prometheus receiver config.
func ConfigToPromConfig(cfg string) (map[interface{}]interface{}, error) {
	prometheus, err := configToPromReceiverConfig(cfg)
	if err != nil {
		return nil, err
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

// ConfigToCollectorTAConfig converts the incoming configuration object into the allocator client config.
func ConfigToCollectorTAConfig(cfg string) (map[interface{}]interface{}, error) {
	prometheus, err := configToPromReceiverConfig(cfg)
	if err != nil {
		return nil, err
	}

	targetAllocatorProperty, ok := prometheus["target_allocator"]
	if !ok {
		return nil, nil // this is expected if the user does not configure any target_allocator configs
	}
	targetAllocatorCfg, ok := targetAllocatorProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errorNotAMap("target_allocator")
	}

	return targetAllocatorCfg, nil
}
