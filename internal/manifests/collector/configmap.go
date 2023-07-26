package collector

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func ConfigMap(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (client.Object, error) {
	return DesiredConfigMap(cfg, logger, otelcol), nil
}

func DesiredConfigMap(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) *corev1.ConfigMap {
	name := naming.ConfigMap(otelcol)
	labels := Labels(otelcol, name, []string{})

	replacedConf, err := ReplaceConfig(otelcol)
	if err != nil {
		logger.V(2).Info("failed to update prometheus config to use sharded targets: ", "err", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
		Data: map[string]string{
			"collector.yaml": replacedConf,
		},
	}
}
