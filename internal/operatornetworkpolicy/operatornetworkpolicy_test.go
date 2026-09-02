// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = discoveryv1.AddToScheme(s)
	return s
}

func operatorDeployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: namespace,
			UID:       "test-uid",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "manager", Image: "test"}},
				},
			},
		},
	}
}

func ownerRef() []metav1.OwnerReference {
	trueVal := true
	return []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               operatorName,
			UID:                types.UID("test-uid"),
			Controller:         &trueVal,
			BlockOwnerDeletion: &trueVal,
		},
	}
}

// buildAndCapture creates a networkPolicy with the given options and returns the built NetworkPolicy.
func buildAndCapture(t *testing.T, clientset *fake.Clientset, scheme *runtime.Scheme, opts ...Option) *networkingv1.NetworkPolicy {
	t.Helper()

	n := NewOperatorNetworkPolicy(clientset, scheme, opts...).(*networkPolicy)
	ctx := context.Background()

	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)
	return n.buildNetworkPolicy(ref)
}

func TestBuild_IPBlockPeersOnly(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	np := buildAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1", "10.0.0.2"}),
	)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	expected := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "opentelemetry-operator",
			Namespace:       namespace,
			OwnerReferences: ownerRef(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To: []networkingv1.NetworkPolicyPeer{
						{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.1/32"}},
						{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.2/32"}},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	assert.Equal(t, expected, np)
}

func TestBuild_SelectorPeersOnly(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := buildAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPISererPodLabelSelector(podSelector),
		WithAPISererNamespaceLabelSelector(nsSelector),
	)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	expected := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "opentelemetry-operator",
			Namespace:       namespace,
			OwnerReferences: ownerRef(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To: []networkingv1.NetworkPolicyPeer{
						{PodSelector: podSelector, NamespaceSelector: nsSelector},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	assert.Equal(t, expected, np)
}

func TestBuild_CombinedIPBlockAndSelectors(t *testing.T) {
	const namespace = "openshift-opentelemetry-operator"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := buildAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1"}),
		WithAPISererPodLabelSelector(podSelector),
		WithAPISererNamespaceLabelSelector(nsSelector),
	)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	expected := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "opentelemetry-operator",
			Namespace:       namespace,
			OwnerReferences: ownerRef(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{}},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To:    []networkingv1.NetworkPolicyPeer{{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.1/32"}}},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To:    []networkingv1.NetworkPolicyPeer{{PodSelector: podSelector, NamespaceSelector: nsSelector}},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	assert.Equal(t, expected, np)
}

func TestBuild_WithIngressPorts(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	np := buildAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1"}),
		WithWebhookPort(9443),
		WithMetricsPort(8443),
	)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	webhookPort := intstr.FromInt32(9443)
	metricsPort := intstr.FromInt32(8443)
	expected := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "opentelemetry-operator",
			Namespace:       namespace,
			OwnerReferences: ownerRef(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &webhookPort},
						{Protocol: &tcp, Port: &metricsPort},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To:    []networkingv1.NetworkPolicyPeer{{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.1/32"}}},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	assert.Equal(t, expected, np)
}

func TestBuild_FullOpenShiftConfig(t *testing.T) {
	const namespace = "openshift-opentelemetry-operator"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := buildAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1", "10.0.0.2"}),
		WithAPISererPodLabelSelector(podSelector),
		WithAPISererNamespaceLabelSelector(nsSelector),
		WithWebhookPort(9443),
		WithMetricsPort(8443),
	)

	tcp := corev1.ProtocolTCP
	apiServerPort := intstr.FromInt32(6443)
	webhookPort := intstr.FromInt32(9443)
	metricsPort := intstr.FromInt32(8443)
	expected := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "opentelemetry-operator",
			Namespace:       namespace,
			OwnerReferences: ownerRef(),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &webhookPort},
						{Protocol: &tcp, Port: &metricsPort},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To: []networkingv1.NetworkPolicyPeer{
						{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.1/32"}},
						{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.2/32"}},
					},
				},
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &apiServerPort}},
					To:    []networkingv1.NetworkPolicyPeer{{PodSelector: podSelector, NamespaceSelector: nsSelector}},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		},
	}

	assert.Equal(t, expected, np)
}

func TestNeedLeaderElection(t *testing.T) {
	n := &networkPolicy{}
	assert.True(t, n.NeedLeaderElection())
}

func TestCreateOrUpdate_CreatesWhenNotExist(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorDeployment(namespace))

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1"}),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	err = n.createOrUpdateNetworkPolicy(ctx, ref)
	require.NoError(t, err)

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, networkPolicyName, np.Name)
	assert.Len(t, np.Spec.Egress[0].To, 1)
	assert.Equal(t, "10.0.0.1/32", np.Spec.Egress[0].To[0].IPBlock.CIDR)
}

