// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestNetworkPolicy(t *testing.T) {
	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	testConfig := config.Config{}
	testConfig.Internal.KubeAPIServerPort = 6443
	testConfig.Internal.KubeAPIServerIPs = []string{"10.0.0.1"}

	tests := []struct {
		name     string
		ta       v1alpha1.TargetAllocator
		cfg      config.Config
		expected *networkingv1.NetworkPolicy
	}{
		{
			name: "network policy disabled",
			ta: v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ta",
					Namespace: "default",
				},
				Spec: v1alpha1.TargetAllocatorSpec{
					NetworkPolicy: v1beta1.NetworkPolicy{
						Enabled: &[]bool{false}[0],
					},
				},
			},
			cfg:      testConfig,
			expected: nil,
		},
		{
			name: "network policy enabled",
			ta: v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ta",
					Namespace: "default",
				},
				Spec: v1alpha1.TargetAllocatorSpec{
					NetworkPolicy: v1beta1.NetworkPolicy{
						Enabled: &[]bool{true}[0],
					},
				},
			},
			cfg: testConfig,
			expected: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ta-targetallocator-networkpolicy",
					Namespace: "default",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-targetallocator",
							"app.kubernetes.io/instance":   "default.test-ta",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Ingress: []networkingv1.NetworkPolicyIngressRule{
						{
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
									Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
								},
							},
						},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &tcp,
									Port:     &apiServerPort,
								},
							},
							To: []networkingv1.NetworkPolicyPeer{
								{
									IPBlock: &networkingv1.IPBlock{
										CIDR: "10.0.0.1/32",
									},
								},
							},
						},
					},
					PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := Params{
				TargetAllocator: test.ta,
				Config:          test.cfg,
			}
			actual, err := NetworkPolicy(params)
			require.NoError(t, err)

			if test.expected == nil {
				assert.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				assert.Equal(t, test.expected.Name, actual.Name)
				assert.Equal(t, test.expected.Namespace, actual.Namespace)
				assert.Equal(t, test.expected.Spec.PodSelector, actual.Spec.PodSelector)
				assert.Equal(t, test.expected.Spec.PolicyTypes, actual.Spec.PolicyTypes)
				assert.Len(t, actual.Spec.Ingress, 1)
				assert.Len(t, actual.Spec.Ingress[0].Ports, 1)
				assert.Equal(t, test.expected.Spec.Ingress[0].Ports[0], actual.Spec.Ingress[0].Ports[0])
				assert.Len(t, actual.Spec.Egress, 1)
				assert.Equal(t, test.expected.Spec.Egress[0].Ports, actual.Spec.Egress[0].Ports)
				assert.Equal(t, test.expected.Spec.Egress[0].To, actual.Spec.Egress[0].To)
			}
		})
	}
}
