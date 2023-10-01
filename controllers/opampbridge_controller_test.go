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
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
)

var opampBridgeLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")
var opampBridgeMockAutoDetector = &mockAutoDetect{
	HPAVersionFunc: func() (autodetect.AutoscalingVersion, error) {
		return autodetect.AutoscalingVersionV2Beta2, nil
	},
	OpenShiftRoutesAvailabilityFunc: func() (autodetect.OpenShiftRoutesAvailability, error) {
		return autodetect.OpenShiftRoutesAvailable, nil
	},
}

func TestNewObjectsOnReconciliation_OpAMPBridge(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithOperatorOpAMPBridgeImage("default-opamp-bridge"),
		config.WithAutoDetect(opampBridgeMockAutoDetector),
	)
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client:   k8sClient,
		Log:      logger,
		Scheme:   testScheme,
		Recorder: record.NewFakeRecorder(10),
		Config:   cfg,
	})
	require.NoError(t, cfg.AutoDetect())
	created := &v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Endpoint:     "ws://opamp-server:4320/v1/opamp",
			Protocol:     "wss",
			Capabilities: []v1alpha1.OpAMPBridgeCapability{v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig, v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig, v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces, v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics, v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs, v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand, v1alpha1.OpAMPBridgeCapabilityReportsHealth, v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig},
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
			"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
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
	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))
}

func TestContinueOnRecoverableFailure_OpAMPBridge(t *testing.T) {
	// prepare
	taskCalled := false
	reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Log: opampBridgeLogger,
		Tasks: []controllers.OpAMPBridgeReconcilerTask{
			{
				Name: "should-fail",
				Do: func(context.Context, manifests.Params) error {
					return errors.New("should fail")
				},
				BailOnError: false,
			},
			{
				Name: "should-be-called",
				Do: func(context.Context, manifests.Params) error {
					taskCalled = true
					return nil
				},
			},
		},
	})

	// test
	err := reconciler.RunTasks(context.Background(), manifests.Params{})

	// verify
	assert.NoError(t, err)
	assert.True(t, taskCalled)
}

func TestBreakOnUnrecoverableError_OpAMPBridge(t *testing.T) {
	// prepare
	cfg := config.New()
	taskCalled := false
	expectedErr := errors.New("should fail")
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.OpAMPBridgeReconcilerTask{
			{
				Name: "should-fail",
				Do: func(context.Context, manifests.Params) error {
					taskCalled = true
					return expectedErr
				},
				BailOnError: true,
			},
			{
				Name: "should-not-be-called",
				Do: func(context.Context, manifests.Params) error {
					assert.Fail(t, "should not have been called")
					return nil
				},
			},
		},
	})
	created := &v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Endpoint:     "ws://opamp-server:4320/v1/opamp",
			Protocol:     "wss",
			Capabilities: []v1alpha1.OpAMPBridgeCapability{v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig, v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig, v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces, v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics, v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs, v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand, v1alpha1.OpAMPBridgeCapabilityReportsHealth, v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig},
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

func TestSkipWhenInstanceDoesNotExist_OpAMPBridge(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client: k8sClient,
		Log:    logger,
		Scheme: scheme.Scheme,
		Config: cfg,
		Tasks: []controllers.OpAMPBridgeReconcilerTask{
			{
				Name: "should-not-be-called",
				Do: func(context.Context, manifests.Params) error {
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

func TestRegisterWithManager_OpAMPBridge(t *testing.T) {
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
