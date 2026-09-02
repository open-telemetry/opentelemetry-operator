// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// podWebhookDeploymentName is the name of the standalone pod webhook deployment managed by OLM.
	podWebhookDeploymentName = "opentelemetry-operator-webhook"
)

// CSVWebhookReconciler reconciles the ClusterServiceVersion to update pod-webhook replica count.
// On OpenShift with OLM, the pod-webhook deployment is managed by OLM via CSV.
// This controller modifies the CSV's deployment spec to change the replica count.
type CSVWebhookReconciler struct {
	client.Client
	Namespace       string
	DesiredReplicas int32
}

// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch;update;patch

func (r *CSVWebhookReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// Only reconcile CSVs in our namespace
	if req.Namespace != r.Namespace {
		return reconcile.Result{}, nil
	}

	csv := &operatorsv1alpha1.ClusterServiceVersion{}
	err := r.Get(ctx, req.NamespacedName, csv)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		logger.Error(err, "Failed to get ClusterServiceVersion")
		return reconcile.Result{}, err
	}

	// Find the pod-webhook deployment spec in the CSV
	deploymentSpecs := csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
	podWebhookIdx := -1
	for i, ds := range deploymentSpecs {
		if ds.Name == podWebhookDeploymentName {
			podWebhookIdx = i
			break
		}
	}

	if podWebhookIdx == -1 {
		// This CSV doesn't have the pod-webhook deployment
		return reconcile.Result{}, nil
	}

	currentReplicas := int32(1)
	if deploymentSpecs[podWebhookIdx].Spec.Replicas != nil {
		currentReplicas = *deploymentSpecs[podWebhookIdx].Spec.Replicas
	}

	if currentReplicas != r.DesiredReplicas {
		logger.Info("Updating pod-webhook replicas in CSV",
			"csv", csv.Name,
			"from", currentReplicas,
			"to", r.DesiredReplicas)

		// Update the replica count in the CSV
		csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs[podWebhookIdx].Spec.Replicas = &r.DesiredReplicas

		if err := r.Update(ctx, csv); err != nil {
			logger.Error(err, "Failed to update CSV")
			return reconcile.Result{}, err
		}
		logger.Info("Successfully updated pod-webhook replicas in CSV", "replicas", r.DesiredReplicas)
	}

	return reconcile.Result{}, nil
}

func (r *CSVWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorsv1alpha1.ClusterServiceVersion{}).
		Complete(r)
}
