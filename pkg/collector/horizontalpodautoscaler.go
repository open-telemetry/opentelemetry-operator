package collector

import (
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HorizontalPodAutoscaler(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) autoscalingv1.HorizontalPodAutoscaler {
	labels := Labels(otelcol)
	labels["app.kubernetes.io/name"] = naming.Collector(otelcol)

	annotations := Annotations(otelcol)
	var cpuTarget int32 = 90

	return autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Collector(otelcol),
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       naming.Collector(otelcol),
			},
			MinReplicas:                    otelcol.Spec.MinReplicas,
			MaxReplicas:                    *otelcol.Spec.MaxReplicas,
			TargetCPUUtilizationPercentage: &cpuTarget,
		},
	}
}
