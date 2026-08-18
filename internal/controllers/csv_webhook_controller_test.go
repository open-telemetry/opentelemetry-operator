// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createTestCSV(name, namespace string, deployments []operatorsv1alpha1.StrategyDeploymentSpec) *operatorsv1alpha1.ClusterServiceVersion {
	return &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			InstallStrategy: operatorsv1alpha1.NamedInstallStrategy{
				StrategySpec: operatorsv1alpha1.StrategyDetailsDeployment{
					DeploymentSpecs: deployments,
				},
			},
		},
	}
}

func createDeploymentSpec(name string, replicas int32) operatorsv1alpha1.StrategyDeploymentSpec {
	return operatorsv1alpha1.StrategyDeploymentSpec{
		Name: name,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "webhook", Image: "test:latest"},
					},
				},
			},
		},
	}
}

func TestCSVWebhookReconciler_Reconcile(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			require.NoError(t, operatorsv1alpha1.AddToScheme(s))

			csv := createTestCSV("opentelemetry-operator.v0.100.0", testNamespace, []operatorsv1alpha1.StrategyDeploymentSpec{
				createDeploymentSpec("opentelemetry-operator-controller-manager", 1),
				createDeploymentSpec(podWebhookDeploymentName, tt.currentReplicas),
			})

			client := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(csv).
				Build()

			reconciler := &CSVWebhookReconciler{
				Client:          client,
				Namespace:       testNamespace,
				DesiredReplicas: tt.desiredReplicas,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      csv.Name,
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			// Verify the CSV was updated
			updatedCSV := &operatorsv1alpha1.ClusterServiceVersion{}
			err = client.Get(context.Background(), req.NamespacedName, updatedCSV)
			require.NoError(t, err)

			// Find pod-webhook deployment spec in CSV
			var podWebhookReplicas int32 = -1
			for _, ds := range updatedCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
				if ds.Name == podWebhookDeploymentName {
					if ds.Spec.Replicas != nil {
						podWebhookReplicas = *ds.Spec.Replicas
					}
					break
				}
			}
			assert.Equal(t, tt.expectedReplicas, podWebhookReplicas)
		})
	}
}

func TestCSVWebhookReconciler_CSVNotFound(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, operatorsv1alpha1.AddToScheme(s))

	client := fake.NewClientBuilder().
		WithScheme(s).
		Build()

	reconciler := &CSVWebhookReconciler{
		Client:          client,
		Namespace:       "test",
		DesiredReplicas: 2,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent-csv",
			Namespace: "test",
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestCSVWebhookReconciler_IgnoreOtherNamespaces(t *testing.T) {
	testNamespace := "opentelemetry-operator-system"
	otherNamespace := "other-namespace"

	s := runtime.NewScheme()
	require.NoError(t, operatorsv1alpha1.AddToScheme(s))

	csv := createTestCSV("opentelemetry-operator.v0.100.0", otherNamespace, []operatorsv1alpha1.StrategyDeploymentSpec{
		createDeploymentSpec(podWebhookDeploymentName, 1),
	})

	client := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(csv).
		Build()

	reconciler := &CSVWebhookReconciler{
		Client:          client,
		Namespace:       testNamespace,
		DesiredReplicas: 2,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      csv.Name,
			Namespace: otherNamespace,
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify CSV was NOT modified (still has 1 replica)
	updatedCSV := &operatorsv1alpha1.ClusterServiceVersion{}
	err = client.Get(context.Background(), req.NamespacedName, updatedCSV)
	require.NoError(t, err)

	var podWebhookReplicas int32 = -1
	for _, ds := range updatedCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		if ds.Name == podWebhookDeploymentName {
			if ds.Spec.Replicas != nil {
				podWebhookReplicas = *ds.Spec.Replicas
			}
			break
		}
	}
	assert.Equal(t, int32(1), podWebhookReplicas)
}

func TestCSVWebhookReconciler_CSVWithoutPodWebhook(t *testing.T) {
	testNamespace := "opentelemetry-operator-system"

	s := runtime.NewScheme()
	require.NoError(t, operatorsv1alpha1.AddToScheme(s))

	// CSV without pod-webhook deployment
	csv := createTestCSV("some-other-operator.v1.0.0", testNamespace, []operatorsv1alpha1.StrategyDeploymentSpec{
		createDeploymentSpec("some-other-deployment", 1),
	})

	client := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(csv).
		Build()

	reconciler := &CSVWebhookReconciler{
		Client:          client,
		Namespace:       testNamespace,
		DesiredReplicas: 2,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      csv.Name,
			Namespace: testNamespace,
		},
	}

	result, err := reconciler.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify CSV was NOT modified
	updatedCSV := &operatorsv1alpha1.ClusterServiceVersion{}
	err = client.Get(context.Background(), req.NamespacedName, updatedCSV)
	require.NoError(t, err)

	// Should still have only 1 deployment with 1 replica
	require.Len(t, updatedCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs, 1)
	assert.Equal(t, int32(1), *updatedCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[0].Spec.Replicas)
}
