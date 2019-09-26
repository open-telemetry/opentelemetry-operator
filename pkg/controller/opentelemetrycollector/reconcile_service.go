package opentelemetrycollector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileService reconciles the service(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryCollector) reconcileService(ctx context.Context) error {
	svcs := []*corev1.Service{
		service(ctx),
		headless(ctx),
	}

	// first, handle the create/update parts
	if err := r.reconcileExpectedServices(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteServices(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %v", err)
	}

	return nil
}

func service(ctx context.Context) *corev1.Service {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	name := fmt.Sprintf("%s-collector", instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	// by coincidence, the selector is the same as the label, but note that the selector points to the deployment
	// whereas 'labels' refers to the service
	selector := labels

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports: []corev1.ServicePort{
				{
					Name:       "jaeger-grpc",
					Port:       14250,
					TargetPort: intstr.FromInt(14250),
				},
			},
		},
	}

}

func headless(ctx context.Context) *corev1.Service {
	h := service(ctx)
	h.Name = fmt.Sprintf("%s-headless", h.Name)
	h.Spec.ClusterIP = "None"
	return h
}

func (r *ReconcileOpenTelemetryCollector) reconcileExpectedServices(ctx context.Context, expected []*corev1.Service) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	for _, obj := range expected {
		desired := obj
		r.setControllerReference(ctx, desired)

		existing := &corev1.Service{}
		err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
		if err != nil && errors.IsNotFound(err) {
			if err := r.client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("service.name", desired.Name, "service.namespace", desired.Namespace).V(2).Info("created")
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

		// we keep the ClusterIP that got assigned by the cluster, if it's empty in the "desired" and not empty on the "current"
		if desired.Spec.ClusterIP == "" && len(updated.Spec.ClusterIP) > 0 {
			desired.Spec.ClusterIP = updated.Spec.ClusterIP
		}
		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if err := r.client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes to service: %v", err)
		}
		logger.V(2).Info("applied", "service.name", desired.Name, "service.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryCollector) deleteServices(ctx context.Context, expected []*corev1.Service) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	opts := client.InNamespace(instance.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	})
	list := &corev1.ServiceList{}
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
			logger.V(2).Info("deleted", "service.name", existing.Name, "service.namespace", existing.Namespace)
		}
	}

	return nil
}
