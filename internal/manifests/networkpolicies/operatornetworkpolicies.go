// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package networkpolicies

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	operatorName = "opentelemetry-operator-controller-manager"
)

type networkPolicies struct {
	clientset         kubernetes.Interface
	operatorNamespace string
	scheme            *runtime.Scheme
}

var _ manager.Runnable = (*networkPolicies)(nil)
var _ manager.LeaderElectionRunnable = (*networkPolicies)(nil)

func NewOperatorNetworkPolicies(operatorNamespace string, clientset kubernetes.Interface, scheme *runtime.Scheme) manager.Runnable {
	return networkPolicies{
		clientset:         clientset,
		operatorNamespace: operatorNamespace,
		scheme:            scheme,
	}
}

func (n networkPolicies) Start(ctx context.Context) error {
	tcp := corev1.ProtocolTCP
	webhookPort := intstr.FromInt32(9443)
	metricsPort := intstr.FromInt32(8443)
	apiServerPort := intstr.FromInt32(6443)

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opentelemetry-operator",
			Namespace: n.operatorNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "opentelemetry-operator",
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &tcp,
							Port:     &webhookPort,
						},
						{
							Protocol: &tcp,
							Port:     &metricsPort,
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
				},
			},
			PolicyTypes: []networkingv1.PolicyType{"Ingress", "Egress"},
		},
	}

	operatorDep, err := n.clientset.AppsV1().Deployments(n.operatorNamespace).Get(ctx, operatorName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// set owner reference to the operator deployment
	err = controllerutil.SetControllerReference(operatorDep, np, n.scheme)
	if err != nil {
		return err
	}

	_, err = n.clientset.NetworkingV1().NetworkPolicies(n.operatorNamespace).Create(ctx, np, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	<-ctx.Done()
	return nil
}

func (d networkPolicies) NeedLeaderElection() bool {
	return true
}
