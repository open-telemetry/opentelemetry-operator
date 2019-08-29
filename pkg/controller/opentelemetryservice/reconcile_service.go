package opentelemetryservice

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileService reconciles the service(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryService) reconcileService(ctx context.Context) error {
	desired := service(ctx)
	r.setControllerReference(ctx, desired)

	expected := &corev1.Service{}
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

	// and finally, we should remove all services that were previously created for this instance that
	// are not in use anymore
	// TODO(jpkroehling)

	return nil
}

func service(ctx context.Context) *corev1.Service {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryService)
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
