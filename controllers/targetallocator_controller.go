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

// Package controllers contains the main controller, where the reconciliation starts.
package controllers

import (
	"context"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyV1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	taStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// TargetAllocatorReconciler reconciles a TargetAllocator object.
type TargetAllocatorReconciler struct {
	client.Client
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config
}

// TargetAllocatorReconcilerParams is the set of options to build a new TargetAllocatorReconciler.
type TargetAllocatorReconcilerParams struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
}

func (r *TargetAllocatorReconciler) getParams(instance v1alpha1.TargetAllocator) targetallocator.Params {
	p := targetallocator.Params{
		Config:          r.config,
		Client:          r.Client,
		Log:             r.log,
		Scheme:          r.scheme,
		Recorder:        r.recorder,
		TargetAllocator: instance,
	}

	return p
}

// NewTargetAllocatorReconciler creates a new reconciler for TargetAllocator objects.
func NewTargetAllocatorReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config config.Config,
	logger logr.Logger,
) *TargetAllocatorReconciler {
	return &TargetAllocatorReconciler{
		Client:   client,
		log:      logger,
		scheme:   scheme,
		config:   config,
		recorder: recorder,
	}
}

// TODO: Uncomment the lines below after enabling the TA controller in main.go
// // +kubebuilder:rbac:groups="",resources=pods;configmaps;services;serviceaccounts;persistentvolumeclaims;persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// // +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// // +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// // +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// // +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete
// // +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;update;patch
// // +kubebuilder:rbac:groups=opentelemetry.io,resources=targetallocators,verbs=get;list;watch;update;patch
// // +kubebuilder:rbac:groups=opentelemetry.io,resources=targetallocators/status,verbs=get;update;patch

// Reconcile the current state of a TargetAllocator resource with the desired state.
func (r *TargetAllocatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("targetallocator", req.NamespacedName)

	var instance v1alpha1.TargetAllocator
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch TargetAllocator")
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

	if instance.Spec.ManagementState == v1beta1.ManagementStateUnmanaged {
		log.Info("Skipping reconciliation for unmanaged TargetAllocator resource", "name", req.String())
		// Stop requeueing for unmanaged TargetAllocator custom resources
		return ctrl.Result{}, nil
	}

	params := r.getParams(instance)
	desiredObjects, buildErr := BuildTargetAllocator(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}

	err := reconcileDesiredObjects(ctx, r.Client, log, &params.TargetAllocator, params.Scheme, desiredObjects, nil)
	return taStatus.HandleReconcileStatus(ctx, log, params, err)
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *TargetAllocatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TargetAllocator{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolume{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&policyV1.PodDisruptionBudget{})

	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		builder.Owns(&monitoringv1.ServiceMonitor{})
		builder.Owns(&monitoringv1.PodMonitor{})
	}

	return builder.Complete(r)
}
