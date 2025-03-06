// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"strings"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// PodMonitor returns the pod monitor for the given instance.
func PodMonitor(params manifests.Params) (*monitoringv1.PodMonitor, error) {
	if !shouldCreatePodMonitor(params) {
		return nil, nil
	}

	name := naming.PodMonitor(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, nil)
	selectorLabels := manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector)
	pm := monitoringv1.PodMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.OtelCol.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: monitoringv1.PodMonitorSpec{
			JobLabel:        "app.kubernetes.io/instance",
			PodTargetLabels: []string{"app.kubernetes.io/name", "app.kubernetes.io/instance", "app.kubernetes.io/managed-by"},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.OtelCol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
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

func metricsEndpointsFromConfig(logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector) []monitoringv1.PodMetricsEndpoint {
	exporterPorts, err := otelcol.Spec.Config.GetExporterPorts(logger)
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

func shouldCreatePodMonitor(params manifests.Params) bool {
	l := params.Log.WithValues(
		"params.OtelCol.name", params.OtelCol.Name,
		"params.OtelCol.namespace", params.OtelCol.Namespace,
	)

	if !params.OtelCol.Spec.Observability.Metrics.EnableMetrics {
		l.V(2).Info("Metrics disabled for this OTEL Collector. PodMonitor will not ve created")
		return false
	} else if params.Config.PrometheusCRAvailability() == prometheus.NotAvailable {
		l.V(2).Info("Cannot enable PodMonitor when prometheus CRDs are unavailable")
		return false
	} else if params.OtelCol.Spec.Mode != v1beta1.ModeSidecar {
		l.V(2).Info("Not using sidecar mode. PodMonitor will not be created")
		return false
	}
	return true
}
