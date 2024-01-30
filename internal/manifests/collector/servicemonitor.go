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
	"strings"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func ServiceMonitor(params manifests.Params) (*monitoringv1.ServiceMonitor, error) {
	if !params.OtelCol.Spec.Observability.Metrics.EnableMetrics {
		params.Log.V(2).Info("Metrics disabled for this OTEL Collector",
			"params.OtelCol.name", params.OtelCol.Name,
			"params.OtelCol.namespace", params.OtelCol.Namespace,
		)
		return nil, nil
	}
	var sm monitoringv1.ServiceMonitor

	if params.OtelCol.Spec.Mode == v1alpha1.ModeSidecar {
		return nil, nil
	}
	name := naming.ServiceMonitor(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	selectorMatchLabels := manifestutils.SelectorMatchLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector)
	sm = monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.OtelCol.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: append([]monitoringv1.Endpoint{
				{
					Port: "monitoring",
				},
			}, endpointsFromConfig(params.Log, params.OtelCol)...),
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.OtelCol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: selectorMatchLabels,
			},
		},
	}

	return &sm, nil
}

func endpointsFromConfig(logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []monitoringv1.Endpoint {
	c, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		logger.V(2).Error(err, "Error while parsing the configuration")
		return []monitoringv1.Endpoint{}
	}

	exporterPorts, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeExporter, c)
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
