// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func NetworkPolicy(params manifests.Params) (*networkingv1.NetworkPolicy, error) {
	if params.OtelCol.Spec.NetworkPolicy.Enabled == nil || !*params.OtelCol.Spec.NetworkPolicy.Enabled {
		return nil, nil
	}

	name := naming.CollectorNetworkPolicy(params.OtelCol.Name)
	labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)
	annotations, err := manifestutils.Annotations(params.OtelCol, params.Config.AnnotationsFilter)
	if err != nil {
		return nil, err
	}

	ports := getContainerPorts(params.Log, params.OtelCol)

	var ingressPorts []intstr.IntOrString
	for _, port := range ports {
		ingressPorts = append(ingressPorts, intstr.FromInt32(port.ContainerPort))
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}

	tcp := corev1.ProtocolTCP
	for i := range ingressPorts {
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &ingressPorts[i],
		})
	}

	return np, nil
}
