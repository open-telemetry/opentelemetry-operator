package opentelemetrycollector

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

func TestProperReconcile(t *testing.T) {
	// prepare
	var (
		reconciled *v1alpha1.OpenTelemetryCollector
		logger     logr.Logger
		req        reconcile.Request
	)

	called := false
	reconciler.reconcileFuncs = []func(context.Context) error{
		func(ctx context.Context) error {
			called = true
			reconciled = ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
			logger = ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
			return nil
		},
	}

	// test
	res, err := reconciler.Reconcile(req)

	// verify
	assert.NoError(t, err)
	assert.False(t, res.Requeue)
	assert.True(t, called)
	assert.Equal(t, instance.Name, reconciled.Name)
	assert.NotNil(t, logger)
}

func TestReconcileDeletedObject(t *testing.T) {
	// prepare
	cl := fake.NewFakeClient() // no objects
	reconciler := New(cl, schem)

	req := reconcile.Request{}
	reconciler.reconcileFuncs = []func(context.Context) error{
		func(context.Context) error {
			assert.Fail(t, "shouldn't have been called")
			return nil
		},
	}

	// test
	res, err := reconciler.Reconcile(req)

	// verify
	assert.False(t, res.Requeue)
	assert.NoError(t, err)

}

func TestReconcileFailsFast(t *testing.T) {
	// prepare
	req := reconcile.Request{}
	reconciler.reconcileFuncs = []func(context.Context) error{
		func(context.Context) error {
			return errors.New("the server made a boo boo")
		},
		func(context.Context) error {
			assert.Fail(t, "shouldn't have been called")
			return nil
		},
	}

	// test
	_, err := reconciler.Reconcile(req)

	// verify
	assert.Error(t, err)
}

func TestReconcileFuncsAreCalled(t *testing.T) {
	// prepare
	called := false
	reconciler.reconcileFuncs = []func(context.Context) error{
		func(context.Context) error {
			called = true
			return nil
		},
	}

	// test
	err := reconciler.handleReconcile(ctx)

	// verify
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestNilReconcileFuncs(t *testing.T) {
	// prepare
	reconciler.reconcileFuncs = nil

	// test
	err := reconciler.handleReconcile(ctx)

	// verify
	assert.NoError(t, err)
}

func TestSetControllerReference(t *testing.T) {
	// prepare
	d := &appsv1.Deployment{}

	// sanity check
	assert.Len(t, d.OwnerReferences, 0)

	// test
	reconciler.setControllerReference(ctx, d)

	// verify
	assert.Len(t, d.OwnerReferences, 1)
	assert.Equal(t, instance.Name, d.OwnerReferences[0].Name)
	assert.Equal(t, instance.TypeMeta.APIVersion, d.OwnerReferences[0].APIVersion)
	assert.Equal(t, instance.TypeMeta.Kind, d.OwnerReferences[0].Kind)

}
