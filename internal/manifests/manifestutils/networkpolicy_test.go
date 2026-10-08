// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNetworkPolicyPorts(t *testing.T) {
	tcp := corev1.ProtocolTCP
	udp := corev1.ProtocolUDP

	assert.Equal(t, []networkingv1.NetworkPolicyPort{
		{Protocol: &tcp, Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 4317}},
		{Protocol: &udp, Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 4317}},
	}, NetworkPolicyPorts([]corev1.ContainerPort{
		{ContainerPort: 4317},
		{ContainerPort: 4317, Protocol: corev1.ProtocolUDP},
	}))
}
