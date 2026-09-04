// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
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

type networkPolicy struct {
	clientset kubernetes.Interface
	scheme    *runtime.Scheme

	operatorNamespace          string
	operatorPodName            string
	apiServerPort              int32
	apiServerIPs               []string
	webhookPort                int32
	metricsPort                int32
	apiServerPodSelector       *metav1.LabelSelector
	apiServerNamespaceSelector *metav1.LabelSelector
}

var (
	_ manager.Runnable               = (*networkPolicy)(nil)
	_ manager.LeaderElectionRunnable = (*networkPolicy)(nil)
)

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

// WithOperatorPodName sets the name of the pod the operator runs in. It is
// used to resolve the operator's own Deployment, whose name depends on how
// the operator was installed (kustomize, Helm chart, OLM).
func WithOperatorPodName(podName string) Option {
	return func(s *networkPolicy) {
		s.operatorPodName = podName
	}
}

// WithAPIServerPort sets the port of the API server and enables it in the network policy.
func WithAPIServerPort(apiServerPort int32) Option {
	return func(s *networkPolicy) {
		s.apiServerPort = apiServerPort
	}
}

// WithAPIServerIPs sets the IPs of the API server for use in network policy IPBlock rules.
func WithAPIServerIPs(ips []string) Option {
	return func(s *networkPolicy) {
		s.apiServerIPs = ips
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
	operatorDep, err := n.operatorDeployment(ctx)
	if err != nil {
		return err
	}

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(n.apiServerPort)

	var apiSeverIPs []networkingv1.NetworkPolicyPeer
	// Add IPBlock rules for API server IPs
	for _, ip := range n.apiServerIPs {
		cidr := ip + "/32"
		apiSeverIPs = append(apiSeverIPs, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opentelemetry-operator",
			Namespace: n.operatorNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: *operatorDep.Spec.Selector,
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
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

	if n.apiServerPodSelector != nil || n.apiServerNamespaceSelector != nil {
		peer := networkingv1.NetworkPolicyPeer{
			PodSelector:       n.apiServerPodSelector,
			NamespaceSelector: n.apiServerNamespaceSelector,
		}
		np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &tcp,
					Port:     &apiServerPort,
				},
			},
			To: []networkingv1.NetworkPolicyPeer{peer},
		})
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
		np.Spec.Ingress[0].Ports = append(np.Spec.Ingress[0].Ports, networkingv1.NetworkPolicyPort{
			Protocol: &tcp,
			Port:     &metricsPort,
		})
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

// operatorDeployment resolves the Deployment the operator runs in by walking
// the owner references of its own pod. The deployment name cannot be assumed:
// it depends on the installation method (kustomize, Helm chart with name
// overrides, OLM).
func (n *networkPolicy) operatorDeployment(ctx context.Context) (*appsv1.Deployment, error) {
	pod, err := n.clientset.CoreV1().Pods(n.operatorNamespace).Get(ctx, n.operatorPodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator pod %q: %w", n.operatorPodName, err)
	}

	rsRef := metav1.GetControllerOf(pod)
	if rsRef == nil || rsRef.Kind != "ReplicaSet" {
		return nil, fmt.Errorf("operator pod %q is not owned by a ReplicaSet", pod.Name)
	}

	rs, err := n.clientset.AppsV1().ReplicaSets(n.operatorNamespace).Get(ctx, rsRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator ReplicaSet %q: %w", rsRef.Name, err)
	}

	depRef := metav1.GetControllerOf(rs)
	if depRef == nil || depRef.Kind != "Deployment" {
		return nil, fmt.Errorf("operator ReplicaSet %q is not owned by a Deployment", rs.Name)
	}

	dep, err := n.clientset.AppsV1().Deployments(n.operatorNamespace).Get(ctx, depRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator Deployment %q: %w", depRef.Name, err)
	}
	return dep, nil
}

func (*networkPolicy) NeedLeaderElection() bool {
	return true
}
