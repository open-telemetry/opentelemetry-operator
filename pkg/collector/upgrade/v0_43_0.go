// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_43_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// return if args exist
	if len(otelcol.Spec.Args) == 0 {
		return otelcol, nil
	}

	//Removing deprecated Spec.Args (--metrics-addr and --metrics-level) based on
	// https://github.com/open-telemetry/opentelemetry-collector/pull/4695
	//Both args can be used now on the Spec.Config
	foundMetricsArgs := make(map[string]string)
	for argKey, argValue := range otelcol.Spec.Args {
		if argKey == "--metrics-addr" || argKey == "--metrics-level" {
			foundMetricsArgs[argKey] = argValue
			delete(otelcol.Spec.Args, argKey)
		}
	}

	// If we find metrics being used on Spec.Args we'll move to the syntax on Spec.Config
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

		// if there are already those Args under Spec.Config
		// then we won't override them.
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
		existing := &corev1.ConfigMap{}
		updated := existing.DeepCopy()
		u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.43.0 dropped the deprecated metrics arguments "+"i.e. %v from otelcol custom resource otelcol.spec.args and adding them to otelcol.spec.config.service.telemetry.metrics, if no metrics arguments are configured already.", keys))
	}
	return otelcol, nil
}
