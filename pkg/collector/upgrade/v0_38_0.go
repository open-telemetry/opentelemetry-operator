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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/collector/v1alpha1"
)

func upgrade0_38_0(cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	// return if args exist
	if len(otelcol.Spec.Args) == 0 {
		return otelcol, nil
	}

	// Remove otelcol args --log-level, --log-profile, --log-format
	// are deprecated in reference to https://github.com/open-telemetry/opentelemetry-collector/pull/4213
	var foundLoggingArgs []string
	for argKey := range otelcol.Spec.Args {
		if argKey == "--log-level" || argKey == "--log-profile" || argKey == "--log-format" {
			foundLoggingArgs = append(foundLoggingArgs, argKey)
			delete(otelcol.Spec.Args, argKey)
		}
	}

	if len(foundLoggingArgs) > 0 {
		otelcol.Status.Messages = append(otelcol.Status.Messages, fmt.Sprintf("upgrade to v0.38.0 dropped the deprecated logging arguments i.e. %v from otelcol custom resource.", foundLoggingArgs))
	}

	return otelcol, nil
}
