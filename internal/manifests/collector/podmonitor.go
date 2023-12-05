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
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func PodMonitor(params manifests.Params) (*monitoringv1.PodMonitor, error) {
	if !params.OtelCol.Spec.Observability.Metrics.EnableMetrics {
		params.Log.V(2).Info("Metrics disabled for this OTEL Collector",
			"params.OtelCol.name", params.OtelCol.Name,
			"params.OtelCol.namespace", params.OtelCol.Namespace,
		)
		return nil, nil
	}
	var pm monitoringv1.PodMonitor

	if params.OtelCol.Spec.Mode != v1alpha1.ModeSidecar {
		return nil, nil
	}

	pm = monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.OtelCol.Namespace,
			Name:      naming.PodMonitor(params.OtelCol.Name),
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.PodMonitor(params.OtelCol.Name),
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: monitoringv1.PodMonitorSpec{
			JobLabel:        "app.kubernetes.io/instance",
			PodTargetLabels: []string{"app.kubernetes.io/name", "app.kubernetes.io/instance", "app.kubernetes.io/managed-by"},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.OtelCol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
				},
			},
			PodMetricsEndpoints: append(
				[]monitoringv1.PodMetricsEndpoint{
					{
						Port: "monitoring",
					},
				}, metricsEndpointsFromConfig(params.Log, params.OtelCol)...),
		},
	}

	return &pm, nil
}

func metricsEndpointsFromConfig(logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) []monitoringv1.PodMetricsEndpoint {
	config, err := adapters.ConfigFromString(otelcol.Spec.Config)
	if err != nil {
		logger.V(2).Error(err, "Error while parsing the configuration")
		return []monitoringv1.PodMetricsEndpoint{}
	}
	exporterPorts, err := adapters.ConfigToComponentPorts(logger, adapters.ComponentTypeExporter, config)
	if err != nil {
		logger.Error(err, "couldn't build endpoints to podMonitors from configuration")
		return []monitoringv1.PodMetricsEndpoint{}
	}
	metricsEndpoints := []monitoringv1.PodMetricsEndpoint{}
	for _, port := range exporterPorts {
		if strings.Contains(port.Name, "prometheus") {
			e := monitoringv1.PodMetricsEndpoint{
				Port: port.Name,
			}
			metricsEndpoints = append(metricsEndpoints, e)
		}
	}
	return metricsEndpoints
}
