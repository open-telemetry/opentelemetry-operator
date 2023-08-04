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

	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"

	corev1 "k8s.io/api/core/v1"
)

func upgrade0_9_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	cfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to parse configuration: %w", err)
	}

	exporters, ok := cfg["exporters"].(map[interface{}]interface{})
	if !ok {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.9.0, failed to extract list of exporters from the configuration: %q", cfg["exporters"])
	}

	for k, v := range exporters {
		if strings.HasPrefix("opencensus", k.(string)) {
			switch exporter := v.(type) {
			case map[interface{}]interface{}:
				// delete is a noop if there's no such entry
				delete(exporter, "reconnection_delay")
				existing := &corev1.ConfigMap{}
				updated := existing.DeepCopy()
				u.Recorder.Event(updated, "Normal", "Upgrade", fmt.Sprintf("upgrade to v0.9.0 removed the property reconnection_delay for exporter %q", k))
				exporters[k] = exporter
			case string:
				if len(exporter) == 0 {
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
