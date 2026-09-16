// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operatornetworkpolicy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
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

// startAndCapture runs Start() and captures the created NetworkPolicy
// by using a reactor that cancels the context after creation.
func startAndCapture(t *testing.T, clientset *fake.Clientset, scheme *runtime.Scheme, opts ...Option) *networkingv1.NetworkPolicy {
	t.Helper()

	var captured *networkingv1.NetworkPolicy
	ctx, cancel := context.WithCancel(context.Background())

	clientset.PrependReactor("create", "networkpolicies", func(action kubeTesting.Action) (bool, runtime.Object, error) {
		createAction := action.(kubeTesting.CreateAction)
		np := createAction.GetObject().(*networkingv1.NetworkPolicy)
		captured = np.DeepCopy()
		cancel()
		return false, np, nil
	})

	n := NewOperatorNetworkPolicy(clientset, scheme, opts...)
	err := n.(*networkPolicy).Start(ctx)
	require.NoError(t, err)
	require.NotNil(t, captured, "NetworkPolicy was not created")
	return captured
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

func TestStart_IPBlockPeersOnly(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestStart_SelectorPeersOnly(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestStart_CombinedIPBlockAndSelectors(t *testing.T) {
	const namespace = "openshift-opentelemetry-operator"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestStart_WithIngressPorts(t *testing.T) {
	const namespace = "test-ns"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestStart_FullOpenShiftConfig(t *testing.T) {
	const namespace = "openshift-opentelemetry-operator"
	clientset := fake.NewClientset(operatorObjects(namespace)...)

	podSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"apiserver": "true"},
	}
	nsSelector := &metav1.LabelSelector{
		MatchLabels: map[string]string{"kubernetes.io/metadata.name": "openshift-kube-apiserver"},
	}

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
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

func TestStart_PodSelectorFromDeployment(t *testing.T) {
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

	np := startAndCapture(t, clientset, newTestScheme(),
		WithOperatorNamespace(namespace),
		WithOperatorPodName(testPodName),
		WithAPIServerPort(6443),
		WithAPIServerIPs([]string{"10.0.0.1"}),
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
