// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

import (
	"context"
	"testing"

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlFake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/internal/apiserverendpoints"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = discoveryv1.AddToScheme(s)
	return s
}

// A Helm-style deployment name proves nothing depends on the kustomize name
// opentelemetry-operator-controller-manager anymore.
const (
	operatorDeploymentName = "my-release-opentelemetry-operator"
	testReplicaSetName     = operatorDeploymentName + "-6b9f7c"
	testPodName            = testReplicaSetName + "-x2x9z"
)

// operatorObjects returns the pod → ReplicaSet → Deployment ownership chain
// Start() walks to resolve the operator's own deployment.
func operatorObjects(namespace string) []runtime.Object {
	trueVal := true
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorDeploymentName,
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
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testReplicaSetName,
			Namespace: namespace,
			UID:       "rs-uid",
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "Deployment", Name: operatorDeploymentName, UID: "test-uid", Controller: &trueVal},
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: namespace,
			UID:       "pod-uid",
			OwnerReferences: []metav1.OwnerReference{
				{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: testReplicaSetName, UID: "rs-uid", Controller: &trueVal},
			},
		},
	}
	return []runtime.Object{dep, rs, pod}
}

func ownerRef() []metav1.OwnerReference {
	trueVal := true
	return []metav1.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               operatorDeploymentName,
			UID:                types.UID("test-uid"),
			Controller:         &trueVal,
			BlockOwnerDeletion: &trueVal,
		},
	}
}

// syncedReader wraps a client.Reader to satisfy apiserverendpoints.CacheReader.
type syncedReader struct {
	client.Reader
}

func (syncedReader) WaitForCacheSync(context.Context) bool {
	return true
}

func newTestTracker(t *testing.T, sliceObjects ...client.Object) *apiserverendpoints.Tracker {
	t.Helper()
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	c := ctrlFake.NewClientBuilder().WithObjects(sliceObjects...).Build()
	return apiserverendpoints.NewTracker(syncedReader{c}, apiserverendpoints.DefaultTTL)
}

func apiServerEndpointSlice(port int32, addresses ...string) *discoveryv1.EndpointSlice {
	portName := "https"
	var eps []discoveryv1.Endpoint
	for _, addr := range addresses {
		eps = append(eps, discoveryv1.Endpoint{Addresses: []string{addr}})
	}
	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
			Labels:    map[string]string{discoveryv1.LabelServiceName: "kubernetes"},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Ports:       []discoveryv1.EndpointPort{{Name: &portName, Port: &port}},
		Endpoints:   eps,
	}
}

// buildAndCapture creates a networkPolicy with the given options and returns the built NetworkPolicy.
func buildAndCapture(t *testing.T, clientset *fake.Clientset, scheme *runtime.Scheme, endpoints []apiserverendpoints.Endpoint, opts ...Option) *networkingv1.NetworkPolicy {
	t.Helper()

	n := NewOperatorNetworkPolicy(clientset, scheme, opts...).(*networkPolicy)
	ctx := context.Background()

	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)
	return n.buildNetworkPolicy(ref, endpoints)
}

func TestBuild_IPBlockPeersOnly(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
		{IP: "10.0.0.2", Port: 6443},
	}
	np := buildAndCapture(t, clientset, newTestScheme(), endpoints,
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestBuild_CombinedIPBlockAndSelectors(t *testing.T) {
	const namespace = "openshift-opentelemetry-operator"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
	}
	np := buildAndCapture(t, clientset, newTestScheme(), endpoints,
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
	}
	np := buildAndCapture(t, clientset, newTestScheme(), endpoints,
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
		{IP: "10.0.0.2", Port: 6443},
	}
	np := buildAndCapture(t, clientset, newTestScheme(), endpoints,
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestBuild_PodSelectorFromDeployment(t *testing.T) {
	const namespace = "test-ns"
	customSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/name": "custom-operator",
			"control-plane":          "controller-manager",
		},
	}
	objects := operatorObjects(namespace)
	objects[0].(*appsv1.Deployment).Spec.Selector = customSelector
	clientset := fake.NewClientset(objects...)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
	}
	np := buildAndCapture(t, clientset, newTestScheme(), endpoints,
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
	)

	assert.Equal(t, *customSelector, np.Spec.PodSelector)
}

func TestStart_PodNotFound(t *testing.T) {
	clientset := fake.NewClientset()

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace("test-ns"),
		WithOperatorPodName("missing-pod"),
	)
	err := n.(*networkPolicy).Start(context.Background())
	require.ErrorContains(t, err, `failed to get operator pod "missing-pod"`)
}

func TestStart_PodWithoutReplicaSetOwner(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "bare-pod", Namespace: "test-ns"},
	}
	clientset := fake.NewClientset(pod)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace("test-ns"),
		WithOperatorPodName("bare-pod"),
	)
	err := n.(*networkPolicy).Start(context.Background())
	require.ErrorContains(t, err, "not owned by a ReplicaSet")
}

func TestStart_ReplicaSetNotFound(t *testing.T) {
	const namespace = "test-ns"
	pod := operatorObjects(namespace)[2]
	clientset := fake.NewClientset(pod)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
	)
	err := n.(*networkPolicy).Start(context.Background())
	require.ErrorContains(t, err, "failed to get operator ReplicaSet")
}

