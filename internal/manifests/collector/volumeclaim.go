// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package collector handles the OpenTelemetry Collector.
package collector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// VolumeClaimTemplates builds the volumeClaimTemplates for the given instance,
// including the config map volume mount.
func VolumeClaimTemplates(otelcol v1beta1.OpenTelemetryCollector) []corev1.PersistentVolumeClaim {
	if otelcol.Spec.Mode != "statefulset" {
		return []corev1.PersistentVolumeClaim{}
	}

	// Add all user specified claims.
	return otelcol.Spec.VolumeClaimTemplates
}
