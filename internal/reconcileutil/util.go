package reconcileutil

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

type Builder func(params Params) ([]client.Object, error)

type ObjectCreator func(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (client.Object, error)

func Conformer[T client.Object](f func(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (T, error)) ObjectCreator {
	return func(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) (client.Object, error) {
		return f(cfg, logger, otelcol)
	}
}
