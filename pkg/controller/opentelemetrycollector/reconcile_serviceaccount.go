package opentelemetrycollector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileServiceAccount reconciles the service(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryCollector) reconcileServiceAccount(ctx context.Context) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)

	svcs := []*corev1.ServiceAccount{}
	if len(instance.Spec.ServiceAccount) == 0 {
		// if there's no Service Account specified we create one and manage it ourselves
		svcs = append(svcs, serviceAccount(ctx))
	}

	// first, handle the create/update parts
	if err := r.reconcileExpectedServiceAccounts(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the expected service accounts: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteServiceAccounts(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the service accounts to be deleted: %v", err)
	}

	return nil
}

func serviceAccount(ctx context.Context) *corev1.ServiceAccount {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	name := resourceName(instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
	}
}

func (r *ReconcileOpenTelemetryCollector) reconcileExpectedServiceAccounts(ctx context.Context, expected []*corev1.ServiceAccount) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	for _, obj := range expected {
		desired := obj

		// #nosec G104 (CWE-703): Errors unhandled.
		r.setControllerReference(ctx, desired)

		svcs := r.clientset.Kubernetes.CoreV1().ServiceAccounts(desired.Namespace)

		existing, err := svcs.Get(ctx, desired.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			if desired, err = svcs.Create(ctx, desired, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("serviceAccount.name", desired.Name, "serviceAccount.namespace", desired.Namespace).V(2).Info("created")
			continue
		} else if err != nil {
			return fmt.Errorf("failed to retrieve: %v", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if updated, err = svcs.Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to apply changes to service account: %v", err)
		}
		logger.V(2).Info("applied", "serviceAccount.name", desired.Name, "serviceAccount.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryCollector) deleteServiceAccounts(ctx context.Context, expected []*corev1.ServiceAccount) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	svcs := r.clientset.Kubernetes.CoreV1().ServiceAccounts(instance.Namespace)

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}).String(),
	}
	list, err := svcs.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list: %v", err)
	}

	for _, existing := range list.Items {
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := svcs.Delete(ctx, existing.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
			logger.V(2).Info("deleted", "serviceAccount.name", existing.Name, "serviceAccount.namespace", existing.Namespace)
		}
	}

	return nil
}

// ServiceAccountNameFor returns the name of the service account for the given context
func ServiceAccountNameFor(ctx context.Context) string {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	if len(instance.Spec.ServiceAccount) == 0 {
		return serviceAccount(ctx).Name
	}

	return instance.Spec.ServiceAccount
}