func TestCreateOrUpdate_UpdatesExistingPolicy(t *testing.T) {
	const namespace = "test-ns"

	tcp := corev1.ProtocolTCP
	oldPort := intstr.FromInt32(6443)
	existingNP := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPolicyName,
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "opentelemetry-operator"},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &oldPort}},
					To: []networkingv1.NetworkPolicyPeer{
						{IPBlock: &networkingv1.IPBlock{CIDR: "10.0.0.99/32"}},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
		},
	}

	clientset := fake.NewClientset(operatorDeployment(namespace), existingNP)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1", "10.0.0.2"}),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	err = n.createOrUpdateNetworkPolicy(ctx, ref)
	require.NoError(t, err)

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)

	require.Len(t, np.Spec.Egress, 1)
	require.Len(t, np.Spec.Egress[0].To, 2)
	assert.Equal(t, "10.0.0.1/32", np.Spec.Egress[0].To[0].IPBlock.CIDR)
	assert.Equal(t, "10.0.0.2/32", np.Spec.Egress[0].To[1].IPBlock.CIDR)
}

func TestIpsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		expected bool
	}{
		{
			name:     "equal same order",
			a:        []string{"10.0.0.1", "10.0.0.2"},
			b:        []string{"10.0.0.1", "10.0.0.2"},
			expected: true,
		},
		{
			name:     "equal different order",
			a:        []string{"10.0.0.2", "10.0.0.1"},
			b:        []string{"10.0.0.1", "10.0.0.2"},
			expected: true,
		},
		{
			name:     "not equal different IPs",
			a:        []string{"10.0.0.1"},
			b:        []string{"10.0.0.2"},
			expected: false,
		},
		{
			name:     "not equal different lengths",
			a:        []string{"10.0.0.1", "10.0.0.2"},
			b:        []string{"10.0.0.1"},
			expected: false,
		},
		{
			name:     "both empty",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one empty one not",
			a:        []string{"10.0.0.1"},
			b:        nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ipsEqual(tt.a, tt.b))
		})
	}
}

func TestHandleEndpointSliceEvent_UpdatesOnIPChange(t *testing.T) {
	const namespace = "test-ns"

	httpsPort := int32(6443)
	portName := "https"
	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
			Labels: map[string]string{
				"kubernetes.io/service-name": "kubernetes",
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name: &portName,
				Port: &httpsPort,
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.0.0.5", "10.0.0.6"}},
		},
	}

	clientset := fake.NewClientset(operatorDeployment(namespace), endpointSlice)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1", "10.0.0.2"}),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	err = n.createOrUpdateNetworkPolicy(ctx, ref)
	require.NoError(t, err)

	n.handleEndpointSliceEvent(ctx, ref)

	assert.Equal(t, []string{"10.0.0.5", "10.0.0.6"}, n.apiServerIPs)

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, np.Spec.Egress, 1)
	require.Len(t, np.Spec.Egress[0].To, 2)
	assert.Equal(t, "10.0.0.5/32", np.Spec.Egress[0].To[0].IPBlock.CIDR)
	assert.Equal(t, "10.0.0.6/32", np.Spec.Egress[0].To[1].IPBlock.CIDR)
}

func TestHandleEndpointSliceEvent_NoUpdateWhenIPsSame(t *testing.T) {
	const namespace = "test-ns"

	httpsPort := int32(6443)
	portName := "https"
	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
			Labels: map[string]string{
				"kubernetes.io/service-name": "kubernetes",
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name: &portName,
				Port: &httpsPort,
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.0.0.1", "10.0.0.2"}},
		},
	}

	clientset := fake.NewClientset(operatorDeployment(namespace), endpointSlice)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1", "10.0.0.2"}),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	err = n.createOrUpdateNetworkPolicy(ctx, ref)
	require.NoError(t, err)

	np1, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	rv1 := np1.ResourceVersion

	n.handleEndpointSliceEvent(ctx, ref)

	np2, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, rv1, np2.ResourceVersion, "NetworkPolicy should not have been updated when IPs haven't changed")
}

func TestStart_CreatesAndWatches(t *testing.T) {
	const namespace = "test-ns"

	httpsPort := int32(6443)
	portName := "https"
	endpointSlice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
			Labels: map[string]string{
				"kubernetes.io/service-name": "kubernetes",
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name: &portName,
				Port: &httpsPort,
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.0.0.1"}},
		},
	}

	clientset := fake.NewClientset(operatorDeployment(namespace), endpointSlice)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1"}),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx, cancel := context.WithCancel(context.Background())

	startDone := make(chan error, 1)
	go func() {
		startDone <- n.Start(ctx)
	}()

	// Give the informer time to start
	time.Sleep(200 * time.Millisecond)

	// Verify the NetworkPolicy was created
	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, networkPolicyName, np.Name)

	cancel()

	err = <-startDone
	assert.NoError(t, err)
}

func TestWithLogger(t *testing.T) {
	n := &networkPolicy{}
	logger := logr.Discard()
	WithLogger(logger)(n)
	assert.Equal(t, logger, n.logger)
}
