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
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
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
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/status"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// OpenTelemetryCollectorReconciler reconciles a OpenTelemetryCollector object.
type OpenTelemetryCollectorReconciler struct {
	client.Client
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config

	tasks   []Task
	muTasks sync.RWMutex
}

// Task represents a reconciliation task to be executed by the reconciler.
type Task struct {
	Do          func(context.Context, manifests.Params) error
	Name        string
	BailOnError bool
}

// Params is the set of options to build a new OpenTelemetryCollectorReconciler.
type Params struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Tasks    []Task
	Config   config.Config
}

func (r *OpenTelemetryCollectorReconciler) onOpenShiftRoutesChange() error {
	plt := r.config.OpenShiftRoutes()
	var (
		routesIdx = -1
	)
	r.muTasks.Lock()
	for i, t := range r.tasks {
		// search for route reconciler
		switch t.Name {
		case "routes":
			routesIdx = i
		}
	}
	r.muTasks.Unlock()

	if err := r.addRouteTask(plt, routesIdx); err != nil {
		return err
	}

	return r.removeRouteTask(plt, routesIdx)
}

func (r *OpenTelemetryCollectorReconciler) addRouteTask(ora autodetect.OpenShiftRoutesAvailability, routesIdx int) error {
	r.muTasks.Lock()
	defer r.muTasks.Unlock()
	// if exists and openshift routes are available
	if routesIdx == -1 && ora == autodetect.OpenShiftRoutesAvailable {
		r.tasks = append([]Task{{collectorreconcile.Routes, "routes", true}}, r.tasks...)
	}
	return nil
}

func (r *OpenTelemetryCollectorReconciler) removeRouteTask(ora autodetect.OpenShiftRoutesAvailability, routesIdx int) error {
	r.muTasks.Lock()
	defer r.muTasks.Unlock()
	if len(r.tasks) < routesIdx {
		return fmt.Errorf("can not remove route task from reconciler")
	}
	// if exists and openshift routes are not available
	if routesIdx != -1 && ora == autodetect.OpenShiftRoutesNotAvailable {
		r.tasks = append(r.tasks[:routesIdx], r.tasks[routesIdx+1:]...)
	}
	return nil
}

func (r *OpenTelemetryCollectorReconciler) doCRUD(ctx context.Context, params manifests.Params) error {
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
			if setErr := ctrl.SetControllerReference(&params.Instance, desired, params.Scheme); setErr != nil {
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
		return fmt.Errorf("failed to create objects for Collector %s: %w", params.Instance.GetName(), errors.Join(errs...))
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

func (r *OpenTelemetryCollectorReconciler) getParams(instance v1alpha1.OpenTelemetryCollector) manifests.Params {
	return manifests.Params{
		Config:   r.config,
		Client:   r.Client,
		Instance: instance,
		Log:      r.log,
		Scheme:   r.scheme,
		Recorder: r.recorder,
	}
}

// NewReconciler creates a new reconciler for OpenTelemetryCollector objects.
func NewReconciler(p Params) *OpenTelemetryCollectorReconciler {
	r := &OpenTelemetryCollectorReconciler{
		Client:   p.Client,
		log:      p.Log,
		scheme:   p.Scheme,
		config:   p.Config,
		tasks:    p.Tasks,
		recorder: p.Recorder,
	}

	if len(r.tasks) == 0 {
		// TODO: put this in line with the rest of how we generate manifests
		// https://github.com/open-telemetry/opentelemetry-operator/issues/2108
		r.config.RegisterOpenShiftRoutesChangeCallback(r.onOpenShiftRoutesChange)
	}
	return r
}

// +kubebuilder:rbac:groups="",resources=configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/finalizers,verbs=get;update;patch

// Reconcile the current state of an OpenTelemetry collector resource with the desired state.
func (r *OpenTelemetryCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("opentelemetrycollector", req.NamespacedName)

	var instance v1alpha1.OpenTelemetryCollector
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpenTelemetryCollector")
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.Spec.ManagementState != v1alpha1.ManagementStateManaged {
		log.Info("Skipping reconciliation for unmanaged OpenTelemetryCollector resource", "name", req.String())
		// Stop requeueing for unmanaged OpenTelemetryCollector custom resources
		return ctrl.Result{}, nil
	}

	params := r.getParams(instance)
	if err := r.RunTasks(ctx, params); err != nil {
		return ctrl.Result{}, err
	}

	err := r.doCRUD(ctx, params)
	return status.HandleReconcileStatus(ctx, log, params, err)
}

// RunTasks runs all the tasks associated with this reconciler.
func (r *OpenTelemetryCollectorReconciler) RunTasks(ctx context.Context, params manifests.Params) error {
	r.muTasks.RLock()
	defer r.muTasks.RUnlock()
	for _, task := range r.tasks {
		if err := task.Do(ctx, params); err != nil {
			// If we get an error that occurs because a pod is being terminated, then exit this loop
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

// BuildAll returns the generation and collected errors of all manifests for a given instance.
func (r *OpenTelemetryCollectorReconciler) BuildAll(params manifests.Params) ([]client.Object, error) {
	builders := []manifests.Builder{
		collector.Build,
		targetallocator.Build,
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

// SetupWithManager tells the manager what our controller is interested in.
func (r *OpenTelemetryCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := r.config.AutoDetect() // We need to call this, so we can get the correct autodetect version
	if err != nil {
		return err
	}
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenTelemetryCollector{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.StatefulSet{})

	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		builder.Owns(&monitoringv1.ServiceMonitor{})
	}

	autoscalingVersion := r.config.AutoscalingVersion()
	if autoscalingVersion == autodetect.AutoscalingVersionV2 {
		builder = builder.Owns(&autoscalingv2.HorizontalPodAutoscaler{})
	} else {
		builder = builder.Owns(&autoscalingv2beta2.HorizontalPodAutoscaler{})
	}

	return builder.Complete(r)
}
