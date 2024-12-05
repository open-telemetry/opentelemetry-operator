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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Volumes builds the volumes for the given instance, including the config map volume.
func Volumes(cfg config.Config, otelcol v1beta1.OpenTelemetryCollector) []corev1.Volume {
	hash, _ := manifestutils.GetConfigMapSHA(otelcol.Spec.Config)
	configMapName := naming.ConfigMap(otelcol.Name, hash)
	volumes := []corev1.Volume{{
		Name: naming.ConfigMapVolume(),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
				Items: []corev1.KeyToPath{{
					Key:  cfg.CollectorConfigMapEntry(),
					Path: cfg.CollectorConfigMapEntry(),
				}},
			},
		},
	}}

	if cfg.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		volumes = append(volumes, corev1.Volume{
			Name: naming.TAClientCertificate(otelcol.Name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: naming.TAClientCertificateSecretName(otelcol.Name),
				},
			},
		})
	}

	if len(otelcol.Spec.Volumes) > 0 {
		volumes = append(volumes, otelcol.Spec.Volumes...)
	}

	if len(otelcol.Spec.ConfigMaps) > 0 {
		for keyCfgMap := range otelcol.Spec.ConfigMaps {
			volumes = append(volumes, corev1.Volume{
				Name: naming.ConfigMapExtra(otelcol.Spec.ConfigMaps[keyCfgMap].Name),
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: otelcol.Spec.ConfigMaps[keyCfgMap].Name,
						},
					},
				},
			})
		}
	}

	return volumes
}
