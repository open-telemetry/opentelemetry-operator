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
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	
	corev1 "k8s.io/api/core/v1"
)

func upgrade0_57_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.57.2, failed to parse configuration: %w", err)
	}

	//Remove externsions.healthcheckextension - Remove deprecated port field from config. (#12668)
	extensionsConfig, ok := cfg["extensions"].(map[interface{}]interface{})
	if !ok {
		// In case there is no health check config.
		return otelcol, nil
	}

	for keyExt, valExt := range extensionsConfig {
		if strings.HasPrefix(keyExt.(string), "health_check") {
			switch extensions := valExt.(type) {
			case map[interface{}]interface{}:
				if port, ok := extensions["port"]; ok {
					endpointV := extensions["endpoint"]
					extensions["endpoint"] = fmt.Sprintf("%s:%d", endpointV, port)
					delete(extensions, "port")
					existing := &corev1.ConfigMap{}
					updated := existing.DeepCopy()
					u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.57.2 has deprecated port for healthcheck extension %q", keyExt))
				}
			default:
				return otelcol, fmt.Errorf("couldn't upgrade to v0.57.2, the extension %q is invalid (expected string or map but was %t)", keyExt, valExt)
			}
		}
	}
	return otelcol, nil
}
