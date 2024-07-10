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

func upgrade0_104_0(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) {
	for key, rc := range otelcol.Spec.Config.Receivers.Object {
		// check if otel is configured
		if !strings.HasPrefix(key, "otlp") {
			continue
		}

		cfg, ok := rc.(*v1beta1.AnyConfig)
		if !ok {
			continue
		}

		protocols, ok := cfg.Object["protocols"].(*v1beta1.AnyConfig)
		if !ok {
			continue
		}

		g, ok := protocols.Object["grpc"].(*v1beta1.AnyConfig)
		if ok {
			var got string
			endpoint, okk := g.Object["endpoint"]
			if okk {
				got, okk = endpoint.(string)
				if !okk {
					return nil, fmt.Errorf("specified otlp endpoint is not a string value")
				}
			}
			if got == "" {
				if g.Object == nil {
					g.Object = make(map[string]interface{})
				}
				g.Object["endpoint"] = "0.0.0.0:4317"
			}
		}

		h, ok := protocols.Object["http"].(*v1beta1.AnyConfig)
		if ok {
			var got string
			endpoint, okk := h.Object["endpoint"]
			if okk {
				got, okk = endpoint.(string)
				if !okk {
					return nil, fmt.Errorf("specified otlp endpoint is not a string value")
				}
			}
			if got == "" {
				if h.Object == nil {
					h.Object = make(map[string]interface{})
				}
				h.Object["endpoint"] = "0.0.0.0:4318"
			}
		}

		const issueID = "https://github.com/open-telemetry/opentelemetry-collector/issues/8510"
		warnStr := fmt.Sprintf(
			"otlp receivers is no longer listen on 0.0.0.0 as default configuration. "+
				"The new default is localhost. Please revisit your configuration. See: %s",
			issueID,
		)
		u.Recorder.Event(otelcol, "Warning", "Upgrade", warnStr)
	}
	return otelcol, nil
}
