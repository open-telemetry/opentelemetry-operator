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

	"github.com/go-logr/logr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/authz"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/parser/processor"
)

// ConfigToRBAC parses the OpenTelemetry Collector configuration and checks what RBAC resources are needed to be created.
func ConfigToRBAC(logger logr.Logger, config map[any]any) []authz.DynamicRolePolicy {
	policyRules := configToRBACForComponentType(logger, config, ComponentTypeProcessor)
	policyRules = append(policyRules, configToRBACForComponentType(logger, config, ComponentTypeExporter)...)
	return policyRules
}

func configToRBACForComponentType(logger logr.Logger, config map[any]any, cType ComponentType) []authz.DynamicRolePolicy {
	var policyRules []authz.DynamicRolePolicy
	componentsRaw, ok := config[cType.Plural()]
	if !ok {
		logger.V(2).Info(fmt.Sprintf("no %s available as part of the configuration", cType.String()))
		return policyRules
	}

	components, ok := componentsRaw.(map[any]any)
	if !ok {
		logger.V(2).Info(fmt.Sprintf("%s doesn't contain valid components", cType.String()))
		return policyRules
	}

	enabledProcessors := getEnabledComponents(config, cType)

	for key, val := range components {
		if !enabledProcessors[key] {
			continue
		}

		componentCfg, ok := val.(map[any]any)
		if !ok {
			logger.V(2).Info(fmt.Sprintf("%s doesn't seem to be a map of properties", cType.String()), "component", key)
			componentCfg = map[any]any{}
		}

		componentName := key.(string)
		componentParser, err := processor.For(logger, componentName, componentCfg)
		if err != nil {
			logger.V(2).Info(fmt.Sprintf("no parser found for %s", cType.String()), "component", componentName)
			continue
		}

		policyRules = append(policyRules, componentParser.GetRBACRules()...)
	}

	return policyRules
}
