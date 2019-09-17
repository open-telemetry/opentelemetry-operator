package opentelemetrycollector

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// commonLabels return the common labels to all objects that are part of a managed OpenTelemetryCollector
func commonLabels(ctx context.Context) map[string]string {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryCollector)

	// new map every time, so that we don't touch the instance's label
	base := map[string]string{}
	if nil != instance.Labels {
		for k, v := range instance.Labels {
			base[k] = v
		}
	}

	base["app.kubernetes.io/managed-by"] = "opentelemetry-operator"
	base["app.kubernetes.io/instance"] = fmt.Sprintf("%s.%s", instance.Namespace, instance.Name)
	base["app.kubernetes.io/part-of"] = "opentelemetry"
	base["app.kubernetes.io/component"] = "opentelemetry-collector"

	return base
}
