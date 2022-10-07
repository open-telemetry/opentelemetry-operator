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
	"errors"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

func upgrade0_61_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if len(otelcol.Spec.Config) == 0 {
		return otelcol, nil
	}

	otelCfg, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		return otelcol, fmt.Errorf("couldn't upgrade to v0.61.0, failed to parse configuration: %w", err)
	}

	// Search for removed Jaeger remote sampling settings. (https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/14163)
	receiversConfig, ok := otelCfg["receivers"].(map[any]any)
	if !ok {
		// In case there is no extensions config.
		return otelcol, nil
	}

	for key, rc := range receiversConfig {
		k, ok := key.(string)
		if !ok {
			continue
		}
		cfg, ok := rc.(map[any]any)
		// check if jaeger is configured
		if !ok || !strings.HasPrefix(k, "jaeger") {
			continue
		}

		// check if remote sampling settings exit
		if _, ok := cfg["remote_sampling"]; !ok {
			continue
		}

		const issueID = "https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/14707"
		errStr := fmt.Sprintf(
			"jaegerremotesampling is no longer available as receiver configuration. "+
				"Please use the extension instead with a different remote sampling port. See: %s",
			issueID,
		)
		u.Recorder.Event(otelcol, "Error", "Upgrade", errStr)
		return nil, errors.New(errStr)
	}
	return otelcol, nil
}
