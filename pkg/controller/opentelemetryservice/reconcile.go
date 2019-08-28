package opentelemetryservice

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// blank assignment to verify that ReconcileOpenTelemetryService implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileOpenTelemetryService{}

// ReconcileOpenTelemetryService reconciles a OpenTelemetryService object
type ReconcileOpenTelemetryService struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a OpenTelemetryService object and makes changes based on the state read
// and what is in the OpenTelemetryService.Spec
func (r *ReconcileOpenTelemetryService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OpenTelemetryService")

	// Fetch the OpenTelemetryService instance
	instance := &v1alpha1.OpenTelemetryService{}
	err := r.client.Get(context.Background(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// set the execution context for this reconcile loop
	ctx := context.WithValue(context.Background(), opentelemetry.Instance, instance)

	if err := r.handleReconcile(ctx); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// handleReconcile compares the existing state vs. the expected state and performs the necessary actions to make the two match
func (r *ReconcileOpenTelemetryService) handleReconcile(ctx context.Context) error {
	funcs := []func(context.Context) error{
		r.reconcileConfigMap,
		r.reconcileService,
		r.reconcileDeployment,
	}

	for _, f := range funcs {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

// setControllerReference should be used by the individual reconcile functions to establish the ownership of the underlying resources
func (r *ReconcileOpenTelemetryService) setControllerReference(ctx context.Context, object v1.Object) error {
	instance := ctx.Value(opentelemetry.Instance).(*v1alpha1.OpenTelemetryService)
	return controllerutil.SetControllerReference(instance, object, r.scheme)
}
