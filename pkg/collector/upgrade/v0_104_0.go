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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_104_0_TA(_ VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	TAUnifyEnvVarExpansion(otelcol)
	return otelcol, nil
}

func upgrade0_104_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	v1beta1.ComponentUseLocalHostAsDefaultHost(otelcol)

	const issueID = "https://github.com/open-telemetry/opentelemetry-collector/issues/8510"
	warnStr := fmt.Sprintf(
		"otlp receivers is no longer listen on 0.0.0.0 as default configuration. "+
			"The new default is localhost. Please revisit your \"%s\" configuration. See: %s",
		otelcol.Name, issueID,
	)
	u.Recorder.Event(otelcol, "Warning", "Upgrade", warnStr)
	return otelcol, nil
}

// TAUnifyEnvVarExpansion disables confmap.unifyEnvVarExpansion featuregate on
// collector instances if a prometheus receiver is configured.
// NOTE: We need this for now until 0.105.0 is out with this fix:
// https://github.com/open-telemetry/opentelemetry-collector/commit/637b1f42fcb7cbb7ef8a50dcf41d0a089623a8b7
func TAUnifyEnvVarExpansion(otelcol *v1beta1.OpenTelemetryCollector) {
	var enable bool
	for receiver := range otelcol.Spec.Config.Receivers.Object {
		if strings.Contains(receiver, "prometheus") {
			enable = true
			break
		}
	}
	if !enable {
		return
	}

	const (
		baseFlag = "feature-gates"
		fgFlag   = "confmap.unifyEnvVarExpansion"
	)
	if otelcol.Spec.Args == nil {
		otelcol.Spec.Args = make(map[string]string)
	}
	args, ok := otelcol.Spec.Args[baseFlag]
	if !ok || len(args) == 0 {
		otelcol.Spec.Args[baseFlag] = "-" + fgFlag
	} else if !strings.Contains(otelcol.Spec.Args[baseFlag], fgFlag) {
		otelcol.Spec.Args[baseFlag] += ",-" + fgFlag
	}
}
