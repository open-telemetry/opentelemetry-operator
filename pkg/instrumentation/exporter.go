// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func configureExporter(exporter v1alpha1.Exporter, pod *corev1.Pod, container *corev1.Container) {
	if exporter.Endpoint != "" {
		if getIndexOfEnv(container.Env, constants.EnvOTELExporterOTLPEndpoint) == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  constants.EnvOTELExporterOTLPEndpoint,
				Value: exporter.Endpoint,
			})
		}
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
		if getIndexOfEnv(container.Env, constants.EnvOTELExporterCertificate) == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  constants.EnvOTELExporterCertificate,
				Value: envVarVal,
			})
		}
	}
	if exporter.TLS.Cert != "" {
		envVarVal := fmt.Sprintf("%s/%s", secretMountPath, exporter.TLS.Cert)
		if filepath.IsAbs(exporter.TLS.Cert) {
			envVarVal = exporter.TLS.Cert
		}
		if getIndexOfEnv(container.Env, constants.EnvOTELExporterClientCertificate) == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  constants.EnvOTELExporterClientCertificate,
				Value: envVarVal,
			})
		}
	}
	if exporter.TLS.Key != "" {
		envVarVar := fmt.Sprintf("%s/%s", secretMountPath, exporter.TLS.Key)
		if filepath.IsAbs(exporter.TLS.Key) {
			envVarVar = exporter.TLS.Key
		}
		if getIndexOfEnv(container.Env, constants.EnvOTELExporterClientKey) == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  constants.EnvOTELExporterClientKey,
				Value: envVarVar,
			})
		}
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
		for _, vol := range container.VolumeMounts {
			if vol.Name == secretVolumeName {
				addVolumeMount = false
			}
		}
		if addVolumeMount {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
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
		for _, vol := range container.VolumeMounts {
			if vol.Name == configMapVolumeName {
				addVolumeMount = false
			}
		}
		if addVolumeMount {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      configMapVolumeName,
				MountPath: configMapMountPath,
				ReadOnly:  true,
			})
		}
	}
}
