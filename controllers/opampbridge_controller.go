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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/opampbridge"
	opampbridgeStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/opampbridge"
)

// OpAMPBridgeReconciler reconciles a OpAMPBridge object.
type OpAMPBridgeReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	log      logr.Logger
	recorder record.EventRecorder
	tasks    []OpAMPBridgeReconcilerTask
	config   config.Config
}

// OpAMPBridgeReconcilerParams is the set of options to build a new OpAMPBridgeReconciler.
type OpAMPBridgeReconcilerParams struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Tasks    []OpAMPBridgeReconcilerTask
	Config   config.Config
}

// OpAMPBridgeReconcilerTask represents a reconciliation task to be executed by the OpAMPBridgeReconciler.
type OpAMPBridgeReconcilerTask struct {
	Do          func(context.Context, manifests.Params) error
	Name        string
	BailOnError bool
}

func (r *OpAMPBridgeReconciler) doCRUD(ctx context.Context, params manifests.Params) error {
	// Collect all objects owned by the operator, to be able to prune objects
	// which exist in the cluster but are not managed by the operator anymore.
	desiredObjects, err := r.BuildAll(params)
	if err != nil {
		return err
	}
	var errs []error
	for _, desired := range desiredObjects {
		l := r.log.WithValues(
			"object_name", desired.GetName(),
			"object_kind", desired.GetObjectKind(),
		)
		if isNamespaceScoped(desired) {
			if setErr := ctrl.SetControllerReference(&params.OpAMPBridge, desired, params.Scheme); setErr != nil {
				l.Error(setErr, "failed to set controller owner reference to desired")
				errs = append(errs, setErr)
				continue
			}
		}

		// existing is an object the controller runtime will hydrate for us
		// we obtain the existing object by deep copying the desired object because it's the most convenient way
		existing := desired.DeepCopyObject().(client.Object)
		mutateFn := manifests.MutateFuncFor(existing, desired)
		op, crudErr := ctrl.CreateOrUpdate(ctx, r.Client, existing, mutateFn)
		if crudErr != nil && errors.Is(crudErr, manifests.ImmutableChangeErr) {
			l.Error(crudErr, "detected immutable field change, trying to delete, new object will be created on next reconcile", "existing", existing.GetName())
			delErr := r.Client.Delete(ctx, existing)
			if delErr != nil {
				return delErr
			}
			continue
		} else if crudErr != nil {
			l.Error(crudErr, "failed to configure desired")
			errs = append(errs, crudErr)
			continue
		}

		l.V(1).Info(fmt.Sprintf("desired has been %s", op))
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to create objects for OpAMPBridge %s: %w", params.OpAMPBridge.GetName(), errors.Join(errs...))
	}
	return nil
}

func isNamespaceScoped(obj client.Object) bool {
	switch obj.(type) {
	case *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
		return false
	default:
		return true
	}
}

func (r *OpAMPBridgeReconciler) getParams(instance v1alpha1.OpAMPBridge) manifests.Params {
	return manifests.Params{
		Config:      r.config,
		Client:      r.Client,
		OpAMPBridge: instance,
		Log:         r.log,
		Scheme:      r.scheme,
		Recorder:    r.recorder,
	}
}

func NewOpAMPBridgeReconciler(params OpAMPBridgeReconcilerParams) *OpAMPBridgeReconciler {
	reconciler := &OpAMPBridgeReconciler{
		Client:   params.Client,
		scheme:   params.Scheme,
		log:      params.Log,
		recorder: params.Recorder,
		tasks:    params.Tasks,
		config:   params.Config,
	}
	return reconciler
}

//+kubebuilder:rbac:groups=opentelemetry.io,resources=opampbridges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opentelemetry.io,resources=opampbridges/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=opentelemetry.io,resources=opampbridges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *OpAMPBridgeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("opamp-bridge", req.NamespacedName)
	var instance v1alpha1.OpAMPBridge
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpAMPBridge")
		}
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	params := r.getParams(instance)
	if err := r.RunTasks(ctx, params); err != nil {
		return ctrl.Result{}, err
	}
	err := r.doCRUD(ctx, params)
	return opampbridgeStatus.HandleReconcileStatus(ctx, log, params, err)
}

func (r *OpAMPBridgeReconciler) RunTasks(ctx context.Context, params manifests.Params) error {
	for _, task := range r.tasks {
		if err := task.Do(ctx, params); err != nil {
			if apierrors.IsForbidden(err) && apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
				r.log.V(2).Info("Exiting reconcile loop because namespace is being terminated", "namespace", params.OpAMPBridge.Namespace)
				return nil
			}
			r.log.Error(err, fmt.Sprintf("failed to reconcile %s", task.Name))
			if task.BailOnError {
				return err
			}
		}
	}
	return nil
}

// BuildAll returns the generation and collected errors of all manifests for a given instance.
func (r *OpAMPBridgeReconciler) BuildAll(params manifests.Params) ([]client.Object, error) {
	builders := []manifests.Builder{
		opampbridge.Build,
	}
	var resources []client.Object
	for _, builder := range builders {
		objs, err := builder(params)
		if err != nil {
			return nil, err
		}
		resources = append(resources, objs...)
	}
	return resources, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpAMPBridgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpAMPBridge{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
