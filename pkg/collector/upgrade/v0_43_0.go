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

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func upgrade0_43_0(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// return if args exist
	if len(otelcol.Spec.Args) == 0 {
		return otelcol, nil
	}

	foundMetricsArgs := make(map[string]string)
	for argKey, argValue := range otelcol.Spec.Args {
		if argKey == "--metrics-addr" || argKey == "--metrics-level" {
			foundMetricsArgs[argKey] = argValue
			delete(otelcol.Spec.Args, argKey)
		}
	}

	if len(foundMetricsArgs) > 0 {
		cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
		if err != nil {
			return otelcol, fmt.Errorf("couldn't upgrade to v0.43.0, failed to parse configuration: %w", err)
		}

		serviceConfig, ok := cfg["service"].(map[interface{}]interface{})
		if !ok {

			cfg["service"] = make(map[interface{}]interface{})
			serviceConfig, _ = cfg["service"].(map[interface{}]interface{})
		}

		telemetryConfig, ok := serviceConfig["telemetry"].(map[interface{}]interface{})
		if !ok {
			serviceConfig["telemetry"] = make(map[interface{}]interface{})
			telemetryConfig, _ = serviceConfig["telemetry"].(map[interface{}]interface{})
		}

		metricsConfig, ok := telemetryConfig["metrics"].(map[interface{}]interface{})
		if !ok {
			telemetryConfig["metrics"] = make(map[interface{}]interface{})
			metricsConfig, _ = telemetryConfig["metrics"].(map[interface{}]interface{})
		}

		if len(metricsConfig) == 0 {
			if val, ok := foundMetricsArgs["--metrics-addr"]; ok {
				metricsConfig["address"] = val
			}
			if val, ok := foundMetricsArgs["--metrics-level"]; ok {
				metricsConfig["level"] = val
			}
		}
		cfg["service"] = serviceConfig
		res, err := yaml.Marshal(cfg)

		if err != nil {
			return otelcol, fmt.Errorf("couldn't upgrade to v0.43.0, failed to marshall back configuration: %w", err)
		}

		otelcol.Spec.Config = string(res)

		keys := make([]string, 0, len(foundMetricsArgs))

		for k := range foundMetricsArgs {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		otelcol.Status.Messages = append(otelcol.Status.Messages, fmt.Sprintf("upgrade to v0.43.0 dropped the deprecated metrics arguments "+"i.e. %v from otelcol custom resource otelcol.spec.args and adding them to otelcol.spec.config.service.telemetry.metrics, if no metrics arguments are configured already.", keys))
	}

	return otelcol, nil
}
