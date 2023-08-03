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
	"strings"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func ServiceMonitor(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (*monitoringv1.ServiceMonitor, error) {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: otelcol.Namespace,
			Name:      naming.ServiceMonitor(otelcol.Name),
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.ServiceMonitor(otelcol.Name),
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

func ServiceMonitorFromConfig(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []*monitoringv1.ServiceMonitor {
	sms := []*monitoringv1.ServiceMonitor{}

	c, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		logger.V(2).Error(err, "Error while parsing the configuration")
		return []*monitoringv1.ServiceMonitor{}
	}

	exporterPorts, err := adapters.ConfigToExporterPorts(logger, c)
	if err != nil {
		logger.Error(err, "couldn't build service monitors from configuration")
		return []*monitoringv1.ServiceMonitor{}
	}

	for _, port := range exporterPorts {
		// Create a ServiceMonitor onhly for those ports related to prometheus exporters
		if strings.Contains(port.Name, "prometheus") {
			sms = append(sms,
				&monitoringv1.ServiceMonitor{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: otelcol.Namespace,
						Name:      naming.ServiceMonitor(port.Name),
						Labels: map[string]string{
							"app.kubernetes.io/name":       naming.ServiceMonitor(port.Name),
							"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name),
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
						},
					},
					Spec: monitoringv1.ServiceMonitorSpec{
						Endpoints: []monitoringv1.Endpoint{{
							Port: port.Name,
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
				},
			)
		}
	}

	return sms
}
