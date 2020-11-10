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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

// OpenTelemetryCollectorDistributionReconciler reconciles a OpenTelemetryCollectorDistribution object
type OpenTelemetryCollectorDistributionReconciler struct {
	client.Client
	log      logr.Logger
	scheme   *runtime.Scheme
	cfg      *config.Config
	onChange []func([]v1alpha1.OpenTelemetryCollectorDistribution)

	// there's no need to protect this in a lock, as we'll get only one worker per reconciler
	distributions []v1alpha1.OpenTelemetryCollectorDistribution
}

func NewDistributionReconciler(p Params) *OpenTelemetryCollectorDistributionReconciler {
	return &OpenTelemetryCollectorDistributionReconciler{
		Client: p.Client,
		log:    p.Log,
		scheme: p.Scheme,
		cfg:    p.Config,
		onChange: []func([]v1alpha1.OpenTelemetryCollectorDistribution){
			p.Config.SetDistributions,
		},
	}
}

// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectordistributions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectordistributions/status,verbs=get;update;patch

func (r *OpenTelemetryCollectorDistributionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.log.WithValues("opentelemetrycollectordistribution", req.NamespacedName)

	var instance v1alpha1.OpenTelemetryCollectorDistribution
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpenTelemetryCollectorDistribution")
			return ctrl.Result{}, err
		}

		// delete the current distribution from the list
		r.delete(req.NamespacedName)
		return ctrl.Result{}, nil
	}

	r.reconcile(instance)

	return ctrl.Result{}, nil
}

func (r *OpenTelemetryCollectorDistributionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenTelemetryCollectorDistribution{}).
		Complete(r)
}

func (r *OpenTelemetryCollectorDistributionReconciler) LoadDistributions() error {
	if len(r.cfg.WatchedNamespaces()) == 0 {
		// get from all namespaces
		list, err := loadDistributionsWithListOptions(r.Client)
		if err != nil {
			return err
		}
		r.distributions = list
	} else {
		for _, ns := range r.cfg.WatchedNamespaces() {
			list, err := loadDistributionsWithListOptions(r.Client, client.InNamespace(ns))
			if err != nil {
				return err
			}
			r.distributions = append(r.distributions, list...)
		}
	}

	count := len(r.distributions)
	if count > 0 {
		r.notify()
	}

	r.log.V(1).Info("initial list of distributions loaded", "count", count)
	return nil
}

func loadDistributionsWithListOptions(cl client.Client, opts ...client.ListOption) ([]v1alpha1.OpenTelemetryCollectorDistribution, error) {
	list := &v1alpha1.OpenTelemetryCollectorDistributionList{}
	if err := cl.List(context.Background(), list, opts...); err != nil {
		return nil, fmt.Errorf("failed to list: %w", err)
	}

	return list.Items, nil
}

func (r *OpenTelemetryCollectorDistributionReconciler) delete(nsn types.NamespacedName) {
	for i := range r.distributions {
		d := r.distributions[i]
		if d.Namespace == nsn.Namespace && d.Name == nsn.Name {
			r.distributions = append(r.distributions[:i], r.distributions[i+1:]...)
			r.notify()
			return
		}
	}
}

func (r *OpenTelemetryCollectorDistributionReconciler) reconcile(instance v1alpha1.OpenTelemetryCollectorDistribution) {
	for i := range r.distributions {
		d := r.distributions[i]
		if d.Namespace == instance.Namespace && d.Name == instance.Name {
			// right now, we have only two fields and they are not part of the spec, so, just compare them
			// if this grows over three fields, we'll need a function that will determine whether the object has changed
			if d.Image != instance.Image || commandChanged(d.Command, instance.Command) {
				d.Command = instance.Command
				d.Image = instance.Image
				r.distributions[i] = d
				r.notify()
				return
			}

			return
		}
	}

	// the distribution wasn't found in the list of existing distributions
	r.distributions = append(r.distributions, instance)
	r.notify()
}

func (r *OpenTelemetryCollectorDistributionReconciler) notify() {
	for _, callback := range r.onChange {
		callback(r.distributions)
	}
}

func commandChanged(new, old []string) bool {
	if len(new) != len(old) {
		return true
	}

	for i := range new {
		if new[i] != old[i] {
			return true
		}
	}

	return false
}
