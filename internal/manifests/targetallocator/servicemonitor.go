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

package targetallocator

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func ServiceMonitor(params manifests.Params) *monitoringv1.ServiceMonitor {

	sm := monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.OtelCol.Namespace,
			Name:      naming.TargetAllocator(params.OtelCol.Name),
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.TargetAllocator(params.OtelCol.Name),
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/component":  "opentelemetry-targetallocator",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "targetallocation",
				},
			},

			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.OtelCol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/component":  "opentelemetry-targetallocator",
				},
			},
		},
	}

	return &sm
}
