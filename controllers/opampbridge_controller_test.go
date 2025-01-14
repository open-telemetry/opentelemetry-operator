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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var opampBridgeLogger = logf.Log.WithName("opamp-bridge-controller-unit-tests")
var opampBridgeMockAutoDetector = &mockAutoDetect{
	OpenShiftRoutesAvailabilityFunc: func() (openshift.RoutesAvailability, error) {
		return openshift.RoutesAvailable, nil
	},
	PrometheusCRsAvailabilityFunc: func() (prometheus.Availability, error) {
		return prometheus.Available, nil
	},
	RBACPermissionsFunc: func(ctx context.Context) (rbac.Availability, error) {
		return rbac.Available, nil
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
		Log:      opampBridgeLogger,
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
			Endpoint: "ws://opamp-server:4320/v1/opamp",
			Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
				v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
				v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
				v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
				v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
				v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
				v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
				v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
				v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
				v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
				v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
				v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
			},
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

func TestSkipWhenInstanceDoesNotExist_OpAMPBridge(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "non-existing-my-instance", Namespace: "default"}
	reconciler := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client: k8sClient,
		Log:    opampBridgeLogger,
		Scheme: scheme.Scheme,
		Config: cfg,
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
