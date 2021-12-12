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

package collector

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// ServiceAccountName returns the name of the existing or self-provisioned service account to use for the given instance.
func ServiceAccountName(instance v1alpha1.OpenTelemetryCollector) string {
	if len(instance.Spec.ServiceAccount) == 0 {
		return naming.ServiceAccount(instance)
	}

	return instance.Spec.ServiceAccount
}

//ServiceAccount returns the service account for the given instance.
func ServiceAccount(otelcol v1alpha1.OpenTelemetryCollector) corev1.ServiceAccount {
	labels := Labels(otelcol)
	labels["app.kubernetes.io/name"] = naming.ServiceAccount(otelcol)

	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.ServiceAccount(otelcol),
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
	}
}
