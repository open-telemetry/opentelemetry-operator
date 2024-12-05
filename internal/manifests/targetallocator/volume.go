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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Volumes builds the volumes for the given instance, including the config map volume.
func Volumes(cfg config.Config, instance v1alpha1.TargetAllocator) []corev1.Volume {
	volumes := []corev1.Volume{{
		Name: naming.TAConfigMapVolume(),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: naming.TAConfigMap(instance.Name)},
				Items: []corev1.KeyToPath{
					{
						Key:  cfg.TargetAllocatorConfigMapEntry(),
						Path: cfg.TargetAllocatorConfigMapEntry(),
					}},
			},
		},
	}}

	if cfg.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		volumes = append(volumes, corev1.Volume{
			Name: naming.TAServerCertificate(instance.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.TAServerCertificateSecretName(instance.Name),
				},
			},
		})
	}

	return volumes
}
