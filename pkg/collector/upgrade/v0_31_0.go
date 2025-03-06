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

func upgrade0_31_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.31.0, failed to parse configuration: %w", err)
	}

	receivers, ok := cfg["receivers"].(map[interface{}]interface{})
	if !ok {
		// no receivers? no need to fail because of that
		return otelcol, nil
	}

	for k, v := range receivers {
		// from the changelog https://github.com/open-telemetry/opentelemetry-collector/blob/main/CHANGELOG.md#v0310-beta
		// Here is the upstream PR https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/4277

		// Remove deprecated field metrics_schema from influxdb receiver
		if strings.HasPrefix(k.(string), "influxdb") {
			influxdbConfig, ok := v.(map[interface{}]interface{})
			if !ok {
				// no influxdbConfig? no need to fail because of that
				return otelcol, nil
			}
			for fieldKey := range influxdbConfig {
				if strings.HasPrefix(fieldKey.(string), "metrics_schema") {
					delete(influxdbConfig, fieldKey)
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.31.0 dropped the 'metrics_schema' field from %q receiver", k))
					continue
				}
			}
		}
	}

	cfg["receivers"] = receivers
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.31.0, failed to marshall back configuration: %w", err)
	}

	otelcol.Spec.Config = string(res)
	return otelcol, nil
}
