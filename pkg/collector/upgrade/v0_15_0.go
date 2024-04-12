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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func upgrade0_15_0(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	delete(otelcol.Spec.Args, "--new-metrics")
	delete(otelcol.Spec.Args, "--legacy-metrics")
	existing := &corev1.ConfigMap{}
	updated := existing.DeepCopy()
	u.Recorder.Event(updated, "Normal", "Upgrade", "upgrade to v0.15.0 dropped the deprecated metrics arguments")

	return otelcol, nil
}
