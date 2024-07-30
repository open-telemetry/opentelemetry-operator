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
	otelcol.Spec.Args = RemoveFeatureGate(otelcol.Spec.Args, envVarExpansionFeatureFlag)

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
