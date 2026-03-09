// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	defaultHTTPPort  = 8080
	defaultHTTPSPort = 8443
)

func NetworkPolicy(params Params) (*networkingv1.NetworkPolicy, error) {
	if params.TargetAllocator.Spec.NetworkPolicy.Enabled == nil || !*params.TargetAllocator.Spec.NetworkPolicy.Enabled {
		return nil, nil
	}

	name := naming.TargetAllocatorNetworkPolicy(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Status.Image, ComponentOpenTelemetryTargetAllocator, params.Config.LabelsFilter)
	annotations := Annotations(params.TargetAllocator, nil, params.Config.AnnotationsFilter)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(params.Config.Internal.KubeAPIServerPort)
	var apiSeverIPs []networkingv1.NetworkPolicyPeer
	// Add IPBlock rules for API server IPs
	for _, ip := range params.Config.Internal.KubeAPIServerIPs {
		cidr := ip + "/32"
		apiSeverIPs = append(apiSeverIPs, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.TargetAllocator.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: manifestutils.SelectorLabels(params.TargetAllocator.ObjectMeta, ComponentOpenTelemetryTargetAllocator),
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &tcp,
							Port:     &apiServerPort,
						},
					},
					To: apiSeverIPs,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	ports := getContainerPorts(params.TargetAllocator, params)
	var ingressPorts []intstr.IntOrString
	for _, port := range ports {
		ingressPorts = append(ingressPorts, intstr.FromInt32(port.ContainerPort))
	}
	for i := range ingressPorts {
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &ingressPorts[i],
		})
	}

	return np, nil
}

func getContainerPorts(instance v1alpha1.TargetAllocator, params Params) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, 0)

	// Default http port
	ports = append(ports, corev1.ContainerPort{
		Name:          "http",
		ContainerPort: defaultHTTPPort,
		Protocol:      corev1.ProtocolTCP,
	})

	// Add custom ports from spec
	for _, p := range instance.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      p.Protocol,
		})
	}

	if params.Config.CertManagerAvailability == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		ports = append(ports, corev1.ContainerPort{
			Name:          "https",
			ContainerPort: defaultHTTPSPort,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	return ports
}
