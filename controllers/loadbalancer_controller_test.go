// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/reconcile"
)

var lblogger = logf.Log.WithName("unit-tests")

func TestNewObjectsOnLbReconciliation(t *testing.T) {
	// prepare
	cfg := config.New()
	configYAML, err := ioutil.ReadFile("../pkg/loadbalancer/reconcile/suite_test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewLbReconciler(controllers.LbParams{
		Client: k8sClient,
		Log:    lblogger,
		Scheme: testScheme,
		Config: cfg,
	})
	created := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode:   v1alpha1.ModeStatefulSet,
			Config: string(configYAML),
			LoadBalancer: v1alpha1.OpenTelemetryLoadBalancer{
				Mode: v1alpha1.ModeLeastConnection,
			},
		},
	}
	err = k8sClient.Create(context.Background(), created)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Name, "loadbalancer"),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}

	// verify that we have at least one object for each of the types we create
	// whether we have the right ones is up to the specific tests for each type
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &corev1.ServiceList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))

}

func TestContinueOnRecoverableLbFailure(t *testing.T) {
	// prepare
	taskCalled := false
	reconciler := controllers.NewLbReconciler(controllers.LbParams{
		Log: lblogger,
		Tasks: []controllers.LbTask{
			{
				Name: "should-fail",
				Do: func(context.Context, reconcile.Params) error {
					return errors.New("should fail!")
				},
				BailOnError: false,
			},
			{
				Name: "should-be-called",
				Do: func(context.Context, reconcile.Params) error {
					taskCalled = true
					return nil
				},
			},
		},
	})

	// test
	err := reconciler.RunTasks(context.Background(), reconcile.Params{})

	// verify
	assert.NoError(t, err)
	assert.True(t, taskCalled)
}

func TestBreakOnUnrecoverableLbError(t *testing.T) {
	// prepare
	cfg := config.New()
	taskCalled := false
	expectedErr := errors.New("should fail!")
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewLbReconciler(controllers.LbParams{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.LbTask{
			{
				Name: "should-fail",
				Do: func(context.Context, reconcile.Params) error {
					taskCalled = true
					return expectedErr
				},
				BailOnError: true,
			},
			{
				Name: "should-not-be-called",
				Do: func(context.Context, reconcile.Params) error {
					assert.Fail(t, "should not have been called")
					return nil
				},
			},
		},
	})
	created := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}
	err := k8sClient.Create(context.Background(), created)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	assert.Equal(t, expectedErr, err)
	assert.True(t, taskCalled)

	// cleanup
	assert.NoError(t, k8sClient.Delete(context.Background(), created))
}

func TestLbSkipWhenInstanceDoesNotExist(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewLbReconciler(controllers.LbParams{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.LbTask{
			{
				Name: "should-not-be-called",
				Do: func(context.Context, reconcile.Params) error {
					assert.Fail(t, "should not have been called")
					return nil
				},
			},
		},
	})

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err := reconciler.Reconcile(context.Background(), req)

	// verify
	assert.NoError(t, err)
}
