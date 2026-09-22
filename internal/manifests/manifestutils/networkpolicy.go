// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkPolicyPorts converts container ports to NetworkPolicy ports while
// preserving their transport protocols. Kubernetes defaults an omitted
// container port protocol to TCP, so normalize an empty protocol accordingly.
func NetworkPolicyPorts(ports []corev1.ContainerPort) []networkingv1.NetworkPolicyPort {
	networkPolicyPorts := make([]networkingv1.NetworkPolicyPort, 0, len(ports))
	for _, port := range ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		networkPolicyPorts = append(networkPolicyPorts, networkingv1.NetworkPolicyPort{
			Protocol: new(protocol),
			Port:     new(intstr.FromInt32(port.ContainerPort)),
		})
	}
	return networkPolicyPorts
}
