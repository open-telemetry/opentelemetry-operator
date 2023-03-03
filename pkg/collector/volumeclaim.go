// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package collector handles the OpenTelemetry Collector.
package collector

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// VolumeClaimTemplates builds the volumeClaimTemplates for the given instance,
// including the config map volume mount.
func VolumeClaimTemplates(otelcol v1alpha1.OpenTelemetryCollector) []corev1.PersistentVolumeClaim {
	if otelcol.Spec.Mode != "statefulset" {
		return []corev1.PersistentVolumeClaim{}
	}

	// Add all user specified claims.
	return otelcol.Spec.VolumeClaimTemplates
}
