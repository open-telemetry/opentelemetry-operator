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
)

func ErrorNoComponent(component string) string {
	return fmt.Sprintf("no %s available as part of the configuration", component)
}

func ErrorNotAMap(component string) string {
	return fmt.Sprintf("%s property in the configuration doesn't contain valid %s", component, component)
}

// ConfigToPromConfig converts the incoming configuration object into a the Prometheus receiver config.
func ConfigToPromConfig(config map[interface{}]interface{}) (map[interface{}]interface{}, string) {
	receiversProperty, ok := config["receivers"]
	if !ok {
		return nil, ErrorNoComponent("receivers")
	}

	receivers, ok := receiversProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrorNotAMap("receivers")
	}

	prometheusProperty, ok := receivers["prometheus"]
	if !ok {
		return nil, ErrorNoComponent("prometheus")
	}

	prometheus, ok := prometheusProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrorNotAMap("prometheus")
	}

	prometheusConfigProperty, ok := prometheus["config"]
	if !ok {
		return nil, ErrorNoComponent("prometheusConfig")
	}

	prometheusConfig, ok := prometheusConfigProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrorNotAMap("prometheusConfig")
	}

	return prometheusConfig, ""
}
