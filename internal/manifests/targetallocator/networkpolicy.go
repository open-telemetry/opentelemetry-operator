// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
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
	annotations := ResourceAnnotations(params.TargetAllocator, params.Config.AnnotationsFilter)

	tcp := corev1.ProtocolTCP

	// The target allocator's egress is normally restricted to the API server, since
	// that's the only destination it needs to reach. Self-telemetry lets users point
	// the exporter at an arbitrary (often external) endpoint that can't be expressed
	// as a NetworkPolicy IPBlock/selector, so restricting egress in that case would
	// silently break the very telemetry export the user configured. Leave egress
	// unrestricted instead, matching the collector's own NetworkPolicy (see
	// internal/manifests/collector/networkpolicy.go), which never restricts egress
	// for the same reason: exporters can target arbitrary destinations.
	policyTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}
	var egress []networkingv1.NetworkPolicyEgressRule
	if !hasSelfTelemetryExporter(params.TargetAllocator) {
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
		policyTypes = append(policyTypes, networkingv1.PolicyTypeEgress)
		egress = []networkingv1.NetworkPolicyEgressRule{
			{
				Ports: []networkingv1.NetworkPolicyPort{
					{
						Protocol: &tcp,
						Port:     &apiServerPort,
					},
				},
				To: apiSeverIPs,
			},
		}
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
			Egress:      egress,
			PolicyTypes: policyTypes,
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

func hasSelfTelemetryExporter(ta v1alpha1.TargetAllocator) bool {
	for _, reader := range ta.Spec.Telemetry.Metrics.Readers {
		if reader.Periodic != nil {
			return true
		}
	}
	return false
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

	if manifestutils.IsTAMTLSEnabled(params.TargetAllocator.Spec.Mtls) {
		ports = append(ports, corev1.ContainerPort{
			Name:          "https",
			ContainerPort: defaultHTTPSPort,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	return ports
}
