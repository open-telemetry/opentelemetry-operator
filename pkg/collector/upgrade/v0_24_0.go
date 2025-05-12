// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_24_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.24.0, failed to parse configuration: %w", err)
	}

	extensions, ok := cfg["extensions"].(map[interface{}]interface{})
	if !ok {
		// We do not need an upgrade if there are no extensions.
		return otelcol, nil
	}

	for k, v := range extensions {
		if strings.HasPrefix(k.(string), "health_check") {
			switch extension := v.(type) {
			case map[interface{}]interface{}:
				if port, ok := extension["port"]; ok {
					delete(extension, "port")
					extension["endpoint"] = fmt.Sprintf("0.0.0.0:%d", port)
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.24.0 migrated the property 'port' to 'endpoint' for extension %q", k))
				}
			case string:
				if len(extension) == 0 {
					// This extension is using the default configuration.
					continue
				}
			case nil:
				// This extension is using the default configuration.
				continue
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.24.0, the extension %q is invalid (expected string or map but was %t)", k, v)
			}
		}
	}

	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.24.0, failed to marshall back configuration: %w", err)
	}

	otelcol.Spec.Config = string(res)
	return otelcol, nil
}
