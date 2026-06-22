// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var testScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))
}

func TestHandleReconcileStatus(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "my-inst",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: v1alpha1.InstrumentationSpec{
			Exporter: v1alpha1.Exporter{
				Endpoint: "http://localhost:4317",
			},
		},
	}

	cli := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(&inst).
		WithStatusSubresource(&inst).
		Build()

	log := logf.Log.WithName("test")
	result, err := HandleReconcileStatus(context.Background(), log, cli, inst)
	require.NoError(t, err)
	assert.False(t, result.Requeue)

	var updated v1alpha1.Instrumentation
	require.NoError(t, cli.Get(context.Background(), types.NamespacedName{Name: "my-inst", Namespace: "default"}, &updated))

	assert.Equal(t, int64(1), updated.Status.ObservedGeneration)
	readyCondition := meta.FindStatusCondition(updated.Status.Conditions, "Ready")
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionTrue, readyCondition.Status)
	assert.Equal(t, int64(1), readyCondition.ObservedGeneration)
	assert.Equal(t, "Reconciled", readyCondition.Reason)
}

func TestHandleReconcileStatusGenerationUpdate(t *testing.T) {
	inst := v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "my-inst",
			Namespace:  "default",
			Generation: 2,
		},
		Status: v1alpha1.InstrumentationStatus{
			ObservedGeneration: 1,
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             "Reconciled",
					Message:            "Successfully reconciled",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	cli := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(&inst).
		WithStatusSubresource(&inst).
		Build()

	log := logf.Log.WithName("test")
	result, err := HandleReconcileStatus(context.Background(), log, cli, inst)
	require.NoError(t, err)
	assert.False(t, result.Requeue)

	var updated v1alpha1.Instrumentation
	require.NoError(t, cli.Get(context.Background(), types.NamespacedName{Name: "my-inst", Namespace: "default"}, &updated))

	assert.Equal(t, int64(2), updated.Status.ObservedGeneration)
	readyCondition := meta.FindStatusCondition(updated.Status.Conditions, "Ready")
	require.NotNil(t, readyCondition)
	assert.Equal(t, metav1.ConditionTrue, readyCondition.Status)
	assert.Equal(t, int64(2), readyCondition.ObservedGeneration)
}
