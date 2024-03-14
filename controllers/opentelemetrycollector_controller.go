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
	"fmt"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	collectorStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// OpenTelemetryCollectorReconciler reconciles a OpenTelemetryCollector object.
type OpenTelemetryCollectorReconciler struct {
	client.Client
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config
}

// Params is the set of options to build a new OpenTelemetryCollectorReconciler.
type Params struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
}

func (r *OpenTelemetryCollectorReconciler) findOtelOwnedObjects(ctx context.Context, params manifests.Params) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}

	listOps := &client.ListOptions{
		Namespace:     params.OtelCol.Namespace,
		LabelSelector: labels.SelectorFromSet(manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, collector.ComponentOpenTelemetryCollector)),
	}
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	err := r.List(ctx, hpaList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing HorizontalPodAutoscalers: %w", err)
	}
	for i := range hpaList.Items {
		ownedObjects[hpaList.Items[i].GetUID()] = &hpaList.Items[i]
	}
	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		servicemonitorList := &monitoringv1.ServiceMonitorList{}
		err = r.List(ctx, servicemonitorList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing ServiceMonitors: %w", err)
		}
		for i := range servicemonitorList.Items {
			ownedObjects[servicemonitorList.Items[i].GetUID()] = servicemonitorList.Items[i]
		}

		podMonitorList := &monitoringv1.PodMonitorList{}
		err = r.List(ctx, podMonitorList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing PodMonitors: %w", err)
		}
		for i := range podMonitorList.Items {
			ownedObjects[podMonitorList.Items[i].GetUID()] = podMonitorList.Items[i]
		}
	}
	ingressList := &networkingv1.IngressList{}
	err = r.List(ctx, ingressList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing Ingresses: %w", err)
	}
	for i := range ingressList.Items {
		ownedObjects[ingressList.Items[i].GetUID()] = &ingressList.Items[i]
	}

	if params.Config.OpenShiftRoutesAvailability() == openshift.RoutesAvailable {
		routesList := &routev1.RouteList{}
		err = r.List(ctx, routesList, listOps)
		if err != nil {
			return nil, fmt.Errorf("error listing Routes: %w", err)
		}
		for i := range routesList.Items {
			ownedObjects[routesList.Items[i].GetUID()] = &routesList.Items[i]
		}
	}
	pdbList := &policyV1.PodDisruptionBudgetList{}
	err = r.List(ctx, pdbList, listOps)
	if err != nil {
		return nil, fmt.Errorf("error listing PodDisruptionBudgets: %w", err)
	}
	for i := range pdbList.Items {
		ownedObjects[pdbList.Items[i].GetUID()] = &pdbList.Items[i]
	}

	return ownedObjects, nil
}

func (r *OpenTelemetryCollectorReconciler) getParams(instance v1beta1.OpenTelemetryCollector) (manifests.Params, error) {
	p := manifests.Params{
		Config:   r.config,
		Client:   r.Client,
		OtelCol:  instance,
		Log:      r.log,
		Scheme:   r.scheme,
		Recorder: r.recorder,
	}

	// generate the target allocator CR from the collector CR
	targetAllocator, err := collector.TargetAllocator(p)
	if err != nil {
		return p, err
	}
	if targetAllocator != nil {
		p.TargetAllocator = *targetAllocator
	}
	return p, nil
}

// NewReconciler creates a new reconciler for OpenTelemetryCollector objects.
func NewReconciler(p Params) *OpenTelemetryCollectorReconciler {
	r := &OpenTelemetryCollectorReconciler{
		Client:   p.Client,
		log:      p.Log,
		scheme:   p.Scheme,
		config:   p.Config,
		recorder: p.Recorder,
	}
	return r
}

// +kubebuilder:rbac:groups="",resources=pods;configmaps;services;serviceaccounts;persistentvolumeclaims;persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=infrastructures;infrastructures/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/finalizers,verbs=get;update;patch

// Reconcile the current state of an OpenTelemetry collector resource with the desired state.
func (r *OpenTelemetryCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("opentelemetrycollector", req.NamespacedName)

	var instance v1beta1.OpenTelemetryCollector
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpenTelemetryCollector")
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
		log.Info("Skipping reconciliation for unmanaged OpenTelemetryCollector resource", "name", req.String())
		// Stop requeueing for unmanaged OpenTelemetryCollector custom resources
		return ctrl.Result{}, nil
	}

	params, err := r.getParams(instance)
	if err != nil {
		log.Error(err, "Failed to create manifest.Params")
		return ctrl.Result{}, err
	}

	desiredObjects, buildErr := BuildCollector(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}

	ownedObjects, err := r.findOtelOwnedObjects(ctx, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconcileDesiredObjects(ctx, r.Client, log, &instance, params.Scheme, desiredObjects, ownedObjects)
	return collectorStatus.HandleReconcileStatus(ctx, log, params, instance, err)
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *OpenTelemetryCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.OpenTelemetryCollector{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.PersistentVolume{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&policyV1.PodDisruptionBudget{})

	if r.config.CreateRBACPermissions() {
		builder.Owns(&rbacv1.ClusterRoleBinding{})
		builder.Owns(&rbacv1.ClusterRole{})
	}

	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		builder.Owns(&monitoringv1.ServiceMonitor{})
		builder.Owns(&monitoringv1.PodMonitor{})
	}
	if r.config.OpenShiftRoutesAvailability() == openshift.RoutesAvailable {
		builder.Owns(&routev1.Route{})
	}

	return builder.Complete(r)
}
