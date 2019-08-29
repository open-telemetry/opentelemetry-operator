package opentelemetryservice

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileDeployment reconciles the deployment(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryService) reconcileConfigMap(ctx context.Context) error {
	desired := configMap(ctx)
	r.setControllerReference(ctx, desired)

	expected := &corev1.ConfigMap{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, expected)
	if err != nil && errors.IsNotFound(err) {
		if err := r.client.Create(ctx, desired); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// it exists already, merge the two if the end result isn't identical to the existing one
	// TODO(jpkroehling)

	return nil
}

func configMap(ctx context.Context) *corev1.ConfigMap {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryService)
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
