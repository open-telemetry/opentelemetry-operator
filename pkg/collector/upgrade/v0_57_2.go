// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_57_2(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {

	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	otelCfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.57.2, failed to parse configuration: %w", err)
	}

	//Remove deprecated port field from config. (https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/10853)
	extensionsConfig, ok := otelCfg["extensions"].(map[interface{}]interface{})
	if !ok {
		// In case there is no extensions config.
		return otelcol, nil
	}

	for keyExt, valExt := range extensionsConfig {
		if strings.HasPrefix(keyExt.(string), "health_check") {
			switch extensions := valExt.(type) {
			case map[interface{}]interface{}:
				if port, ok := extensions["port"]; ok {
					endpointV := extensions["endpoint"]
					extensions["endpoint"] = fmt.Sprintf("%s:%s", endpointV, port)
					delete(extensions, "port")

					otelCfg["extensions"] = extensionsConfig
					res, err := yaml.Marshal(otelCfg)
					if err != nil {
						return otelcol, fmt.Errorf("couldn't upgrade to v0.57.2, failed to marshall back configuration: %w", err)
					}

					otelcol.Spec.Config = string(res)
					u.Recorder.Event(otelcol, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.57.2 has deprecated port for healthcheck extension %q", keyExt))
				}
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.57.2, the extension %q is invalid (expected string or map but was %t)", keyExt, valExt)
			}
		}
	}
	return otelcol, nil
}
