// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package controllers contains the main controller, where the reconciliation starts.
package controllers

import (
	"context"
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyV1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	taStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
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

func (r *TargetAllocatorReconciler) getParams(ctx context.Context, instance v1alpha1.TargetAllocator) (targetallocator.Params, error) {
	collector, err := r.getCollector(ctx, instance)
	if err != nil {
		return targetallocator.Params{}, err
	}
	p := targetallocator.Params{
		Config:          r.config,
		Client:          r.Client,
		Log:             r.log,
		Scheme:          r.scheme,
		Recorder:        r.recorder,
		TargetAllocator: instance,
		Collector:       collector,
	}

	return p, nil
}

// getCollector finds the OpenTelemetryCollector for the given TargetAllocator. We have the following possibilities:
//   - Collector is the owner of the TargetAllocator
//   - Collector is labeled with the TargetAllocator's name
//   - No collector
func (r *TargetAllocatorReconciler) getCollector(ctx context.Context, instance v1alpha1.TargetAllocator) (*v1beta1.OpenTelemetryCollector, error) {
	var collector v1beta1.OpenTelemetryCollector

	// check if a collector is the owner of this Target Allocator
	ownerReferences := instance.GetOwnerReferences()
	collectorIndex := slices.IndexFunc(ownerReferences, func(reference metav1.OwnerReference) bool {
		return reference.Kind == "OpenTelemetryCollector"
	})
	if collectorIndex != -1 {
		collectorRef := ownerReferences[collectorIndex]
		collectorKey := client.ObjectKey{Name: collectorRef.Name, Namespace: instance.GetNamespace()}
		if err := r.Get(ctx, collectorKey, &collector); err != nil {
			return nil, fmt.Errorf(
				"error getting owner for TargetAllocator %s/%s: %w",
				instance.GetNamespace(), instance.GetName(), err)
		}
		return &collector, nil
	}

	// check if there are Collectors labeled with this Target Allocator's name
	var collectors v1beta1.OpenTelemetryCollectorList
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels{
			constants.LabelTargetAllocator: instance.GetName(),
		},
	}
	err := r.List(ctx, &collectors, listOpts...)
	if err != nil {
		return nil, err
	}
	if len(collectors.Items) == 0 {
		return nil, nil
	} else if len(collectors.Items) > 1 {
		return nil, fmt.Errorf("found multiple OpenTelemetry collectors annotated with the same Target Allocator: %s/%s", instance.GetNamespace(), instance.GetName())
	}

	return &collectors.Items[0], nil
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

// +kubebuilder:rbac:groups="",resources=pods;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=targetallocators,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=targetallocators/status,verbs=get;update;patch

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

	params, err := r.getParams(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	desiredObjects, buildErr := BuildTargetAllocator(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}

	err = reconcileDesiredObjects(ctx, r.Client, log, &params.TargetAllocator, params.Scheme, desiredObjects, nil)
	return taStatus.HandleReconcileStatus(ctx, log, params, err)
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *TargetAllocatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.TargetAllocator{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&policyV1.PodDisruptionBudget{})

	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		ctrlBuilder.Owns(&monitoringv1.ServiceMonitor{})
		ctrlBuilder.Owns(&monitoringv1.PodMonitor{})
	}

	// watch collectors which have embedded Target Allocator enabled
	// we need to do this separately from collector reconciliation, as changes to Config will not lead to changes
	// in the TargetAllocator CR
	ctrlBuilder.Watches(
		&v1beta1.OpenTelemetryCollector{},
		handler.EnqueueRequestsFromMapFunc(getTargetAllocatorForCollector),
		builder.WithPredicates(
			predicate.NewPredicateFuncs(func(object client.Object) bool {
				collector := object.(*v1beta1.OpenTelemetryCollector)
				return collector.Spec.TargetAllocator.Enabled
			}),
		),
	)

	// watch collectors which have the target allocator label
	collectorSelector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      constants.LabelTargetAllocator,
				Operator: metav1.LabelSelectorOpExists,
			},
		},
	}
	selectorPredicate, err := predicate.LabelSelectorPredicate(collectorSelector)
	if err != nil {
		return err
	}
	ctrlBuilder.Watches(
		&v1beta1.OpenTelemetryCollector{},
		handler.EnqueueRequestsFromMapFunc(getTargetAllocatorRequestsFromLabel),
		builder.WithPredicates(selectorPredicate),
	)

	return ctrlBuilder.Complete(r)
}

func getTargetAllocatorForCollector(_ context.Context, collector client.Object) []reconcile.Request {
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      collector.GetName(),
				Namespace: collector.GetNamespace(),
			},
		},
	}
}

func getTargetAllocatorRequestsFromLabel(_ context.Context, collector client.Object) []reconcile.Request {
	if taName, ok := collector.GetLabels()[constants.LabelTargetAllocator]; ok {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      taName,
					Namespace: collector.GetNamespace(),
				},
			},
		}
	}
	return []reconcile.Request{}
}
