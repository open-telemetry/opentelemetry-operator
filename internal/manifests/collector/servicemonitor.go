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
	"fmt"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil/naming"
)

// ServiceMonitor returns the service account for the given instance.
func ServiceMonitor(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (*monitoringv1.ServiceMonitor, error) {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: otelcol.Namespace,
			Name:      naming.ServiceMonitor(otelcol),
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.ServiceMonitor(otelcol),
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port: "monitoring",
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{otelcol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
		},
	}, nil
}
