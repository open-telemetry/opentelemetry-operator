// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// ServiceMonitor returns the service monitor for the given instance.
func ServiceMonitor(params Params) *monitoringv1.ServiceMonitor {
	name := naming.TargetAllocator(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "targetallocation",
				},
			},

			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{params.TargetAllocator.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: manifestutils.TASelectorLabels(params.TargetAllocator, ComponentOpenTelemetryTargetAllocator),
			},
		},
	}
}
