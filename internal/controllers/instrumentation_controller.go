// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	instrumentationStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/instrumentation"
)

// InstrumentationReconciler reconciles an Instrumentation object.
type InstrumentationReconciler struct {
	client.Client
	log logr.Logger
}

// NewInstrumentationReconciler creates a new InstrumentationReconciler.
func NewInstrumentationReconciler(c client.Client, log logr.Logger) *InstrumentationReconciler {
	return &InstrumentationReconciler{
		Client: c,
		log:    log,
	}
}

// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations/status,verbs=get;update;patch

func (r *InstrumentationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("instrumentation", req.NamespacedName)

	var instance v1alpha1.Instrumentation
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch Instrumentation")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	return instrumentationStatus.HandleReconcileStatus(ctx, log, r.Client, instance)
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumentationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Instrumentation{}).
		Complete(r)
}
