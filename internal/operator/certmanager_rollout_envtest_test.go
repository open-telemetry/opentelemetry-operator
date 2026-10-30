// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/internal/testenv"
)

func TestCertManagerRolloutWithAPIServer(t *testing.T) {
	ctx := context.Background()
	testScheme := runtime.NewScheme()
	require.NoError(t, scheme.AddToScheme(testScheme))
	environment, err := testenv.Start(&envtest.Environment{}, testScheme)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, environment.Stop()) })
	clientset, err := kubernetes.NewForConfig(environment.Config)
	require.NoError(t, err)

	namespace, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "operator-rollout-test"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	labels := map[string]string{"app": "operator"}
	template := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: labels},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "operator", Image: "operator:test"}}},
	}
	deployment, err := clientset.AppsV1().Deployments(namespace.Name).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "operator"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: template,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	controller := true
	rs, err := clientset.AppsV1().ReplicaSets(namespace.Name).Create(ctx, &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "operator-rs", OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployment.Name, UID: deployment.UID, Controller: &controller}}},
		Spec:       appsv1.ReplicaSetSpec{Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: template},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	for _, name := range []string{"operator-pod-1", "operator-pod-2"} {
		_, err = clientset.CoreV1().Pods(namespace.Name).Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels, OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: rs.Name, UID: rs.UID, Controller: &controller}}},
			Spec:       template.Spec,
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	rollout := certManagerRollout{clientset: clientset}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, name := range []string{"operator-pod-1", "operator-pod-2"} {
		go func() {
			<-start
			results <- rollout.rolloutDeployment(ctx, namespace.Name, name)
		}()
	}
	close(start)
	for range 2 {
		rolloutErr := <-results
		require.True(t, rolloutErr == nil || apierrors.IsConflict(rolloutErr), "unexpected rollout error: %v", rolloutErr)
	}
	updated, err := clientset.AppsV1().Deployments(namespace.Name).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, updated.Spec.Template.Annotations[certManagerRolloutAnnotation])
	require.NoError(t, rollout.rolloutDeployment(ctx, namespace.Name, "operator-pod-2"))
	again, err := clientset.AppsV1().Deployments(namespace.Name).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, updated.ResourceVersion, again.ResourceVersion)
}
