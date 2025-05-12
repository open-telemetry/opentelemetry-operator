// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
)

func upgrade0_38_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// return if args exist
	if len(otelcol.Spec.Args) == 0 {
		return otelcol, nil
	}

	// Remove otelcol args --log-level, --log-profile, --log-format
	// are deprecated in reference to https://github.com/open-telemetry/opentelemetry-collector/pull/4213
	foundLoggingArgs := make(map[string]string)
	for argKey, argValue := range otelcol.Spec.Args {
		if argKey == "--log-level" || argKey == "--log-profile" || argKey == "--log-format" {
			foundLoggingArgs[argKey] = argValue
			delete(otelcol.Spec.Args, argKey)
		}
	}

	if len(foundLoggingArgs) > 0 {
		cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
		if err != nil {
			return otelcol, fmt.Errorf("couldn't upgrade to v0.38.0, failed to parse configuration: %w", err)
		}

		serviceConfig, ok := cfg["service"].(map[interface{}]interface{})
		if !ok {
			// no serviceConfig? create one as we need to configure logging parameters
			cfg["service"] = make(map[interface{}]interface{})
			serviceConfig, _ = cfg["service"].(map[interface{}]interface{})
		}

		telemetryConfig, ok := serviceConfig["telemetry"].(map[interface{}]interface{})
		if !ok {
			// no telemetryConfig? create one as we need to configure logging parameters
			serviceConfig["telemetry"] = make(map[interface{}]interface{})
			telemetryConfig, _ = serviceConfig["telemetry"].(map[interface{}]interface{})
		}

		logsConfig, ok := telemetryConfig["logs"].(map[interface{}]interface{})
		if !ok {
			// no logsConfig? create one as we need to configure logging parameters
			telemetryConfig["logs"] = make(map[interface{}]interface{})
			logsConfig, _ = telemetryConfig["logs"].(map[interface{}]interface{})
		}

		// if there is already loggingConfig
		// do not override it with values from deprecated args
		if len(logsConfig) == 0 {
			if val, ok := foundLoggingArgs["--log-level"]; ok {
				logsConfig["level"] = val
			}
			if _, ok := foundLoggingArgs["--log-profile"]; ok {
				logsConfig["development"] = true
			}
			if val, ok := foundLoggingArgs["--log-format"]; ok {
				logsConfig["encoding"] = val
			}
		}
		cfg["service"] = serviceConfig
		res, err := yaml.Marshal(cfg)
		if err != nil {
			return otelcol, fmt.Errorf("couldn't upgrade to v0.38.0, failed to marshall back configuration: %w", err)
		}
		otelcol.Spec.Config = string(res)
		keys := reflect.ValueOf(foundLoggingArgs).MapKeys()
		// sort keys to get always the same message
		sort.Slice(keys, func(i, j int) bool {
			return strings.Compare(keys[i].String(), keys[j].String()) <= 0
		})
		existing := &corev1.ConfigMap{}
		updated := existing.DeepCopy()
		u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.38.0 dropped the deprecated logging arguments "+"i.e. %v from otelcol custom resource otelcol.spec.args and adding them to otelcol.spec.config.service.telemetry.logs, if no logging parameters are configured already.", keys))
	}
	return otelcol, nil
}
