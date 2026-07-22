// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
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
						Key:  cfg.TargetAllocatorConfigMapEntry,
						Path: cfg.TargetAllocatorConfigMapEntry,
					},
				},
			},
		},
	}}

	if manifestutils.IsTAMTLSEnabled(&instance) {
		serverVolumes, _ := manifestutils.TAServerCertificateVolumes(&instance)
		volumes = append(volumes, serverVolumes...)
	}

	volumes = append(volumes, instance.Spec.Volumes...)

	return volumes
}
