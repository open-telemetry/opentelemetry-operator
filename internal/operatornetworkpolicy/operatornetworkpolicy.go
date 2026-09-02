// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

import (
	"context"
	"slices"
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	operatorName       = "opentelemetry-operator-controller-manager"
	networkPolicyName  = "opentelemetry-operator"
	endpointSliceLabel = "kubernetes.io/service-name=kubernetes"
)

type networkPolicy struct {
	clientset kubernetes.Interface
	scheme    *runtime.Scheme
	logger    logr.Logger

	operatorNamespace          string
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
		logger:    logr.Discard(),
	}

	for _, opt := range options {
		opt(n)
	}
	return n
}

type Option func(policy *networkPolicy)

// WithLogger sets the logger for the network policy reconciler.
func WithLogger(logger logr.Logger) Option {
	return func(s *networkPolicy) {
		s.logger = logger
	}
}

// WithOperatorNamespace sets the namespace of the operator and enables it in the network policy.
func WithOperatorNamespace(operatorNamespace string) Option {
	return func(s *networkPolicy) {
		s.operatorNamespace = operatorNamespace
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
	ownerRef, err := n.getOwnerReference(ctx)
	if err != nil {
		return err
	}

	if err := n.createOrUpdateNetworkPolicy(ctx, ownerRef); err != nil {
		return err
	}

	n.logger.Info("Starting EndpointSlice watcher for API server IP changes")
	return n.watchEndpointSlices(ctx, ownerRef)
}

// watchEndpointSlices sets up an informer on EndpointSlices in the default namespace
// filtered by the kubernetes service label. When IPs change, it updates the NetworkPolicy.
func (n *networkPolicy) watchEndpointSlices(ctx context.Context, ownerRef []metav1.OwnerReference) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		n.clientset,
		0,
		informers.WithNamespace("default"),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = endpointSliceLabel
		}),
	)

	informer := factory.Discovery().V1().EndpointSlices().Informer()

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			n.handleEndpointSliceEvent(ctx, ownerRef)
		},
		UpdateFunc: func(_, _ interface{}) {
			n.handleEndpointSliceEvent(ctx, ownerRef)
		},
		DeleteFunc: func(obj interface{}) {
			n.handleEndpointSliceEvent(ctx, ownerRef)
		},
	}

	if _, err := informer.AddEventHandler(handler); err != nil {
		return err
	}

	informer.Run(ctx.Done())
	return nil
}

// handleEndpointSliceEvent is called on any EndpointSlice event. It re-reads all matching
// EndpointSlices, extracts the current API server IPs, and updates the NetworkPolicy if changed.
func (n *networkPolicy) handleEndpointSliceEvent(ctx context.Context, ownerRef []metav1.OwnerReference) {
	newIPs, err := n.discoverAPIServerIPs(ctx)
	if err != nil {
		n.logger.Error(err, "Failed to discover API server IPs from EndpointSlices")
		return
	}

	if ipsEqual(n.apiServerIPs, newIPs) {
		return
	}

	n.logger.Info("API server IPs changed, updating NetworkPolicy", "oldIPs", n.apiServerIPs, "newIPs", newIPs)
	n.apiServerIPs = newIPs

	if err := n.createOrUpdateNetworkPolicy(ctx, ownerRef); err != nil {
		n.logger.Error(err, "Failed to update NetworkPolicy after IP change")
	}
}

// discoverAPIServerIPs lists EndpointSlices for the kubernetes service and extracts endpoint IPs.
func (n *networkPolicy) discoverAPIServerIPs(ctx context.Context) ([]string, error) {
	endpointSlices, err := n.clientset.DiscoveryV1().EndpointSlices("default").List(ctx, metav1.ListOptions{
		LabelSelector: endpointSliceLabel,
	})
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, es := range endpointSlices.Items {
		for _, endpoint := range es.Endpoints {
			ips = append(ips, endpoint.Addresses...)
		}
	}
	sort.Strings(ips)
	return ips, nil
}

// ipsEqual compares two IP slices for equality, ignoring order.
func ipsEqual(a, b []string) bool {
	sortedA := make([]string, len(a))
	copy(sortedA, a)
	sort.Strings(sortedA)

	sortedB := make([]string, len(b))
	copy(sortedB, b)
	sort.Strings(sortedB)

	return slices.Equal(sortedA, sortedB)
}

// getOwnerReference fetches the operator deployment and returns the owner reference for it.
func (n *networkPolicy) getOwnerReference(ctx context.Context) ([]metav1.OwnerReference, error) {
	operatorDep, err := n.clientset.AppsV1().Deployments(n.operatorNamespace).Get(ctx, operatorName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPolicyName,
			Namespace: n.operatorNamespace,
		},
	}
	if err := controllerutil.SetControllerReference(operatorDep, np, n.scheme); err != nil {
		return nil, err
	}
	return np.OwnerReferences, nil
}

// buildNetworkPolicy constructs the desired NetworkPolicy object from the current state.
func (n *networkPolicy) buildNetworkPolicy(ownerRef []metav1.OwnerReference) *networkingv1.NetworkPolicy {
	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(n.apiServerPort)

	var apiServerPeers []networkingv1.NetworkPolicyPeer
	for _, ip := range n.apiServerIPs {
		cidr := ip + "/32"
		apiServerPeers = append(apiServerPeers, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	}

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            networkPolicyName,
			Namespace:       n.operatorNamespace,
			OwnerReferences: ownerRef,
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
					To: apiServerPeers,
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

	return np
}

// createOrUpdateNetworkPolicy creates the NetworkPolicy if it doesn't exist, or updates it if it does.
func (n *networkPolicy) createOrUpdateNetworkPolicy(ctx context.Context, ownerRef []metav1.OwnerReference) error {
	desired := n.buildNetworkPolicy(ownerRef)

	_, err := n.clientset.NetworkingV1().NetworkPolicies(n.operatorNamespace).Create(ctx, desired, metav1.CreateOptions{})
	if err == nil {
		n.logger.Info("Created NetworkPolicy", "name", networkPolicyName, "namespace", n.operatorNamespace)
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return err
	}

	existing, err := n.clientset.NetworkingV1().NetworkPolicies(n.operatorNamespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	existing.Spec = desired.Spec
	existing.OwnerReferences = desired.OwnerReferences
	_, err = n.clientset.NetworkingV1().NetworkPolicies(n.operatorNamespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	n.logger.Info("Updated NetworkPolicy", "name", networkPolicyName, "namespace", n.operatorNamespace)
	return nil
}

// extractIPsFromEndpointSlice extracts endpoint addresses from an EndpointSlice.
func extractIPsFromEndpointSlice(obj interface{}) []string {
	es, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return nil
	}
	var ips []string
	for _, endpoint := range es.Endpoints {
		ips = append(ips, endpoint.Addresses...)
	}
	return ips
}

func (*networkPolicy) NeedLeaderElection() bool {
	return true
}
