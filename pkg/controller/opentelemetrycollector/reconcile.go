package opentelemetrycollector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// blank assignment to verify that ReconcileOpenTelemetryCollector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOpenTelemetryCollector{}

// ReconcileOpenTelemetryCollector reconciles a OpenTelemetryCollector object
type ReconcileOpenTelemetryCollector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	// the list of reconciliation functions to execute
	reconcileFuncs []func(context.Context) error
}

// New constructs a ReconcileOpenTelemetryCollector based on the client and scheme, with the default reconciliation functions
func New(client client.Client, scheme *runtime.Scheme) *ReconcileOpenTelemetryCollector {
	r := &ReconcileOpenTelemetryCollector{
		client: client,
		scheme: scheme,
	}
	r.reconcileFuncs = []func(context.Context) error{
		r.reconcileConfigMap,
		r.reconcileService,
		r.reconcileDeployment,
	}

	return r
}

// Reconcile reads that state of the cluster for a OpenTelemetryCollector object and makes changes based on the state read
// and what is in the OpenTelemetryCollector.Spec
func (r *ReconcileOpenTelemetryCollector) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues(
		"Request.Namespace", request.Namespace,
		"Request.Name", request.Name,
		"Request.ID", time.Now().UTC().UnixNano(),
	)
	reqLogger.Info("Reconciling OpenTelemetryCollector")

	// Fetch the OpenTelemetryCollector instance
	instance := &v1alpha1.OpenTelemetryCollector{}
	err := r.client.Get(context.Background(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("OpenTelemetryCollector was deleted, reconciliation terminated")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// set the execution context for this reconcile loop
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, reqLogger)

	if err := r.applyUpgrades(ctx); err != nil {
		reqLogger.Error(err, "failed to upgrade the custom resource and its underlying resources, reconciliation aborted")
		return reconcile.Result{}, err
	}

	if err := r.handleReconcile(ctx); err != nil {
		reqLogger.Error(err, "failed to reconcile custom resource")
		return reconcile.Result{}, err
	}

	// update the status object, which might have also been updated
	if err := r.client.Status().Update(ctx, instance); err != nil {
		reqLogger.Error(err, "failed to store the custom resource's status")
		return reconcile.Result{}, err
	}

	// apply it back, as it might have been updated
	if err := r.client.Update(ctx, instance); err != nil {
		reqLogger.Error(err, "failed to store back the custom resource")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Finished reconciling OpenTelemetryCollector")
	return reconcile.Result{}, nil
}

// handleReconcile compares the existing state vs. the expected state and performs the necessary actions to make the two match
func (r *ReconcileOpenTelemetryCollector) handleReconcile(ctx context.Context) error {
	if nil == r.reconcileFuncs {
		// nothing to do!
		return nil
	}

	for _, f := range r.reconcileFuncs {
		if err := f(ctx); err != nil {
			return fmt.Errorf("reconciliation failed: %v", err)
		}
	}

	return nil
}

// setControllerReference should be used by the individual reconcile functions to establish the ownership of the underlying resources
func (r *ReconcileOpenTelemetryCollector) setControllerReference(ctx context.Context, object v1.Object) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	return controllerutil.SetControllerReference(instance, object, r.scheme)
}
