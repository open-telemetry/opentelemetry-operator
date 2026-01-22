// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestNetworkPolicy(t *testing.T) {
	trueValue := true
	t.Run("should return network policy with metrics port even when no receivers configured", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{},
					NetworkPolicy: v1beta1.NetworkPolicy{
						Enabled: &trueValue,
					},
				},
			},
		}

		actual, err := NetworkPolicy(params)
		assert.NoError(t, err)
		assert.NotNil(t, actual)

		// Should have metrics port (8888) by default
		assert.Len(t, actual.Spec.Ingress, 1)
		assert.Len(t, actual.Spec.Ingress[0].Ports, 1)
		assert.Equal(t, int32(8888), actual.Spec.Ingress[0].Ports[0].Port.IntVal)
	})

	t.Run("create network policy from networkpolicies.yaml", func(t *testing.T) {
		configYAML, err := os.ReadFile("testdata/networkpolicies.yaml")
		assert.NoError(t, err)

		cfg := v1beta1.Config{}
		err = yaml.Unmarshal(configYAML, &cfg)
		assert.NoError(t, err)

		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: cfg,
					NetworkPolicy: v1beta1.NetworkPolicy{
						Enabled: &trueValue,
					},
				},
			},
		}

		actual, err := NetworkPolicy(params)
		assert.NoError(t, err)
		assert.NotNil(t, actual)

		tcp := corev1.ProtocolTCP
		expected := &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.CollectorNetworkPolicy(params.OtelCol.Name),
				Namespace:   params.OtelCol.Namespace,
				Annotations: map[string]string{},
				Labels:      manifestutils.Labels(params.OtelCol.ObjectMeta, naming.CollectorNetworkPolicy(params.OtelCol.Name), params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter),
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, ComponentOpenTelemetryCollector),
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &tcp,
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 3333},
							},
							{
								Protocol: &tcp,
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 12345},
							},
							{
								Protocol: &tcp,
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 4318},
							},
							{
								Protocol: &tcp,
								Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 1111},
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			},
		}

		assert.Equal(t, expected, actual)
	})
}
