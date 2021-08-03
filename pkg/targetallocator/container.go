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

package targetallocator

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// Container builds a container for the given TargetAllocator.
func Container(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) corev1.Container {
	image := otelcol.Spec.TargetAllocator.Image
	if len(image) == 0 {
		image = cfg.TargetAllocatorImage()
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.TAConfigMapVolume(),
		MountPath: "/conf",
	}}

	envVars := []corev1.EnvVar{}

	envVars = append(envVars, corev1.EnvVar{
		Name: "OTELCOL_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})

	return corev1.Container{
		Name:         naming.TAContainer(),
		Image:        image,
		Env:          envVars,
		VolumeMounts: volumeMounts,
	}
}
