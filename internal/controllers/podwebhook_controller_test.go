// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPodWebhookReconciler_Reconcile(t *testing.T) {
	testNamespace := "opentelemetry-operator-system"

	tests := []struct {
		name             string
		currentReplicas  int32
		desiredReplicas  int32
		expectUpdate     bool
		expectedReplicas int32
	}{
		{
			name:             "scale up from 1 to 2",
			currentReplicas:  1,
			desiredReplicas:  2,
			expectUpdate:     true,
			expectedReplicas: 2,
		},
		{
			name:             "scale down from 2 to 1",
			currentReplicas:  2,
			desiredReplicas:  1,
			expectUpdate:     true,
			expectedReplicas: 1,
		},
		{
			name:             "no change when replicas match",
			currentReplicas:  2,
			desiredReplicas:  2,
			expectUpdate:     false,
			expectedReplicas: 2,
		},
		{
			name:             "scale to 0 disables webhook",
			currentReplicas:  2,
			desiredReplicas:  0,
			expectUpdate:     true,
			expectedReplicas: 0,
		},
		{
			name:             "refuse to scale beyond CSV default",
			currentReplicas:  2,
			desiredReplicas:  5,
			expectUpdate:     false,
			expectedReplicas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			require.NoError(t, scheme.AddToScheme(s))
			require.NoError(t, appsv1.AddToScheme(s))

			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podWebhookDeploymentName,
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: &tt.currentReplicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "pod-webhook"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "pod-webhook"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "webhook", Image: "test:latest"},
							},
						},
					},
				},
			}

			client := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(deployment).
				Build()

			reconciler := &PodWebhookReconciler{
				Client:          client,
				Namespace:       testNamespace,
				DesiredReplicas: tt.desiredReplicas,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      podWebhookDeploymentName,
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			// Verify the deployment was updated
			updatedDeployment := &appsv1.Deployment{}
			err = client.Get(context.Background(), req.NamespacedName, updatedDeployment)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedReplicas, *updatedDeployment.Spec.Replicas)
		})
	}
}

func TestPodWebhookReconciler_DeploymentNotFound(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, scheme.AddToScheme(s))
	require.NoError(t, appsv1.AddToScheme(s))

	client := fake.NewClientBuilder().
		WithScheme(s).
		Build()

	reconciler := &PodWebhookReconciler{
		Client:          client,
		Namespace:       "test",
		DesiredReplicas: 2,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      podWebhookDeploymentName,
			Namespace: "test",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestPodWebhookReconciler_IgnoreOtherDeployments(t *testing.T) {
	testNamespace := "opentelemetry-operator-system"

	s := runtime.NewScheme()
	require.NoError(t, scheme.AddToScheme(s))
	require.NoError(t, appsv1.AddToScheme(s))

	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-other-deployment",
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "other"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "other"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "other", Image: "test:latest"},
					},
				},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(deployment).
		Build()

	reconciler := &PodWebhookReconciler{
		Client:          client,
		Namespace:       testNamespace,
		DesiredReplicas: 2,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "some-other-deployment",
			Namespace: testNamespace,
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify deployment was NOT modified
	updatedDeployment := &appsv1.Deployment{}
	err = client.Get(context.Background(), req.NamespacedName, updatedDeployment)
	require.NoError(t, err)
	assert.Equal(t, int32(1), *updatedDeployment.Spec.Replicas)
}
