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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
)

var logger = logf.Log.WithName("unit-tests")
var mockAutoDetector = &mockAutoDetect{
	OpenShiftRoutesAvailabilityFunc: func() (autodetect.OpenShiftRoutesAvailability, error) {
		return autodetect.OpenShiftRoutesAvailable, nil
	},
}

func TestContinueOnRecoverableFailure(t *testing.T) {
	// prepare
	taskCalled := false
	reconciler := controllers.NewReconciler(controllers.Params{
		Log: logger,
		Tasks: []controllers.Task{
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
	OpenShiftRoutesAvailabilityFunc func() (autodetect.OpenShiftRoutesAvailability, error)
}

func (m *mockAutoDetect) OpenShiftRoutesAvailability() (autodetect.OpenShiftRoutesAvailability, error) {
	if m.OpenShiftRoutesAvailabilityFunc != nil {
		return m.OpenShiftRoutesAvailabilityFunc()
	}
	return autodetect.OpenShiftRoutesNotAvailable, nil
}
