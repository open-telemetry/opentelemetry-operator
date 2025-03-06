// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"slices"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_105_0(_ VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	var enable bool
	for receiver := range otelcol.Spec.Config.Receivers.Object {
		if strings.Contains(receiver, "prometheus") {
			enable = true
			break
		}
	}
	if !enable {
		return otelcol, nil
	}

	envVarExpansionFeatureFlag := "-confmap.unifyEnvVarExpansion"
	otelcol.Spec.OpenTelemetryCommonFields.Args = RemoveFeatureGate(otelcol.Spec.OpenTelemetryCommonFields.Args, envVarExpansionFeatureFlag)

	return otelcol, nil
}

const featureGateFlag = "feature-gates"

// RemoveFeatureGate removes a feature gate from args.
func RemoveFeatureGate(args map[string]string, feature string) map[string]string {
	featureGates, ok := args[featureGateFlag]
	if !ok {
		return args
	}
	if !strings.Contains(featureGates, feature) {
		return args
	}

	features := strings.Split(featureGates, ",")
	features = slices.DeleteFunc(features, func(s string) bool {
		return s == feature
	})
	if len(features) == 0 {
		delete(args, featureGateFlag)
	} else {
		featureGates = strings.Join(features, ",")
		args[featureGateFlag] = featureGates
	}
	return args
}
