// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// podWebhookDeploymentName is the name of the standalone pod webhook deployment managed by OLM.
	podWebhookDeploymentName = "opentelemetry-operator-pod-webhook"
	// csvDefaultReplicas is the default replica count defined in the CSV.
	csvDefaultReplicas = 2
)

// PodWebhookReconciler reconciles the standalone pod webhook deployment replicas.
// On OpenShift with OLM, the pod-webhook deployment is managed by OLM via CSV.
// This controller scales the deployment to the desired replica count (up or down)
// but refuses to scale beyond the CSV default to avoid race conditions with OLM.
type PodWebhookReconciler struct {
	client.Client
	Namespace       string
	DesiredReplicas int32
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

func (r *PodWebhookReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// Only reconcile the pod-webhook deployment in our namespace
	if req.Name != podWebhookDeploymentName || req.Namespace != r.Namespace {
		return reconcile.Result{}, nil
	}

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: podWebhookDeploymentName, Namespace: r.Namespace}, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment doesn't exist (not on OpenShift or not installed via OLM)
			return reconcile.Result{}, nil
		}
		logger.Error(err, "Failed to get pod-webhook deployment")
		return reconcile.Result{}, err
	}

	// Check current replicas
	currentReplicas := int32(1)
	if deployment.Spec.Replicas != nil {
		currentReplicas = *deployment.Spec.Replicas
	}

	// Refuse to scale beyond the CSV default to avoid race conditions with OLM.
	if r.DesiredReplicas > csvDefaultReplicas {
		logger.Info("Ignoring scale request beyond CSV default",
			"requested", r.DesiredReplicas,
			"csvDefault", csvDefaultReplicas)
		return reconcile.Result{}, nil
	}

	if currentReplicas != r.DesiredReplicas {
		logger.Info("Scaling pod-webhook deployment",
			"from", currentReplicas,
			"to", r.DesiredReplicas)

		deployment.Spec.Replicas = &r.DesiredReplicas
		if err := r.Update(ctx, deployment); err != nil {
			logger.Error(err, "Failed to scale pod-webhook deployment")
			return reconcile.Result{}, err
		}
		logger.Info("Successfully scaled pod-webhook deployment", "replicas", r.DesiredReplicas)
	}

	return reconcile.Result{}, nil
}

func (r *PodWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
