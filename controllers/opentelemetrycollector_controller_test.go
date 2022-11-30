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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/reconcile"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

var logger = logf.Log.WithName("unit-tests")
var mockAutoDetector = &mockAutoDetect{
	HPAVersionFunc: func() (autodetect.AutoscalingVersion, error) {
		return autodetect.AutoscalingVersionV2Beta2, nil
	},
}

func TestNewObjectsOnReconciliation(t *testing.T) {
	// prepare
	cfg := config.New(config.WithCollectorImage("default-collector"), config.WithTargetAllocatorImage("default-ta-allocator"), config.WithAutoDetect(mockAutoDetector))
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewReconciler(controllers.Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
		Config: cfg,
	})
	created := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode: v1alpha1.ModeDeployment,
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
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
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
		list := &corev1.ServiceAccountList{}
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
	{
		list := &appsv1.DaemonSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		// attention! we expect daemonsets to be empty in the default configuration
		assert.Empty(t, list.Items)
	}
	{
		list := &appsv1.StatefulSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		// attention! we expect statefulsets to be empty in the default configuration
		assert.Empty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))

}

func TestNewStatefulSetObjectsOnReconciliation(t *testing.T) {
	// prepare
	cfg := config.New(config.WithAutoDetect(mockAutoDetector))
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewReconciler(controllers.Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
		Config: cfg,
	})
	created := &v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode: v1alpha1.ModeStatefulSet,
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
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", nsn.Namespace, nsn.Name),
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
		list := &corev1.ServiceAccountList{}
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
		list := &appsv1.StatefulSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		// attention! we expect deployments to be empty when starting in the statefulset mode.
		assert.Empty(t, list.Items)
	}
	{
		list := &appsv1.DaemonSetList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		// attention! we expect daemonsets to be empty when starting in the statefulset mode.
		assert.Empty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))

}

func TestContinueOnRecoverableFailure(t *testing.T) {
	// prepare
	taskCalled := false
	reconciler := controllers.NewReconciler(controllers.Params{
		Log: logger,
		Tasks: []controllers.Task{
			{
				Name: "should-fail",
				Do: func(context.Context, reconcile.Params) error {
					return errors.New("should fail")
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

func TestBreakOnUnrecoverableError(t *testing.T) {
	// prepare
	cfg := config.New()
	taskCalled := false
	expectedErr := errors.New("should fail")
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewReconciler(controllers.Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.Task{
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

func TestSkipWhenInstanceDoesNotExist(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewReconciler(controllers.Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.Task{
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

func TestRegisterWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)

	reconciler := controllers.NewReconciler(controllers.Params{})

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	assert.NoError(t, err)
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	PlatformFunc   func() (platform.Platform, error)
	HPAVersionFunc func() (autodetect.AutoscalingVersion, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	return m.HPAVersionFunc()
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
