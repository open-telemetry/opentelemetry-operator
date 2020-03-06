package opentelemetrycollector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/client"
)

// blank assignment to verify that ReconcileOpenTelemetryCollector implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOpenTelemetryCollector{}

// ReconcileOpenTelemetryCollector reconciles a OpenTelemetryCollector object
type ReconcileOpenTelemetryCollector struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	scheme *runtime.Scheme

	// the clients that compose this reconciler
	clientset *client.Clientset

	// the list of reconciliation functions to execute
	reconcileFuncs []func(context.Context) error
}

// WithManager creates a new reconciler based on the manager information
func WithManager(manager manager.Manager) (*ReconcileOpenTelemetryCollector, error) {
	cl, err := client.ForManager(manager)
	if err != nil {
		return nil, err
	}

	return New(manager.GetScheme(), cl), nil
}

// New constructs a ReconcileOpenTelemetryCollector based on the client and scheme, with the default reconciliation functions
func New(scheme *runtime.Scheme, clientset *client.Clientset) *ReconcileOpenTelemetryCollector {
	r := &ReconcileOpenTelemetryCollector{
		scheme:    scheme,
		clientset: clientset,
	}
	r.reconcileFuncs = []func(context.Context) error{
		r.reconcileServiceAccount,
		r.reconcileConfigMap,
		r.reconcileService,
		r.reconcileDeployment,
		r.reconcileDaemonSet,
		r.reconcileServiceMonitor,
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

	otelCols := r.clientset.OpenTelemetry.OpentelemetryV1alpha1().OpenTelemetryCollectors(request.Namespace)

	// Fetch the OpenTelemetryCollector instance
	instance, err := otelCols.Get(request.Name, v1.GetOptions{})
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
	if instance, err = otelCols.UpdateStatus(instance); err != nil {
		reqLogger.Error(err, "failed to store the custom resource's status")
		return reconcile.Result{}, err
	}

	// apply it back, as it might have been updated
	if _, err := otelCols.Update(instance); err != nil {
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

func resourceName(instanceName string) string {
	return fmt.Sprintf("%s-collector", instanceName)
}
