// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
