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

func upgrade0_9_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if otelcol.Spec.Config == "" {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to parse configuration: %w", err)
	}

	exporters, ok := cfg["exporters"].(map[any]any)
	if !ok {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to extract list of exporters from the configuration: %q", cfg["exporters"])
	}

	for k, v := range exporters {
		if strings.HasPrefix(k.(string), "opencensus") {
			switch exporter := v.(type) {
			case map[any]any:
				// delete is a noop if there's no such entry
				delete(exporter, "reconnection_delay")
				existing := &corev1.ConfigMap{}
				updated := existing.DeepCopy()
				u.Recorder.Eventf(updated, nil, "Normal", "Upgrade", "Upgrade", "upgrade to v0.9.0 removed the property reconnection_delay for exporter %q", k)
				exporters[k] = exporter
			case string:
				if exporter == "" {
					// this exporter is using the default configuration
					continue
				}
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, the exporter %q is invalid (neither a string nor map)", k)
			}
		}
	}

	cfg["exporters"] = exporters
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to marshall back configuration: %w", err)
	}

	otelcol.Spec.Config = string(res)
	return otelcol, nil
}
