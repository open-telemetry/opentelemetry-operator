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
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/processor"
)

// ConfigToRBAC parses the OpenTelemetry Collector configuration and checks what RBAC resources are needed to be created.
func ConfigToRBAC(logger logr.Logger, config map[interface{}]interface{}) []rbacv1.PolicyRule {
	var policyRules []rbacv1.PolicyRule
	processorsRaw, ok := config["processors"]
	if !ok {
		logger.V(2).Info("no processors available as part of the configuration")
		return policyRules
	}

	processors, ok := processorsRaw.(map[interface{}]interface{})
	if !ok {
		logger.V(2).Info("processors doesn't contain valid components")
		return policyRules
	}

	enabledProcessors := getEnabledComponents(config, ComponentTypeProcessor)

	for key, val := range processors {
		if !enabledProcessors[key] {
			continue
		}

		processorCfg, ok := val.(map[interface{}]interface{})
		if !ok {
			logger.V(2).Info("processor doesn't seem to be a map of properties", "processor", key)
			processorCfg = map[interface{}]interface{}{}
		}

		processorName := key.(string)
		processorParser, err := processor.For(logger, processorName, processorCfg)
		if err != nil {
			logger.V(2).Info("no parser found for", "processor", processorName)
			continue
		}

		policyRules = append(policyRules, processorParser.GetRBACRules()...)
	}

	return policyRules
}
