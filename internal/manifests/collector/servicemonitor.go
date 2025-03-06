// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
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

// ServiceMonitor returns the service monitor for the collector.
func ServiceMonitor(params manifests.Params) (*monitoringv1.ServiceMonitor, error) {
	name := naming.ServiceMonitor(params.OtelCol.Name)
	endpoints := endpointsFromConfig(params.Log, params.OtelCol)
	if len(endpoints) > 0 {
		return createServiceMonitor(name, params, BaseServiceType, endpoints)
	}
	return nil, nil
}

// ServiceMonitor returns the service monitor for the monitoring service of the collector.
func ServiceMonitorMonitoring(params manifests.Params) (*monitoringv1.ServiceMonitor, error) {
	name := naming.ServiceMonitor(fmt.Sprintf("%s-monitoring", params.OtelCol.Name))
	endpoints := []monitoringv1.Endpoint{
		{
			Port: "monitoring",
		},
	}
	return createServiceMonitor(name, params, MonitoringServiceType, endpoints)
}

// createServiceMonitor creates a Service Monitor using the provided name, the params from the instance, a label to identify the service
// to target (like the monitoring or the collector services) and the endpoints to scrape.
func createServiceMonitor(name string, params manifests.Params, serviceType ServiceType, endpoints []monitoringv1.Endpoint) (*monitoringv1.ServiceMonitor, error) {
	if !shouldCreateServiceMonitor(params) {
		return nil, nil
	}

	var sm monitoringv1.ServiceMonitor

	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})
	selectorLabels := manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector)
	// This label is the one which differentiates the services
	selectorLabels[serviceTypeLabel] = serviceType.String()

	sm = monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.OtelCol.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: endpoints,
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.OtelCol.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
		},
	}

	return &sm, nil
}

func shouldCreateServiceMonitor(params manifests.Params) bool {
	l := params.Log.WithValues(
		"params.OtelCol.name", params.OtelCol.Name,
		"params.OtelCol.namespace", params.OtelCol.Namespace,
	)

	if !params.OtelCol.Spec.Observability.Metrics.EnableMetrics {
		l.V(2).Info("Metrics disabled for this OTEL Collector. ServiceMonitor will not ve created")
		return false
	} else if params.Config.PrometheusCRAvailability() == prometheus.NotAvailable {
		l.V(2).Info("Cannot enable ServiceMonitor when prometheus CRDs are unavailable")
		return false
	} else if params.OtelCol.Spec.Mode == v1beta1.ModeSidecar {
		l.V(2).Info("Using sidecar mode. ServiceMonitor will not be created")
		return false
	}
	return true
}

func endpointsFromConfig(logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector) []monitoringv1.Endpoint {
	exporterPorts, err := otelcol.Spec.Config.GetExporterPorts(logger)
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
