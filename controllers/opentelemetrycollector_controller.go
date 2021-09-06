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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/signalf/splunk-otel-operator/api/v1alpha1"
	"github.com/signalf/splunk-otel-operator/internal/config"
	"github.com/signalf/splunk-otel-operator/pkg/collector/reconcile"
)

// SplunkOtelAgentReconciler reconciles a SplunkOtelAgent object.
type SplunkOtelAgentReconciler struct {
	client.Client
	log      logr.Logger
	scheme   *runtime.Scheme
	config   config.Config
	tasks    []Task
	recorder record.EventRecorder
}

// Task represents a reconciliation task to be executed by the reconciler.
type Task struct {
	Name        string
	Do          func(context.Context, reconcile.Params) error
	BailOnError bool
}

// Params is the set of options to build a new openTelemetryCollectorReconciler.
type Params struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Config   config.Config
	Tasks    []Task
	Recorder record.EventRecorder
}

// NewReconciler creates a new reconciler for SplunkOtelAgent objects.
func NewReconciler(p Params) *SplunkOtelAgentReconciler {
	if len(p.Tasks) == 0 {
		p.Tasks = []Task{
			/*
				{
					"namespaces",
					reconcile.Namespaces,
					true,
				},
			*/
			{
				"config maps",
				reconcile.ConfigMaps,
				true,
			},
			{
				"service accounts",
				reconcile.ServiceAccounts,
				true,
			},
			{
				"services",
				reconcile.Services,
				true,
			},
			{
				"deployments",
				reconcile.Deployments,
				true,
			},
			{
				"daemon sets",
				reconcile.DaemonSets,
				true,
			},
			{
				"stateful sets",
				reconcile.StatefulSets,
				true,
			},
			{
				"opentelemetry",
				reconcile.Self,
				true,
			},
		}
	}

	return &SplunkOtelAgentReconciler{
		Client:   p.Client,
		log:      p.Log,
		scheme:   p.Scheme,
		config:   p.Config,
		tasks:    p.Tasks,
		recorder: p.Recorder,
	}
}

// +kubebuilder:rbac:groups=splunk.com,resources=splunkotelagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=splunk.com,resources=splunkotelagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=splunk.com,resources=splunkotelagents/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile the current state of an OpenTelemetry collector resource with the desired state.
func (r *SplunkOtelAgentReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.log.WithValues("splunkotelagent", req.NamespacedName)

	var instance v1alpha1.SplunkOtelAgent
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch SplunkOtelAgent")
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	setAgentDefaults(&instance)

	params := reconcile.Params{
		Config:   r.config,
		Client:   r.Client,
		Instance: instance,
		Log:      log,
		Scheme:   r.scheme,
		Recorder: r.recorder,
	}

	if err := r.RunTasks(ctx, params); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// RunTasks runs all the tasks associated with this reconciler.
func (r *SplunkOtelAgentReconciler) RunTasks(ctx context.Context, params reconcile.Params) error {
	for _, task := range r.tasks {
		if err := task.Do(ctx, params); err != nil {
			r.log.Error(err, fmt.Sprintf("failed to reconcile %s", task.Name))
			if task.BailOnError {
				return err
			}
		}
	}

	return nil
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *SplunkOtelAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SplunkOtelAgent{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
