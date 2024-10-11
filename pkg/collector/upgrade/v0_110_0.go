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
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func upgrade0_110_0(_ VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error) { //nolint:unparam
	envVarExpansionFeatureFlag := "-component.UseLocalHostAsDefaultHost"
	otelcol.Spec.OpenTelemetryCommonFields.Args = RemoveFeatureGate(otelcol.Spec.OpenTelemetryCommonFields.Args, envVarExpansionFeatureFlag)
	return otelcol, nil
}
