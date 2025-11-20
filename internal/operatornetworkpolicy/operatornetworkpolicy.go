// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

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
	operatorName         = "opentelemetry-operator-controller-manager"
	defaultAPIServerPort = 6443
	defaultRBACProxyPort = 8443
)

type networkPolicy struct {
	clientset kubernetes.Interface
	scheme    *runtime.Scheme

	operatorNamespace          string
	webhookPort                int32
	metricsPort                int32
	apiServerPodSelector       *metav1.LabelSelector
	apiServerNamespaceSelector *metav1.LabelSelector
}

var _ manager.Runnable = (*networkPolicy)(nil)
var _ manager.LeaderElectionRunnable = (*networkPolicy)(nil)

func NewOperatorNetworkPolicy(clientset kubernetes.Interface, scheme *runtime.Scheme, options ...Option) manager.Runnable {
	n := &networkPolicy{
		clientset: clientset,
		scheme:    scheme,
	}

	for _, opt := range options {
		opt(n)
	}
	return n
}

type Option func(policy *networkPolicy)

// WithOperatorNamespace sets the namespace of the operator and enables it in the network policy.
func WithOperatorNamespace(operatorNamespace string) Option {
	return func(s *networkPolicy) {
		s.operatorNamespace = operatorNamespace
	}
}

// WithWebhookPort sets the port of the webhook and enables it in the network policy.
func WithWebhookPort(webhookPort int32) Option {
	return func(s *networkPolicy) {
		s.webhookPort = webhookPort
	}
}

// WithMetricsPort sets the port of the metrics endpoint and enables it in the network policy.
func WithMetricsPort(metricsPort int32) Option {
	return func(s *networkPolicy) {
		s.metricsPort = metricsPort
	}
}

// WithAPISererPodLabelSelector sets the label selector for the pod of the API server.
func WithAPISererPodLabelSelector(selector *metav1.LabelSelector) Option {
	return func(s *networkPolicy) {
		s.apiServerPodSelector = selector
	}
}

// WithAPISererNamespaceLabelSelector sets the label selector for tbe namespace of the API server.
func WithAPISererNamespaceLabelSelector(selector *metav1.LabelSelector) Option {
	return func(s *networkPolicy) {
		s.apiServerNamespaceSelector = selector
	}
}

func (n *networkPolicy) Start(ctx context.Context) error {
	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(defaultAPIServerPort)

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
			Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
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
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	if n.apiServerPodSelector != nil {
		np.Spec.Egress[0].To = append(np.Spec.Egress[0].To, networkingv1.NetworkPolicyPeer{
			PodSelector: n.apiServerPodSelector,
		})
	}
	if n.apiServerNamespaceSelector != nil {
		if np.Spec.Egress[0].To == nil {
			np.Spec.Egress[0].To = append(np.Spec.Egress[0].To, networkingv1.NetworkPolicyPeer{
				NamespaceSelector: n.apiServerNamespaceSelector,
			})
		} else {
			np.Spec.Egress[0].To[0].NamespaceSelector = n.apiServerNamespaceSelector
		}
	}

	if n.webhookPort != 0 {
		webhookPort := intstr.FromInt32(n.webhookPort)
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &webhookPort,
		})
	}
	if n.metricsPort != 0 {
		metricsPort := intstr.FromInt32(n.metricsPort)
		// The RBAC proxy is used to secure the metrics endpoint.
		rbacProxyPort := intstr.FromInt32(defaultRBACProxyPort)
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &metricsPort,
		})
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &rbacProxyPort,
		})
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

func (d *networkPolicy) NeedLeaderElection() bool {
	return true
}
