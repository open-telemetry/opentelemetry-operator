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
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	opampbridgereconcile "github.com/open-telemetry/opentelemetry-operator/pkg/reconcile/opampbridge"
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
	Do          func(context.Context, manifests.OpAMPBridgeParams) error
	Name        string
	BailOnError bool
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

	params := opampbridgereconcile.Params{
		Client:   r.Client,
		Recorder: r.recorder,
		Scheme:   r.scheme,
		Log:      r.log,
		Instance: instance,
		Config:   r.config,
	}
	if err := r.RunTasks(ctx, params); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *OpAMPBridgeReconciler) RunTasks(ctx context.Context, params opampbridgereconcile.Params) error {
	for _, task := range r.tasks {
		if err := task.Do(ctx, params); err != nil {
			if apierrors.IsForbidden(err) && apierrors.HasStatusCause(err, corev1.NamespaceTerminatingCause) {
				r.log.V(2).Info("Exiting reconcile loop because namespace is being terminated", "namespace", params.Instance.Namespace)
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
