package opentelemetryservice

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// commonLabels return the common labels to all objects that are part of a managed OpenTelemetryService
func commonLabels(ctx context.Context) map[string]string {
	instance := ctx.Value(opentelemetry.Instance).(*v1alpha1.OpenTelemetryService)
	base := instance.Labels
	if nil == base {
		base = map[string]string{}
	}

	base["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	base["app.kubernetes.io/instance"] = fmt.Sprintf("%s.%s", instance.Namespace, instance.Name)
	base["app.kubernetes.io/part-of"] = "opentelemetry"
	base["app.kubernetes.io/component"] = "opentelemetry-service"

	return base
}
