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

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestReconcile(t *testing.T) {
	logger := zap.New()
	ctx := context.Background()

	scheme := runtime.NewScheme()
	require.NoError(t, v1beta1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name           string
		existingState  []runtime.Object
		expectedResult ctrl.Result
		expectedError  bool
	}{
		{
			name:           "collector not found",
			existingState:  []runtime.Object{},
			expectedResult: ctrl.Result{},
			expectedError:  false,
		},
		{
			name: "unmanaged collector",
			existingState: []runtime.Object{
				&v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "default",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
							ManagementState: v1beta1.ManagementStateUnmanaged,
						},
					},
				},
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tt.existingState...).
				Build()

			r := &OpenTelemetryCollectorReconciler{
				Client:   client,
				log:      logger,
				scheme:   scheme,
				config:   config.New(),
				recorder: record.NewFakeRecorder(100),
			}

			result, err := r.Reconcile(ctx, ctrl.Request{})

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestNewReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(100)
	logger := zap.New()
	cfg := config.New()

	params := Params{
		Client:   client,
		Recorder: recorder,
		Scheme:   scheme,
		Log:      logger,
		Config:   cfg,
	}

	r := NewReconciler(params)

	assert.Equal(t, client, r.Client)
	assert.Equal(t, recorder, r.recorder)
	assert.Equal(t, scheme, r.scheme)
	assert.Equal(t, logger, r.log)
	assert.Equal(t, cfg, r.config)
}
