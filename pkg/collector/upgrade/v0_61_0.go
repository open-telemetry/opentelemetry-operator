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

package upgrade

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func upgrade0_61_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	otelCfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.61.0, failed to parse configuration: %w", err)
	}

	// Search for removed Jaeger remote sampling settings. (https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/14163)
	receiversConfig, ok := otelCfg["receivers"].(map[any]any)
	if !ok {
		// In case there is no extensions config.
		return otelcol, nil
	}

	jaegerRec := make(map[any]any, 0)
	for key, rc := range receiversConfig {
		k, ok := key.(string)
		if !ok {
			continue
		}
		cfg, ok := rc.(map[any]any)
		// check if jaeger is configured
		if !ok || !strings.HasPrefix(k, "jaeger") {
			continue
		}
		// check if remote sampling settings exit
		rs, ok := cfg["remote_sampling"]
		if !ok {
			continue
		}
		jaegerRec[k] = rs
	}

	if len(jaegerRec) == 0 {
		// nothing to do
		return otelcol, nil
	}

	extensionsConfig, ok := otelCfg["extensions"].(map[any]any)
	if !ok {
		// In case there is no extensions config.
		extensionsConfig = make(map[any]any)
	}

	var jaegerExtenions []string
	for name, oldCfg := range jaegerRec {
		recName, ok := name.(string)
		if !ok {
			continue
		}
		extName := "jaegerremotesampling"
		split := strings.Split(recName, "/")
		if len(split) > 2 {
			return nil, fmt.Errorf("couldn't upgrade to v0.61.0, failed to define extension name: %w", err)
		} else if len(split) == 2 {
			extName = fmt.Sprintf("%s/%s", extName, split[1])
		}
		// Configure new Jaeger remote sampling extension. (https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/6510)
		newCfg, err := upgradeRemoteSamplingConfigTo0_61_0(oldCfg)
		if err != nil {
			return nil, err
		}
		extensionsConfig[extName] = newCfg
		jaegerExtenions = append(jaegerExtenions, extName)
	}

	otelCfg["extensions"] = extensionsConfig
	serviceConfig, ok := otelCfg["service"].(map[any]any)
	if !ok {
		// In case there is no extensions config.
		serviceConfig = make(map[any]any)
	}

	extensionConfig, ok := serviceConfig["extensions"].([]any)
	if !ok {
		extensionConfig = make([]any, 0)
	}
	for _, name := range jaegerExtenions {
		extensionConfig = append(extensionConfig, name)
	}
	serviceConfig["extensions"] = extensionConfig
	otelCfg["service"] = serviceConfig

	res, err := yaml.Marshal(otelCfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.61.0, failed to marshall back configuration: %w", err)
	}
	otelcol.Spec.Config = string(res)
	keys := make([]string, 0, len(jaegerRec))
	for kk := range jaegerRec {
		if k, ok := kk.(string); ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	existing := &corev1.ConfigMap{}
	updated := existing.DeepCopy()
	u.Recorder.Event(updated, "Normal", "Upgrade",
		fmt.Sprintf("upgrade to v0.61.0 replaced the deprecated jaeger receiver settings with "+
			"jaegerremotesampling extension settings. Affected receivers are: %v", keys),
	)
	fmt.Println(otelcol.Spec.Config)
	return otelcol, nil
}

func upgradeRemoteSamplingConfigTo0_61_0(input any) (any, error) {
	origin, ok := input.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("couldn't upgrade to v0.61.0, failed to convert receiver: %v", origin)
	}
	newCfg := make(map[any]map[any]any)
	if _, ok := newCfg["source"]; !ok {
		newCfg["source"] = make(map[any]any)
	}
	newCfg["source"]["remote"] = map[any]any{"endpoint": origin["endpoint"]}
	newCfg["source"]["tls"] = origin["tls"]
	newCfg["source"]["reload_interval"] = origin["strategy_file"]
	newCfg["source"]["file"] = origin["strategy_file_reload_interval"]

	delete(origin, "endpoint")
	delete(origin, "tls")
	delete(origin, "strategy_file")
	delete(origin, "strategy_file_reload_interval")

	return origin, nil
}
