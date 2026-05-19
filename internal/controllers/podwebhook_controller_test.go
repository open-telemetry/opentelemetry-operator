// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newPodWebhookDeployment(replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podWebhookDeploymentName,
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "pod-webhook"},
			},
		},
	}
}

func TestPodWebhookReconciler_ScaleDown(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 1,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(1), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_ScaleUp(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(1)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 2,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_NoChangeWhenAlreadyAtDesired(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 2,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_IgnoreScaleBeyondCSVDefault(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 5,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_IgnoreOtherDeployments(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 1,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "some-other-deployment", Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_IgnoreWrongNamespace(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 1,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: "other-ns"},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestPodWebhookReconciler_DeploymentNotFound(t *testing.T) {
	ns := "test-ns"

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 1,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestPodWebhookReconciler_ScaleToZero(t *testing.T) {
	ns := "test-ns"
	dep := newPodWebhookDeployment(2)

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dep).Build()

	r := &PodWebhookReconciler{
		Client:          cl,
		Namespace:       ns,
		DesiredReplicas: 0,
	}

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns},
	})
	require.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)

	updated := &appsv1.Deployment{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Name: podWebhookDeploymentName, Namespace: ns}, updated))
	assert.Equal(t, int32(0), *updated.Spec.Replicas)
}
