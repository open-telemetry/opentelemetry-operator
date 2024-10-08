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

package instrumentation

import (
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func configureExporter(exporter v1alpha1.Exporter, pod *corev1.Pod, container Container) {
	if exporter.Endpoint != "" {
		container.appendIfNotExists(pod, constants.EnvOTELExporterOTLPEndpoint, exporter.Endpoint)
	}
	if exporter.TLS == nil {
		return
	}
	// the name cannot be longer than 63 characters
	secretVolumeName := naming.Truncate("otel-auto-secret-%s", 63, exporter.TLS.SecretName)
	secretMountPath := fmt.Sprintf("/otel-auto-instrumentation-secret-%s", exporter.TLS.SecretName)
	configMapVolumeName := naming.Truncate("otel-auto-configmap-%s", 63, exporter.TLS.ConfigMapName)
	configMapMountPath := fmt.Sprintf("/otel-auto-instrumentation-configmap-%s", exporter.TLS.ConfigMapName)

	if exporter.TLS.CA != "" {
		mountPath := secretMountPath
		if exporter.TLS.ConfigMapName != "" {
			mountPath = configMapMountPath
		}
		envVarVal := fmt.Sprintf("%s/%s", mountPath, exporter.TLS.CA)
		if filepath.IsAbs(exporter.TLS.CA) {
			envVarVal = exporter.TLS.CA
		}
		container.appendIfNotExists(pod, constants.EnvOTELExporterCertificate, envVarVal)
	}
	if exporter.TLS.Cert != "" {
		envVarVal := fmt.Sprintf("%s/%s", secretMountPath, exporter.TLS.Cert)
		if filepath.IsAbs(exporter.TLS.Cert) {
			envVarVal = exporter.TLS.Cert
		}
		container.appendIfNotExists(pod, constants.EnvOTELExporterClientCertificate, envVarVal)
	}
	if exporter.TLS.Key != "" {
		envVarVar := fmt.Sprintf("%s/%s", secretMountPath, exporter.TLS.Key)
		if filepath.IsAbs(exporter.TLS.Key) {
			envVarVar = exporter.TLS.Key
		}
		container.appendIfNotExists(pod, constants.EnvOTELExporterClientKey, envVarVar)
	}

	if exporter.TLS.SecretName != "" {
		addVolume := true
		for _, vol := range pod.Spec.Volumes {
			if vol.Name == secretVolumeName {
				addVolume = false
			}
		}
		if addVolume {
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
				Name: secretVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: exporter.TLS.SecretName,
					},
				}})
		}
		addVolumeMount := true
		for _, vol := range pod.Spec.Containers[container.index].VolumeMounts {
			if vol.Name == secretVolumeName {
				addVolumeMount = false
			}
		}
		if addVolumeMount {
			pod.Spec.Containers[container.index].VolumeMounts = append(pod.Spec.Containers[container.index].VolumeMounts, corev1.VolumeMount{
				Name:      secretVolumeName,
				MountPath: secretMountPath,
				ReadOnly:  true,
			})
		}
	}

	if exporter.TLS.ConfigMapName != "" {
		addVolume := true
		for _, vol := range pod.Spec.Volumes {
			if vol.Name == configMapVolumeName {
				addVolume = false
			}
		}
		if addVolume {
			pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
				Name: configMapVolumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: exporter.TLS.ConfigMapName,
						},
					},
				}})
		}
		addVolumeMount := true
		for _, vol := range pod.Spec.Containers[container.index].VolumeMounts {
			if vol.Name == configMapVolumeName {
				addVolumeMount = false
			}
		}
		if addVolumeMount {
			pod.Spec.Containers[container.index].VolumeMounts = append(pod.Spec.Containers[container.index].VolumeMounts, corev1.VolumeMount{
				Name:      configMapVolumeName,
				MountPath: configMapMountPath,
				ReadOnly:  true,
			})
		}
	}
}
