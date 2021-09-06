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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/config"
)

// VolumeClaimTemplates builds the volumeClaimTemplates for the given instance,
// including the config map volume mount.
func VolumeClaimTemplates(cfg config.Config, otelcol v1alpha1.SplunkOtelAgent) []corev1.PersistentVolumeClaim {

	var volumeClaimTemplates []corev1.PersistentVolumeClaim

	if otelcol.Spec.Mode != "statefulset" {
		return volumeClaimTemplates
	}

	// Add all user specified claims or use default.
	if len(otelcol.Spec.VolumeClaimTemplates) > 0 {
		volumeClaimTemplates = append(volumeClaimTemplates,
			otelcol.Spec.VolumeClaimTemplates...)
	} else {
		volumeClaimTemplates = []corev1.PersistentVolumeClaim{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-volume",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{"storage": resource.MustParse("50Mi")},
				},
			},
		}}
	}

	return volumeClaimTemplates
}
