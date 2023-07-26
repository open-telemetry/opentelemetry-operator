package targetallocator

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil/naming"
)

func Service(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (client.Object, error) {
	name := naming.TAService(otelcol)
	labels := Labels(otelcol, name)

	selector := Labels(otelcol, name)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TAService(otelcol),
			Namespace: otelcol.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:       "targetallocation",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}, nil
}
