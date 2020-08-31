package collector

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// ServiceAccountName returns the name of the existing or self-provisioned service account to use for the given instance
func ServiceAccountName(instance v1alpha1.OpenTelemetryCollector) string {
	if len(instance.Spec.ServiceAccount) == 0 {
		return naming.ServiceAccount(instance)
	}

	return instance.Spec.ServiceAccount
}

//ServiceAccount returns the service account for the given instance
func ServiceAccount(otelcol v1alpha1.OpenTelemetryCollector) corev1.ServiceAccount {
	labels := Labels(otelcol)
	labels["app.kubernetes.io/name"] = naming.ServiceAccount(otelcol)

	return corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.ServiceAccount(otelcol),
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
	}
}
