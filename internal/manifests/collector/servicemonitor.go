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
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func ServiceMonitor(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector, otelColConfig manifests.OtelConfig) (*monitoringv1.ServiceMonitor, error) {
	if !otelcol.Spec.Observability.Metrics.EnableMetrics {
		logger.V(2).Info("Metrics disabled for this OTEL Collector",
			"otelcol.name", otelcol.Name,
			"otelcol.namespace", otelcol.Namespace,
		)
		return nil, nil
	}

	sm := monitoringv1.ServiceMonitor{
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
			Endpoints: []monitoringv1.Endpoint{},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{otelcol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name),
				},
			},
		},
	}

	endpoints := []monitoringv1.Endpoint{
		{
			Port: "monitoring",
		},
	}

	sm.Spec.Endpoints = append(endpoints, endpointsFromConfig(logger, otelcol)...)
	return &sm, nil
}

func endpointsFromConfig(logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []monitoringv1.Endpoint {
	c, err := adapters.ConfigFromString(otelcol.Spec.ConfigSpec.String())
	if err != nil {
		logger.V(2).Error(err, "Error while parsing the configuration")
		return []monitoringv1.Endpoint{}
	}

	exporterPorts, err := adapters.ConfigToExporterPorts(logger, c)
	if err != nil {
		logger.Error(err, "couldn't build service monitors from configuration")
		return []monitoringv1.Endpoint{}
	}

	endpoints := []monitoringv1.Endpoint{}

	for _, port := range exporterPorts {
		if strings.Contains(port.Name, "prometheus") {
			e := monitoringv1.Endpoint{
				Port: port.Name,
			}
			endpoints = append(endpoints, e)
		}
	}
	return endpoints
}
