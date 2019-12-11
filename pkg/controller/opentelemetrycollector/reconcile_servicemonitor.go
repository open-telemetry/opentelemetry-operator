package opentelemetrycollector

import (
	"context"
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileServiceMonitor reconciles the service monitor(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryCollector) reconcileServiceMonitor(ctx context.Context) error {
	if !viper.GetBool(opentelemetry.SvcMonitorAvailable) {
		logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
		logger.V(2).Info("skipping reconciliation for service monitor, as the CRD isn't registered with the cluster")
		return nil
	}

	svcs := []*monitoringv1.ServiceMonitor{
		serviceMonitor(ctx),
	}

	// first, handle the create/update parts
	if err := r.reconcileExpectedServiceMonitors(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the expected service monitors: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteServiceMonitors(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the service monitors to be deleted: %v", err)
	}

	return nil
}

func serviceMonitor(ctx context.Context) *monitoringv1.ServiceMonitor {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	name := resourceName(instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	selector := commonLabels(ctx)
	selector["app.kubernetes.io/name"] = fmt.Sprintf("%s-monitoring", name)

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selector,
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port: "monitoring",
			}},
		},
	}

}

func (r *ReconcileOpenTelemetryCollector) reconcileExpectedServiceMonitors(ctx context.Context, expected []*monitoringv1.ServiceMonitor) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	for _, obj := range expected {
		desired := obj
		r.setControllerReference(ctx, desired)

		existing, err := r.clients.monclient.ServiceMonitors(desired.Namespace).Get(desired.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			if desired, err = r.clients.monclient.ServiceMonitors(desired.Namespace).Create(desired); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("svcmon.name", desired.Name, "svcmon.namespace", desired.Namespace).V(2).Info("created")
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

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if _, err = r.clients.monclient.ServiceMonitors(desired.Namespace).Update(updated); err != nil {
			return fmt.Errorf("failed to apply changes to service monitor: %v", err)
		}
		logger.V(2).Info("applied", "svcmon.name", desired.Name, "svcmon.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryCollector) deleteServiceMonitors(ctx context.Context, expected []*monitoringv1.ServiceMonitor) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}).String(),
	}

	list, err := r.clients.monclient.ServiceMonitors(instance.Namespace).List(opts)
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
			if err := r.clients.monclient.ServiceMonitors(instance.Namespace).Delete(existing.Name, &metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
			logger.V(2).Info("deleted", "svcmon.name", existing.Name, "svcmon.namespace", existing.Namespace)
		}
	}

	return nil
}