func TestStart_ReplicaSetWithoutDeploymentOwner(t *testing.T) {
	const namespace = "test-ns"
	objects := operatorObjects(namespace)
	objects[1].(*appsv1.ReplicaSet).OwnerReferences = nil
	clientset := fake.NewClientset(objects[1], objects[2])

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
	)
	err := n.(*networkPolicy).Start(context.Background())
	require.ErrorContains(t, err, "not owned by a Deployment")
}

func TestStart_DeploymentNotFound(t *testing.T) {
	const namespace = "test-ns"
	objects := operatorObjects(namespace)
	clientset := fake.NewClientset(objects[1], objects[2])

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
	)
	err := n.(*networkPolicy).Start(context.Background())
	require.ErrorContains(t, err, "failed to get operator Deployment")
}

func TestNeedLeaderElection(t *testing.T) {
	n := &networkPolicy{}
	assert.True(t, n.NeedLeaderElection())
}

func TestCreateOrUpdate_CreatesWhenNotExist(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
	}
	err = n.createOrUpdateNetworkPolicy(ctx, ref, endpoints)
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

	clientset := fake.NewClientset(append(operatorObjects(namespace), existingNP)...)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
		{IP: "10.0.0.2", Port: 6443},
	}
	err = n.createOrUpdateNetworkPolicy(ctx, ref, endpoints)
	require.NoError(t, err)

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)

	require.Len(t, np.Spec.Egress, 1)
	require.Len(t, np.Spec.Egress[0].To, 2)
	assert.Equal(t, "10.0.0.1/32", np.Spec.Egress[0].To[0].IPBlock.CIDR)
	assert.Equal(t, "10.0.0.2/32", np.Spec.Egress[0].To[1].IPBlock.CIDR)
}

func TestHandleEndpointSliceEvent_UpdatesOnIPChange(t *testing.T) {
	const namespace = "test-ns"

	tracker := newTestTracker(t, apiServerEndpointSlice(6443, "10.0.0.5", "10.0.0.6"))
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
		WithTracker(tracker),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	oldEndpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
		{IP: "10.0.0.2", Port: 6443},
	}
	n.apiServerEndpoints = oldEndpoints
	err = n.createOrUpdateNetworkPolicy(ctx, ref, oldEndpoints)
	require.NoError(t, err)

	n.handleEndpointSliceEvent(ctx, ref)

	assert.Equal(t, []apiserverendpoints.Endpoint{
		{IP: "10.0.0.5", Port: 6443},
		{IP: "10.0.0.6", Port: 6443},
	}, n.apiServerEndpoints)

	np, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, np.Spec.Egress, 1)
	require.Len(t, np.Spec.Egress[0].To, 2)
	assert.Equal(t, "10.0.0.5/32", np.Spec.Egress[0].To[0].IPBlock.CIDR)
	assert.Equal(t, "10.0.0.6/32", np.Spec.Egress[0].To[1].IPBlock.CIDR)
}

func TestHandleEndpointSliceEvent_NoUpdateWhenEndpointsSame(t *testing.T) {
	const namespace = "test-ns"

	tracker := newTestTracker(t, apiServerEndpointSlice(6443, "10.0.0.1", "10.0.0.2"))
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	n := NewOperatorNetworkPolicy(clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
		WithTracker(tracker),
		WithLogger(logr.Discard()),
	).(*networkPolicy)

	ctx := context.Background()
	ref, err := n.getOwnerReference(ctx)
	require.NoError(t, err)

	endpoints := []apiserverendpoints.Endpoint{
		{IP: "10.0.0.1", Port: 6443},
		{IP: "10.0.0.2", Port: 6443},
	}
	n.apiServerEndpoints = endpoints
	err = n.createOrUpdateNetworkPolicy(ctx, ref, endpoints)
	require.NoError(t, err)

	np1, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	rv1 := np1.ResourceVersion

	n.handleEndpointSliceEvent(ctx, ref)

	np2, err := clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, rv1, np2.ResourceVersion, "NetworkPolicy should not have been updated when endpoints haven't changed")
}

func TestApiServerEgressRules(t *testing.T) {
	tcp := corev1.ProtocolTCP

	t.Run("single port", func(t *testing.T) {
		endpoints := []apiserverendpoints.Endpoint{
			{IP: "10.0.0.1", Port: 6443},
			{IP: "10.0.0.2", Port: 6443},
		}
		rules := apiServerEgressRules(endpoints)
		require.Len(t, rules, 1)
		port := intstr.FromInt32(6443)
		assert.Equal(t, []networkingv1.NetworkPolicyPort{{Protocol: &tcp, Port: &port}}, rules[0].Ports)
		assert.Len(t, rules[0].To, 2)
	})

	t.Run("multiple ports", func(t *testing.T) {
		endpoints := []apiserverendpoints.Endpoint{
			{IP: "10.96.0.1", Port: 443},
			{IP: "10.0.0.1", Port: 6443},
		}
		rules := apiServerEgressRules(endpoints)
		require.Len(t, rules, 2)
	})

	t.Run("empty", func(t *testing.T) {
		rules := apiServerEgressRules(nil)
		assert.Empty(t, rules)
	})
}

func TestWithLogger(t *testing.T) {
	n := &networkPolicy{}
	logger := logr.Discard()
	WithLogger(logger)(n)
	assert.Equal(t, logger, n.logger)
}
