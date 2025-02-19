// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

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
	opampbridgeStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/opampbridge"
)

// OpAMPBridgeReconciler reconciles a OpAMPBridge object.
type OpAMPBridgeReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	log      logr.Logger
	recorder record.EventRecorder
	config   config.Config
}

// OpAMPBridgeReconcilerParams is the set of options to build a new OpAMPBridgeReconciler.
type OpAMPBridgeReconcilerParams struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
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
	// We have a deletion, short circuit and let the deletion happen
	if deletionTimestamp := instance.GetDeletionTimestamp(); deletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	params := r.getParams(instance)

	desiredObjects, buildErr := BuildOpAMPBridge(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}
	err := reconcileDesiredObjects(ctx, r.Client, log, &params.OpAMPBridge, params.Scheme, desiredObjects, nil)
	return opampbridgeStatus.HandleReconcileStatus(ctx, log, params, err)
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
