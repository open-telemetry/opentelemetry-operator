package opentelemetrycollector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileConfigMap reconciles the config map(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryCollector) reconcileConfigMap(ctx context.Context) error {
	desired := []*corev1.ConfigMap{
		configMap(ctx),
	}

	// first, handle the create/update parts
	if err := r.reconcileExpectedConfigMaps(ctx, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected deployments: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteConfigMaps(ctx, desired); err != nil {
		return fmt.Errorf("failed to reconcile the deployments to be deleted: %v", err)
	}

	return nil
}

func configMap(ctx context.Context) *corev1.ConfigMap {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	name := fmt.Sprintf("%s-collector", instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Data: map[string]string{
			opentelemetry.CollectorConfigMapEntry: instance.Spec.Config,
		},
	}
}

func (r *ReconcileOpenTelemetryCollector) reconcileExpectedConfigMaps(ctx context.Context, expected []*corev1.ConfigMap) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	for _, obj := range expected {
		desired := obj
		r.setControllerReference(ctx, desired)

		existing := &corev1.ConfigMap{}
		err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
		if err != nil && errors.IsNotFound(err) {
			if err := r.client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("configmap.name", desired.Name, "configmap.namespace", desired.Namespace).V(2).Info("created")
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %v", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.Data = desired.Data
		updated.BinaryData = desired.BinaryData
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if err := r.client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes: %v", err)
		}
		logger.V(2).Info("applied", "configmap.name", desired.Name, "configmap.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryCollector) deleteConfigMaps(ctx context.Context, expected []*corev1.ConfigMap) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	opts := client.InNamespace(instance.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	})
	list := &corev1.ConfigMapList{}
	if err := r.client.List(ctx, opts, list); err != nil {
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
			if err := r.client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
			logger.V(2).Info("deleted", "configmap.name", existing.Name, "configmap.namespace", existing.Namespace)
		}
	}

	return nil
}
