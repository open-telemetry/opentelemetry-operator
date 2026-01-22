// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifestutils

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func GetDesiredReplicas(otelCol v1beta1.OpenTelemetryCollector) *int32 {
	// If autoscaling is configured, ensure we never set replicas below MinReplicas,
	// but still respect Spec.Replicas which may be updated via the scale subresource (e.g., by HPA).
	if otelCol.Spec.Autoscaler != nil && otelCol.Spec.Autoscaler.MinReplicas != nil {
		if otelCol.Spec.Replicas != nil {
			replicas := max(*otelCol.Spec.Autoscaler.MinReplicas, *otelCol.Spec.Replicas)
			return &replicas
		}
		return otelCol.Spec.Autoscaler.MinReplicas
	}
	return otelCol.Spec.Replicas
}
