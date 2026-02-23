// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_41_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.41.0, failed to parse configuration: %w", err)
	}

	// Re-structure the cors section in otlp receiver
	// in reference to https://github.com/open-telemetry/opentelemetry-collector/pull/4492
	receivers, _ := cfg["receivers"].(map[any]any)

	for k1, v1 := range receivers {
		if strings.HasPrefix(k1.(string), "otlp") {
			otlpReceiver, _ := v1.(map[any]any)
			var createdCors bool
			for k2, v2 := range otlpReceiver {
				if k2.(string) == "cors_allowed_origins" || k2.(string) == "cors_allowed_headers" {
					if !createdCors {
						otlpReceiver["cors"] = make(map[any]any)
						createdCors = true
					}
					newsCorsKey := strings.Replace(k2.(string), "cors_", "", 1)
					otlpCors, _ := otlpReceiver["cors"].(map[any]any)
					otlpCors[newsCorsKey] = v2
					delete(otlpReceiver, k2)

					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.41.0 has re-structured the %s inside otlp "+"receiver config according to the upstream otlp receiver changes in 0.41.0 release.", k2))
				}
			}
		}
	}

	return updateConfig(otelcol, cfg)
}
