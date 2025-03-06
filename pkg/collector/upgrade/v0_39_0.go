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

func upgrade0_39_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.39.0, failed to parse configuration: %w", err)
	}

	// Remove processors.memory_limiter.ballast_size_mib
	// as it is deprecated in reference to https://github.com/open-telemetry/opentelemetry-collector/pull/4365
	processors, _ := cfg["processors"].(map[interface{}]interface{})

	for k1, v1 := range processors {
		// Drop the deprecated field ballast_size_mib from memory_limiter
		if strings.HasPrefix(k1.(string), "memory_limiter") {
			memoryLimiter, _ := v1.(map[interface{}]interface{})
			for k2 := range memoryLimiter {
				if k2 == "ballast_size_mib" {
					delete(memoryLimiter, k2)
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.39.0 has dropped the ballast_size_mib field name from %s processor", k1))
				}
			}
		}
	}

	otelcol, err = updateConfig(otelcol, cfg)
	if err != nil {
		return otelcol, err
	}

	// Rename httpd receiver to apache receiver
	// in reference to https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/6207
	receivers, _ := cfg["receivers"].(map[interface{}]interface{})

	for k1, v1 := range receivers {
		if strings.HasPrefix(k1.(string), "httpd") {
			// Rename httpd with apache
			apacheKey := strings.Replace(k1.(string), "httpd", "apache", 1)
			receivers[apacheKey] = v1
			delete(receivers, k1)

			// rename receiver name in service pipelines config
			serviceConfig, ok := cfg["service"].(map[interface{}]interface{})
			if !ok {
				// no serviceConfig?
				return otelcol, nil
			}

			pipelinesConfig, ok := serviceConfig["pipelines"].(map[interface{}]interface{})
			if !ok {
				// no pipelinesConfig?
				return otelcol, nil
			}

			for k2, v2 := range pipelinesConfig {
				if k2.(string) == "metrics" {
					metricsConfig, ok := v2.(map[interface{}]interface{})
					if !ok {
						// no metricsConfig in service pipelines?
						return otelcol, nil
					}
					for k3, v3 := range metricsConfig {
						if k3.(string) == "receivers" {
							receiversList, ok := v3.([]interface{})
							if !ok {
								// no receivers list in service pipeline?
								return otelcol, nil
							}
							for i, k4 := range receiversList {
								if strings.HasPrefix(k4.(string), "httpd") {
									receiversList[i] = strings.Replace(k4.(string), "httpd", "apache", 1)
									existing := &corev1.ConfigMap{}
									updated := existing.DeepCopy()
									u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.39.0 has dropped the ballast_size_mib field name from %s processor", receiversList[i]))
								}
							}
						}
					}
				}
			}
		}
	}

	return updateConfig(otelcol, cfg)
}

func updateConfig(otelcol *v1alpha1.OpenTelemetryCollector, cfg map[interface{}]interface{}) (*v1alpha1.OpenTelemetryCollector, error) {
	res, err := yaml.Marshal(cfg)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.39.0, failed to marshall back configuration: %w", err)
	}

	// Note: This is a hack to drop null occurrences from our config parsing above in upgrade routine
	otelcol.Spec.Config = strings.ReplaceAll(string(res), " null", "")

	return otelcol, nil
}
